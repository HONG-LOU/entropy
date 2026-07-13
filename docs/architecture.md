# Entropy MVP technical design

## Decision summary

Entropy is one Go process with three delivery surfaces:

```text
Wails desktop UI
       |
       v
internal/node     lifecycle, mining jobs, peer HTTP, chain selection
       |
       +---- internal/core    pure wallet and consensus rules
       |
       +---- internal/store   atomic local persistence

cmd/entropy       headless entry point using the same node service
```

The desktop app never receives private key material. UI calls cross the Wails
binding into the Go node service, which signs locally.

## Consensus objects

A transaction contains a random nonce, inputs, and outputs. Each input points
to a previous transaction output and carries a SEC1 public key plus ASN.1 DER
ECDSA signature. Its signing hash uses deterministic big-endian encoding and
binds:

- every input outpoint and public key;
- every output amount and address;
- the input index being signed;
- the amount and address of that input's previous output.

Transaction IDs hash the complete signed transaction. An address is
`ent1 || hex(SHA-256(pubkey)[0:20] || checksum[0:4])`.

A block header binds version, height, timestamp, previous hash, Merkle root,
difficulty, and nonce. Proof of work requires the SHA-256 header hash to have
at least `difficulty` leading zero bits. Fork choice compares cumulative work,
where a block contributes `2^difficulty`; height alone never decides the tip.
Block timestamps must exceed the median of the previous 11 blocks and may be no
more than 120 seconds ahead of local time. Difficulty first adjusts at height
120 and then every 60 blocks.

## Validation order

On startup and before persistence, the node replays the complete chain from the
fixed genesis block and rebuilds UTXO state. It rejects:

1. Wrong chain identity, genesis block, height, previous hash, or difficulty.
2. Wrong Merkle root, header hash, or proof of work.
3. Missing, repeated, or already-spent transaction inputs.
4. Public keys that do not own the referenced output.
5. Invalid signatures, transaction IDs, addresses, or zero-valued outputs.
6. Output totals above input totals or overflowing integer arithmetic.
7. Duplicate coinbase transactions or a reward unequal to subsidy plus fees.
8. Transactions or blocks above deterministic consensus resource limits.

Derived UTXO data is not persisted, avoiding two competing sources of truth.

## Issuance

Consensus uses height, never local wall-clock time, to choose the subsidy:

```text
N    = 10 * 365 * 24 * 60 * 60 / 10 = 31,536,000 blocks
MAX  = 2,000,000 * 100,000,000       = 200,000,000,000,000 units
BASE = floor(MAX / N)                 = 6,341,958 units
REM  = MAX mod N                      = 12,512,000 units

subsidy(h) = 0                    when h == 0 or h > N
             BASE + 1             when 1 <= h <= REM
             BASE                 when REM < h <= N
```

Fees are existing value moved into the coinbase and do not increase supply.

## Node lifecycle

On desktop startup:

1. Load or generate the local wallet.
2. Load, replay, and validate the chain file.
3. Start the peer listener on `0.0.0.0:47821`.
4. Poll configured peers every two seconds.
5. Accept and relay valid pending transactions immediately.

Mining remains opt-in. A mining worker operates on a snapshot. If the tip or
pending revision changes before proof of work completes, the stale candidate is
discarded and work restarts. This prevents a newly received payment from being
silently removed when a local block is committed.

On shutdown the app cancels mining, stops peer synchronization, shuts down the
HTTP listener, and waits for workers before the process exits.

## Peer protocol

The MVP protocol has four endpoints:

```text
GET  /v1/status         chain identity, height, tip, cumulative work
GET  /v1/state          replayable chain and pending transactions
POST /v1/state          candidate chain; adopt only if valid and more work
POST /v1/transactions   validate, persist, and relay a signed transaction
```

Request bodies are size-limited. Chain adoption is persisted atomically. Local
pending transactions are revalidated and retained when possible after a reorg.

This full-state synchronization is deliberately small in code, not bandwidth.
The production protocol must exchange headers and missing blocks incrementally.

## Persistence

Files live under `%APPDATA%\Entropy` by default:

```text
wallet.json   private/public key and address, mode 0600 where supported
chain.json    blocks plus pending transactions
peers.json    configured peer URLs
node.lock     operating-system lock preventing concurrent writers
```

Writes use a same-directory temporary file, flush it, close it, and atomically
rename it over the destination. Invalid or truncated state is rejected rather
than silently reset.

## Acceptance criteria

The MVP is complete when all of these pass:

- A fresh desktop launch creates one wallet and starts a listening node.
- A miner receives the height-specific subsidy.
- Two local nodes synchronize a mined chain by cumulative work.
- A signed transfer appears as pending on the peer and confirms in a block.
- Tampered amounts, signatures, hashes, and corrupt persistence are rejected.
- Restarting produces the same wallet, tip, balances, and pending set.
- The exact terminal subsidy height sums to 2,000,000 ENT.
- A Windows amd64 single-file executable builds and starts successfully.
