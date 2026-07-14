# Entropy (ENT)

Entropy v1.0.0 is a compact proof-of-work mainnet packaged as a Windows desktop
full node. Starting one application starts the wallet, SQLite ledger, full block
and transaction validation, peer synchronization, relay server, and optional
miner in the same process. It does not require a separate database server,
browser tab, or background daemon.

> `entropy-mainnet-v1` is a genesis and consensus compatibility identifier, not
> an audit or production-security claim. Entropy has not received an independent
> security or consensus audit. ENT must not be bought, sold, redeemed, or used
> to carry real-world value unless and until appropriate independent audits
> establish a suitable security basis.

The source repository is public and MIT-licensed:
<https://github.com/HONG-LOU/entropy>.

## What v1.0.0 includes

- A Wails desktop node with send, receive, mining, peer, history, wallet
  recovery, database, and pruning controls.
- A UTXO ledger in SQLite with WAL, `synchronous=FULL`, indexed balances and
  history, persistent mempool and peers, per-block undo records, atomic reorgs,
  and startup integrity checks.
- Header-first incremental synchronization over bounded HTTP batches, plus a
  negotiated same-WebSocket reverse reconciliation path for outbound-only
  nodes. The old whole-chain `/v1/state` exchange is removed.
- WebSocket transaction and block broadcast for low-latency relay, with HTTP
  broadcast as a fallback delivery path and bounded reconnect catch-up.
- Automatic LAN discovery plus persistent manual peers, connection limits,
  and exponential retry backoff.
- A Windows DPAPI-protected local wallet, 24-word BIP39 recovery for newly
  created wallets, and portable Argon2id/XChaCha20-Poly1305 `.entwallet`
  backups.
- Archive and pruned storage modes. Both validate incoming blocks locally;
  archive nodes additionally keep and serve all historical bodies.
- Built-in HTTPS bootstrap manifests with bounded validation and independent
  repository/CDN delivery paths, plus persistent manual peers.
- Strict network isolation: published testnet chains are never imported or
  replayed by mainnet. A known recovery phrase or verified `.entwallet` backup
  can restore the same wallet key into a fresh mainnet ledger.
- A headless CLI using the same node, consensus, wallet, ledger, and P2P code.
- An explicit archive-only `--seed-mode` for Linux or Windows public relays. It
  creates no wallet file, uses a new ephemeral identity after each restart,
  and disables sending, mining, recovery, backup, and restore operations.
- A portable desktop executable, NSIS per-user installer, headless CLI, and an
  optional Windows archive-seed deployment package.

## How many nodes are required?

One node is enough to create a wallet, mine, send transactions to itself, and
validate its local chain. Two nodes are the practical minimum for a network:
they can relay a payment, synchronize blocks, and resolve a fork by cumulative
work. There is no coordinator and no fixed quorum.

For a useful public network, run several independently operated nodes. At least
one must be reachable from the internet so outbound-only nodes behind NAT have
somewhere to connect. A node behind NAT is still a full validating and relaying
node. Once it connects, the reachable peer can reconcile that node's offline
transaction backlog and stronger chain through the same outbound WebSocket; the
NAT node simply cannot accept unsolicited inbound connections.

## Run the Windows desktop node

Until a v1 public seed is deployed and the v1 release is published, use the
portable executable or installer produced from this mainnet source tree. Do not
use the older v0.2 release assets for `entropy-mainnet-v1`. The portable build is
produced locally at:

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

The mainnet data directory for a clean installation is:

```text
%LOCALAPPDATA%\Entropy\mainnet-v1
```

The per-user uninstaller removes the application but deliberately keeps this
directory, including the wallet and chain. Back it up before deleting it
manually. Historical `%APPDATA%\Entropy` and `%LOCALAPPDATA%\Entropy` testnet
state is never selected as mainnet state. Do not copy a testnet database or
chain file into `mainnet-v1`.

A new desktop ledger starts in pruned mode retaining the newest 20,000 complete
block bodies. Later launches respect the storage policy persisted in that
ledger. This changes storage only: the desktop still fully validates every new
block and transaction and relays accepted data.

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
10-second network has a higher stale-block risk than Bitcoin, and six ENT
confirmations do not provide Bitcoin-level economic security.

## Monetary and consensus parameters

Entropy has no premine. Height zero is a fixed reward-free genesis block.
Amounts use eight decimal places and integer arithmetic.

```text
Network identity                  entropy-mainnet-v1
Maximum supply                    2,000,000.00000000 ENT
Atomic units per ENT              100,000,000
Target block spacing              10 seconds
Reward-bearing heights            1 through 31,536,000
Target emission duration          3,650 days (about 10 years)
Heights 1..12,512,000             0.06341959 ENT
Heights 12,512,001..31,536,000    0.06341958 ENT
Later heights                     transaction fees only
Coinbase maturity                 100 blocks, enforced from height 1
Maximum block body                1 MiB
Fork choice                       greatest cumulative proof of work
```

The one-atomic-unit reward difference distributes the integer division
remainder. The reward schedule sums to exactly `200,000,000,000,000` atomic
units at height 31,536,000. Fees move existing ENT and do not increase supply.

"Ten years" means 31,536,000 target intervals of 10 seconds, not a wall-clock
guarantee. Maturity applies from the first reward block: the height-1 coinbase
cannot be spent through height 100 and becomes spendable at height 101.

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

# Public archive relay behind a local HTTPS/WSS reverse proxy
.\build\bin\entropy-cli.exe node --seed-mode --data .\data\seed `
  --listen 127.0.0.1:47821 --prune-depth 0 --no-discovery `
  --trust-loopback-proxy
```

CLI nodes load the built-in HTTPS mainnet manifest sources by default. Repeat
`--bootstrap-manifest https://.../mainnet.json` to provide operator-managed
fallback sources, or use `--no-bootstrap` for a deliberately isolated/manual
topology. Explicit manifest URLs replace the built-in sources.

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
  reachable. You can still connect outbound to a public node; reconnect
  reconciliation does not require that public node to dial your advertised
  listen port.

The application fetches a small versioned bootstrap manifest over HTTPS from
the public repository, with a CDN mirror as fallback. At the v1.0.0 release the
checked-in manifest has an empty `peers` array, so the project does **not** claim
that an active public seed exists. An empty or unreachable manifest is reported
but does not prevent local startup.

Cross-internet joining therefore requires either a manually supplied reachable
peer or an operator to deploy a public seed and then publish its HTTPS URL in
the manifest. The release includes `entropy-windows-seed-deploy.zip`; see
[Public seed deployment](docs/public-seed.md). The seed package runs an archive
node behind a Caddy HTTPS/WSS reverse proxy on TCP 443. Direct desktop P2P is
plain HTTP/WebSocket, has no peer authentication, and does not automatically
traverse NAT. NAT traversal is unnecessary for a client node to reconcile over
an outbound connection, but at least one reachable peer is still required for
initial discovery and network-wide connectivity.

## Storage modes

Archive mode (`--prune-depth 0`) is the fresh CLI default and is enforced by the
public-seed package. It keeps headers, bodies, transaction data, indexes, UTXOs,
and undo records, so it can serve historical blocks and handle any reorg
available in its local history. A fresh desktop ledger instead starts at a
20,000-block prune depth; changing the setting later persists it.

Pruned mode retains every header, the current UTXO set, transaction/address
indexes, mempool, and peer records, but permanently deletes old block bodies,
old transaction bodies, and old undo data outside the configured horizon. It
still validates new blocks and transactions. It cannot serve deleted bodies or
reorganize below its prune horizon; recovery requires resynchronizing from an
archive peer. Switching back to archive mode does not recreate deleted data.

For a small personal desktop node use pruned mode; for public bootstrap,
historical service, debugging, and maximum reorg tolerance use archive mode.

## Move a wallet from a testnet release

Mainnet deliberately starts from a different genesis and isolated directory.
No v0.1 or v0.2 testnet chain, SQLite ledger, mempool, peer database, balance,
or transaction history is migrated or replayed. Keep old data backed up and
outside `%LOCALAPPDATA%\Entropy\mainnet-v1`.

Wallet keys are separate from chain history. Before leaving the testnet app,
record its 24-word recovery phrase or export and verify an encrypted
`.entwallet` backup. Start v1.0.0 to create the mainnet directory, then use the
desktop Wallet view to restore that phrase or backup. The address is recovered,
while balances and history are rebuilt only from the mainnet chain.

For a v0.1 plaintext `wallet.json`, use `wallet-migrate` against a copy of the
old data directory to create a verified `.entwallet`, then restore that backup
in the mainnet desktop app. A legacy wallet has no recovery phrase, so the
portable encrypted backup is essential.

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

Create the portable EXE, NSIS installer, CLI, public-seed deployment ZIP, and
`SHA256SUMS.txt`:

```powershell
.\scripts\build.ps1
```

The build script runs Go tests and vet first. Current release artifacts are not
Authenticode-signed and builds are not yet reproducible.

## Documentation

- [Architecture](docs/architecture.md)
- [Node operations and wallet recovery](docs/operations.md)
- [Mainnet protocol](docs/protocol.md)
- [Public seed deployment](docs/public-seed.md)
- [Roadmap](docs/roadmap.md)
- [Security policy and boundaries](SECURITY.md)
- [Changelog](CHANGELOG.md)

Entropy is licensed under the [MIT License](LICENSE).
