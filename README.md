# Entropy (ENT)

Entropy v0.2 is a compact proof-of-work **public testnet** packaged as a
Windows desktop full node. Starting one application starts the wallet, SQLite
ledger, block and transaction validator, peer synchronization, relay server,
and optional miner in the same process. It does not require a separate database
server, browser tab, or background daemon.

> Entropy has not received an independent security or consensus audit. ENT is
> testnet currency only. Do not buy it, sell it, promise redemption, or use it
> to hold anything with real-world value. This repository does not claim
> mainnet or production-security readiness.

Repository: <https://github.com/HONG-LOU/entropy>

## What v0.2 includes

- A Wails desktop node with send, receive, mining, peer, history, wallet
  recovery, database, and pruning controls.
- A UTXO ledger in SQLite with WAL, `synchronous=FULL`, indexed balances and
  history, persistent mempool and peers, per-block undo records, atomic reorgs,
  and startup integrity checks.
- Header-first incremental synchronization over bounded HTTP batches. The old
  whole-chain `/v1/state` exchange is removed.
- WebSocket transaction and block broadcast for low-latency relay, with HTTP
  broadcast as a fallback delivery path.
- Automatic LAN discovery plus persistent manual peers, connection limits,
  and exponential retry backoff.
- A Windows DPAPI-protected local wallet, 24-word BIP39 recovery for newly
  created wallets, and portable Argon2id/XChaCha20-Poly1305 `.entwallet`
  backups.
- Archive and pruned storage modes. Both validate incoming blocks locally;
  archive nodes additionally keep and serve all historical bodies.
- Automatic, verified migration of v0.1 `chain.json` and `peers.json`, plus a
  mandatory encrypted migration path for the old plaintext `wallet.json`.
- A headless CLI using the same node, consensus, wallet, ledger, and P2P code.
- A portable desktop executable and an NSIS per-user installer build.

## How many nodes are required?

One node is enough to create a wallet, mine, send transactions to itself, and
validate its local chain. Two nodes are the practical minimum for a network:
they can relay a payment, synchronize blocks, and resolve a fork by cumulative
work. There is no coordinator and no fixed quorum.

For a useful public testnet, run several independently operated nodes. At least
one must be reachable from the internet so outbound-only nodes behind NAT have
somewhere to connect. A node behind NAT is still a full validating and relaying
node; it simply cannot accept unsolicited inbound connections.

## Run the Windows desktop node

Use an artifact from the
[GitHub Releases page](https://github.com/HONG-LOU/entropy/releases), or run the
portable executable produced locally at:

```text
D:\Entropy\build\bin\Entropy.exe
```

The NSIS build is the `*installer*.exe` artifact in the same directory. The
installer is the simplest distribution for other Windows users; the portable
EXE can be launched directly. Windows 10/11 x64 and Microsoft WebView2 Runtime
are required. The installer build downloads the WebView2 bootstrapper when
needed. Releases are currently unsigned, so Windows SmartScreen may show an
unknown-publisher warning.

On first launch the application:

1. Creates a new P-256 wallet and protects it with Windows user-scope DPAPI.
2. Creates and verifies the SQLite ledger.
3. Listens on `0.0.0.0:47821` for peers, or chooses a free port if another
   program already owns the default desktop port.
4. Announces itself on the local network.
5. Synchronizes configured and discovered peers.

Mining is off by default. Opening the application always runs a validating
node; mining begins only after **Start mining** or **Mine one block**.

The default data directory for a clean installation is:

```text
%LOCALAPPDATA%\Entropy
```

The per-user uninstaller removes the application but deliberately keeps this
directory, including the wallet and chain. Back it up before deleting it
manually. An existing v0.1/v0.2 data directory under `%APPDATA%\Entropy` is
detected and reused so an upgrade never silently switches wallets. Do not
allow roaming-profile software to merge that live SQLite database between PCs.

The wallet screen will ask you to record the 24-word phrase or export an
encrypted `.entwallet` backup. Do that before mining or receiving funds. The
chain database can be downloaded again; the wallet cannot.

See [Operations](docs/operations.md) for multi-node examples, firewall and NAT
setup, backups, migration, pruning, and troubleshooting.

## Transfer and confirmation speed

A valid transfer enters the sender's local mempool immediately and is broadcast
to connected peers over WebSocket. A healthy LAN peer will usually see it in
well under one second, but relay is not confirmation. Consensus targets one
proof-of-work block every 10 seconds:

```text
Mempool relay          usually below 1 second on a healthy LAN
1 confirmation         target about 10 seconds
6 confirmations        target about 1 minute
```

Actual time depends on miners, hash rate, peer connectivity, and forks. A
10-second testnet has a higher stale-block risk than Bitcoin, and six ENT
confirmations do not provide Bitcoin-level economic security.

## Monetary and consensus parameters

Entropy has no premine. Height zero is a fixed reward-free genesis block.
Amounts use eight decimal places and integer arithmetic.

```text
Network identity                  entropy-testnet-v3
Maximum supply                    2,000,000.00000000 ENT
Atomic units per ENT              100,000,000
Target block spacing              10 seconds
Reward-bearing heights            1 through 31,536,000
Target emission duration          3,650 days (about 10 years)
Heights 1..12,512,000             0.06341959 ENT
Heights 12,512,001..31,536,000    0.06341958 ENT
Later heights                     transaction fees only
Coinbase maturity                 100 blocks, enforced from height 100
Maximum block body                1 MiB
Fork choice                       greatest cumulative proof of work
```

The one-atomic-unit reward difference distributes the integer division
remainder. The reward schedule sums to exactly `200,000,000,000,000` atomic
units at height 31,536,000. Fees move existing ENT and do not increase supply.

"Ten years" means 31,536,000 target intervals of 10 seconds, not a wall-clock
guarantee. Beginning at spending height 100, a coinbase output must be 100
blocks old. The activation boundary preserves replay compatibility with the
published v0.1 testnet history.

## Run the CLI

Build the headless executable or use `go run`:

```powershell
go build -o .\build\bin\entropy-cli.exe .\cmd\entropy

# Archive node, automatic LAN discovery enabled
.\build\bin\entropy-cli.exe node --data .\data\node-a --listen 0.0.0.0:47821 --mine

# Second node on the same computer
.\build\bin\entropy-cli.exe node --data .\data\node-b --listen 127.0.0.1:47822 `
  --peer http://127.0.0.1:47821

# Pruned node retaining the newest 20,000 complete block bodies
.\build\bin\entropy-cli.exe node --data .\data\node-c --listen 0.0.0.0:47823 `
  --peer http://127.0.0.1:47821 --prune-depth 20000
```

Available commands are:

```text
node            run a node; optionally mine, prune, add a peer, or disable LAN discovery
status          print wallet, height, peer, and issuance status
mine-one        mine exactly one block
history         print wallet transaction history
wallet-backup   create a password-encrypted portable backup
wallet-migrate  migrate a v0.1 plaintext wallet before startup
```

Commands that open a data directory take its exclusive `node.lock`. Stop the
desktop app or running CLI node before using another command against the same
directory. Full command examples are in [Operations](docs/operations.md#cli-reference).

## Public connectivity

- Allow inbound TCP `47821` in Windows Firewall for the desktop node.
- On a home router, forward external TCP `47821` to the node computer's TCP
  `47821` if you want to accept internet peers.
- Do not forward UDP `47822`; it is local multicast discovery only.
- Give peers a reachable base URL such as `http://203.0.113.20:47821`.
- If your ISP uses carrier-grade NAT, port forwarding will not make the node
  reachable. You can still connect outbound to a public node.

Entropy v0.2 does not include public seed infrastructure or automatic NAT
traversal. Manual public peers are therefore required outside the LAN. P2P is
plain HTTP/WebSocket unless the operator supplies a TLS reverse proxy; it has
no peer authentication.

## Storage modes

Archive mode is the default (`--prune-depth 0`). It keeps headers, bodies,
transaction data, indexes, UTXOs, and undo records, so it can serve historical
blocks and handle any reorg available in its local history.

Pruned mode retains every header, the current UTXO set, transaction/address
indexes, mempool, and peer records, but permanently deletes old block bodies,
old transaction bodies, and old undo data outside the configured horizon. It
still validates new blocks and transactions. It cannot serve deleted bodies or
reorganize below its prune horizon; recovery requires resynchronizing from an
archive peer. Switching back to archive mode does not recreate deleted data.

For a small personal desktop node use pruned mode; for public bootstrap,
historical service, debugging, and maximum reorg tolerance use archive mode.

## Upgrade from v0.1

Keep the entire existing Entropy data directory backed up before upgrading.
For v0.1 this is normally `%APPDATA%\Entropy`; the dashboard shows the active
v0.2 database path.

- `chain.json` is fully validated, imported transaction by transaction into
  SQLite, checked against its old tip, then renamed to
  `chain.json.migrated.bak`.
- `peers.json` is imported into SQLite and renamed to
  `peers.json.migrated.bak`.
- A plaintext `wallet.json` blocks node startup until the desktop migration
  flow or `wallet-migrate` creates both a verified DPAPI `wallet.vault` and a
  verified password-protected `.entwallet` backup. The plaintext file is
  removed only after both copies pass verification.
- If legacy and SQLite tips disagree, startup fails without replacing either
  copy.

Legacy wallets preserve their original address but cannot be assigned a new
recovery phrase. Their portable `.entwallet` backup is therefore essential.

## Build and verify

Prerequisites: Go 1.26.5, Node.js 22+, Wails v2.13.0, WebView2, and NSIS for
the installer. The patch-level Go pin includes standard-library security fixes
used by the public network stack.

```powershell
cd D:\Entropy
go test ./...
go vet ./...
npm --prefix frontend ci
wails dev
```

Create the portable EXE, NSIS installer, and `SHA256SUMS.txt`:

```powershell
.\scripts\build.ps1
```

The build script runs Go tests and vet first. Current release artifacts are not
Authenticode-signed and builds are not yet reproducible.

## Documentation

- [Architecture](docs/architecture.md)
- [Node operations and wallet recovery](docs/operations.md)
- [Protocol v3](docs/protocol.md)
- [Roadmap](docs/roadmap.md)
- [Security policy and boundaries](SECURITY.md)
- [Changelog](CHANGELOG.md)

Entropy is licensed under the [MIT License](LICENSE).
