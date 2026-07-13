# Entropy v0.2 architecture

## Scope

Entropy v0.2 is a public-testnet implementation, not an audited mainnet. It is
designed so one Windows application can be a wallet, a full validator, a peer,
and an optional proof-of-work miner without an external database service.

The phrase "full node" means that the node independently verifies consensus
rules before changing its active ledger. It does not mean the implementation
has been independently audited or is safe for real-value assets.

## Process layout

The desktop and CLI use the same Go service and consensus code:

```text
Wails desktop UI                     cmd/entropy CLI
        |                                  |
        +----------------+-----------------+
                         |
                  internal/node
          lifecycle, mining, peers, sync,
           HTTP/WebSocket, LAN discovery
                  /       |       \
                 /        |        \
        internal/core  internal/ledger  internal/vault
        consensus and  SQLite ledger,   DPAPI wallet,
        cryptography    UTXO and undo    backup/recovery
                            |
                     internal/store
                 lock and legacy migration
```

The browser UI never receives the private key. Wails calls enter Go, and the Go
node signs locally. The public P2P listener exposes chain and mempool messages,
not wallet-control methods.

## Consensus identity and objects

The network identity is `entropy-testnet-v3`. Nodes reject a peer status,
discovery message, or WebSocket message with another protocol identity.

A regular transaction contains a random nonce, inputs, and outputs. Each input
references a previous output and carries a SEC1 P-256 public key and an ASN.1
DER ECDSA signature. Signatures use low-S normalization. The signing digest
uses deterministic big-endian encoding and binds:

- every input outpoint and public key;
- every output address and amount;
- the index of the input being signed;
- the amount and address of that input's referenced output.

The transaction ID hashes the complete signed transaction. An address is a
checksummed `ent1...` encoding derived from the first 20 bytes of the public
key's SHA-256 hash plus a four-byte checksum.

A block header binds version, height, timestamp, previous hash, Merkle root,
difficulty, nonce, and resulting hash. Proof of work requires the SHA-256 header
hash to have at least `difficulty` leading zero bits. A block contributes
`2^difficulty` work. Fork choice compares cumulative work, never height alone.

The timestamp must be greater than median-time-past over the previous 11 blocks
and no more than 120 seconds ahead of local time. Difficulty begins at 22
leading zero bits, first adjusts at height 120, and then adjusts every 60
blocks. This compact DAA is part of the testnet protocol and has not had the
long-duration or adversarial review expected of a production monetary network.

Deterministic resource limits include a 1 MiB block, 64 KiB transaction, 2,000
transactions per block, 256 inputs and outputs per transaction, and 5,000
pending transactions. Local relay policy further caps the mempool at 32 MiB
and 20,000 total inputs and requires at least 1,000 atomic units (0.00001000
ENT) of fee per started KiB. Even a maximum-size transaction remains below the
desktop's default 0.001 ENT fee. Miners prioritize fee rate while preserving
pending parent-before-child dependencies. Those mempool limits and ordering
rules are policy, not block-consensus rules.

## Issuance and coinbase maturity

Consensus selects subsidy by height rather than local time:

```text
N    = 10 * 365 * 24 * 60 * 60 / 10 = 31,536,000 blocks
MAX  = 2,000,000 * 100,000,000       = 200,000,000,000,000 units
BASE = floor(MAX / N)                 = 6,341,958 units
REM  = MAX mod N                      = 12,512,000 units

subsidy(h) = 0                    when h == 0 or h > N
             BASE + 1             when 1 <= h <= REM
             BASE                 when REM < h <= N
```

The fixed genesis block has no reward and there is no premine. Fees are
existing value transferred into coinbase and cannot increase total issuance.
The terminal reward height sums exactly to 2,000,000 ENT.

Coinbase maturity is 100 blocks and activates at spending height 100. Before
that activation height, the v0.1 testnet rules remain replayable. At and after
height 100, spending a coinbase requires:

```text
spending_height - coinbase_height >= 100
```

For example, the height-1 coinbase is rejected at spending height 100 and
accepted at height 101. A coinbase created at height 100 becomes spendable at
height 200.

## Validation pipeline

The ledger validates before committing. For a new block the important order is:

1. Check chain identity, expected height, previous hash, expected difficulty,
   timestamp, Merkle root, header hash, proof of work, and size limits.
2. Begin one SQLite write transaction and read the active UTXO records needed
   by each regular transaction.
3. Reject missing, duplicate, already spent, immature, or conflicting inputs.
4. Verify ownership, signature, transaction ID, address, positive outputs,
   integer overflow, and `outputs <= inputs`.
5. Accumulate fees and verify exactly one coinbase paying subsidy plus fees.
6. Remove spent UTXOs, insert created UTXOs, indexes, bodies, cumulative work,
   and an undo record.
7. Remove confirmed mempool entries, revalidate remaining entries, and commit.

Any failure rolls back the whole SQLite transaction. Mining and P2P use this
same commit path; neither can bypass validation.

## SQLite ledger

The ledger is `%LOCALAPPDATA%\Entropy\entropy.db` for a clean install and uses
the pure-Go `modernc.org/sqlite` driver. Connections set:

```text
journal_mode = WAL
synchronous  = FULL
foreign_keys = ON
busy_timeout = 5000 ms
trusted_schema = OFF
```

Startup creates or migrates the schema and verifies the fixed genesis block.
It runs `PRAGMA quick_check` on first or unclean startup; clean shutdown records
a checkpointed session marker so routine starts do not rescan a ten-year file.
The database stores:

- active-chain headers, bodies, encoded size, and cumulative work;
- transaction bodies and per-address sent/received indexes;
- the current UTXO set with creation height and coinbase flag;
- one undo record for every retained non-genesis block;
- the validated mempool and its spent/created output indexes;
- configured and discovered peers, retry state, and errors;
- health events and storage-policy metadata.

Consensus work and nonces use explicit big-endian byte encodings where SQLite's
signed integer range is unsuitable. Amount validation prevents values above the
signed range from entering SQL columns.

One operating-system `node.lock` protects each data directory. A desktop app,
CLI command, or second process cannot concurrently write the same ledger.
Clean shutdown checkpoints and truncates the WAL before closing.

## Reorganization and undo

Each connected block records both the UTXOs it consumed and the outpoints it
created. A higher-work replacement is applied in one database transaction:

```text
remote block locator
        |
        v
find common active-chain ancestor
        |
        v
validate candidate headers and cumulative work
        |
        v
fetch missing bodies in bounded batches
        |
        v
BEGIN SQLite transaction
  disconnect old tip to ancestor using undo records
  connect candidate blocks in forward order
  require candidate cumulative work > old cumulative work
  revalidate orphaned and existing mempool transactions
COMMIT, or ROLLBACK everything
```

Transactions orphaned from the old branch return to the mempool only if they
remain valid against the replacement UTXO set. The candidate never becomes
visible as a partially applied chain.

## Mining consistency

A mining job snapshots the expected tip, pending transactions, fee total,
timestamp rule, and target difficulty. CPU workers search nonces concurrently.
Receiving or committing a new tip cancels every job built on the old tip so
continuous mining immediately builds a fresh template. After proof of work is
found, `CommitMinedBlock` still requires the active tip to equal the snapshot
tip and then runs normal block validation.

Mining is always opt-in. An outbound-only node validates and relays without
mining, and a miner is not a network coordinator.

## Incremental peer synchronization

Protocol v3 separates catch-up from real-time relay:

- HTTP provides status, block locators, header batches, requested body batches,
  bounded mempool catch-up, and a fallback transaction/block submission path.
- WebSocket `/v2/p2p` carries hello/status, transaction, block, ping, and pong
  messages without waiting for the next sync poll.
- LAN multicast announces the node ID and TCP listen port on
  `239.255.78.21:47822/UDP`.

The synchronizer requests headers first, validates them locally, computes the
candidate work, and downloads bodies only for a chain that can beat local work.
Bodies must match the requested headers and are fully validated during the
atomic reorg. The removed v0.1 `/v1/state` endpoint cannot replace local state.

Peer failures persist in SQLite. Retry starts at one second, doubles after each
failure, and caps at five minutes. Success resets the failure count. Global and
per-IP request/WebSocket limits, cross-reconnect invalid-message scoring,
strict JSON decoding, bounded response/staging bytes, timeouts, and a 64-peer
cap bound common resource-exhaustion paths. These controls reduce risk but are
not a substitute for hostile-network auditing.

See [Protocol v3](protocol.md) for endpoint and message details.

## Archive and pruned nodes

Both modes retain all headers, cumulative work, the current UTXO set, address
indexes, mempool, and peers. The difference is historical body and undo
retention.

An archive node retains all block and transaction bodies and all undo records.
It can serve a new peer from genesis and can reorganize anywhere in its stored
history.

A pruned node permanently clears old block and transaction bodies and deletes
old undo records while retaining the most recent configured window. The prune
depth is persisted in SQLite: `0` means archive for future blocks, while
`120..31,536,000` is the retained complete-body horizon. Deleted data does not
reappear merely by switching to archive mode.

A pruned node continues to validate and relay all new data, but it returns HTTP
`410 Gone` for deleted bodies and rejects a reorg crossing its prune horizon.
It must resynchronize from an archive peer to recover from such a deep fork or
to regain historical bodies. Public bootstrap nodes should therefore be
archives.

## Wallet vault and recovery

New wallets use 256 bits of entropy encoded as 24 English BIP39 words. Entropy
then deterministically derives a P-256 private scalar using the versioned
`entropy-p256-bip39-hmac-sha256-v1` derivation. This is not BIP32, SLIP-0010, or
a Bitcoin wallet path; importing the phrase into unrelated wallet software
will not produce the Entropy address.

The local `wallet.vault` is encrypted by Windows user-scope DPAPI and opens
automatically for the same Windows account on the same installation. It is not
a portable backup. A `.entwallet` backup uses:

```text
KDF       Argon2id v1.3, time=3, memory=64 MiB, threads=2
Cipher    XChaCha20-Poly1305
Password  12 to 1024 UTF-8 bytes
```

Envelope metadata is authenticated, imports reject hostile KDF parameters
before allocating memory, and one process permits only one password KDF at a
time. Local and portable writes use temporary files, flush, and atomic replace
semantics. A corrupt or missing vault never causes silent wallet regeneration.

Restoring a wallet replaces the active address and is blocked while mining.
The ledger remains independent and can be resynchronized after wallet restore.

## Legacy migration

Migration is intentionally fail-closed:

- A valid v0.1 `chain.json` imports into an empty SQLite ledger. The imported
  tip must exactly match the legacy tip before the file is renamed
  `chain.json.migrated.bak`.
- `peers.json` imports into SQLite before becoming `peers.json.migrated.bak`.
- If SQLite and legacy chain tips disagree, neither copy is replaced.
- A plaintext `wallet.json` prevents normal startup. Migration must create and
  reopen both a DPAPI `wallet.vault` and a password-encrypted `.entwallet`
  backup with the same address. Only then is plaintext removed.

Legacy P-256 wallet keys are preserved exactly. They have no BIP39 phrase, so
their encrypted portable backup is the only application-level recovery copy.

## Security boundary

The implementation verifies signatures, proof of work, monetary policy,
coinbase maturity, timestamps, size limits, UTXO ownership, and cumulative-work
fork choice. Private keys stay outside P2P and frontend bindings.

It does not yet provide independent audit evidence, authenticated peers,
built-in transport encryption, public seed infrastructure, NAT traversal,
signed binaries, reproducible builds, or a mature protocol-upgrade process.
P-256 and the compact fast-chain DAA are testnet design choices, not claims of
Bitcoin compatibility. Read [SECURITY.md](../SECURITY.md) before exposing a
node or handling wallet material.
