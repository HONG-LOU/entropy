# Entropy (ENT)

Entropy is a compact proof-of-work blockchain packaged as a Windows desktop node.
Open the app and the wallet, ledger validator, transaction relay, and peer server
start in the same process.

> This is an educational MVP. It uses real signatures and proof of work, but it
> has not been audited and must not hold assets with real-world value.

## What works

- P-256 wallet generation and checksummed `ent1...` addresses
- UTXO transactions, change, fees, signatures, and double-spend rejection
- SHA-256 proof of work and cumulative-work chain selection
- Pending transaction propagation in seconds over configured peers
- 10-second target block interval and persistent local chain state
- Exact 2,000,000 ENT maximum supply released over 31,536,000 reward blocks
- Single-file Wails desktop app plus a headless CLI node
- Atomic JSON persistence and full replay validation on startup

## Run the desktop node

The packaged Windows application is:

```text
D:\Entropy\build\bin\Entropy.exe
```

Double-click it. The first launch creates a wallet and chain data under:

```text
%APPDATA%\Entropy
```

The desktop window requires Microsoft WebView2 Runtime. Current Windows 10/11
installations normally already include it.

The app listens for peers on TCP port `47821`. Windows Firewall may ask whether
to allow access. A second user can add the first user's reachable node URL in
the Peers panel, for example:

```text
http://192.168.1.20:47821
```

Mining is off by default. Starting the app always runs a validating/relaying
node; mining starts only after pressing **Start mining**.

Back up `wallet.json` before using a persistent wallet. Anyone with that file
can spend the wallet's ENT. There is no password encryption in this MVP.

## Confirmation speed

The sender first commits a valid transfer to its local pending pool, then relays
to configured peers concurrently. A healthy LAN peer normally receives it in
well under a second, but the app does not call that a confirmation. The target
is one proof-of-work block every 10 seconds:

```text
Broadcast receipt       typically < 1 second on a LAN
1 confirmation          target about 10 seconds
6 confirmations         target about 1 minute
```

Fast proof-of-work chains have more short forks than Bitcoin. Six ENT
confirmations do not provide Bitcoin-level economic security.

## Monetary policy

There is no premine. Height zero is a fixed, reward-free genesis block. Amounts
use eight decimal places and integer arithmetic only.

```text
Maximum supply                  2,000,000.00000000 ENT
Target block spacing            10 seconds
Reward-bearing blocks           31,536,000
Expected emission duration      3,650 days (about 10 years)
Blocks 1..12,512,000            0.06341959 ENT
Blocks 12,512,001..31,536,000   0.06341958 ENT
Later blocks                    fees only
```

The one-atomic-unit difference distributes the division remainder. The two
reward ranges add up to exactly `200,000,000,000,000` atomic units.

Ten years is the target implied by block height and 10-second spacing, not a
wall-clock guarantee. The DAA can raise difficulty up to 255 leading zero bits,
but a public launch still requires long-running hash-rate and timestamp tests.

## Develop

Prerequisites: Go 1.25+, Node.js 22+, Wails v2, and WebView2.

```powershell
cd D:\Entropy
go test ./...
npm --prefix frontend install
wails dev
```

Build the Windows executable:

```powershell
.\scripts\build.ps1
```

Run a headless node:

```powershell
go run .\cmd\entropy node --data .\data\node-a --listen 0.0.0.0:47821 --mine
```

Run another node on the same computer:

```powershell
go run .\cmd\entropy node --data .\data\node-b --listen 127.0.0.1:47822 --peer http://127.0.0.1:47821
```

## MVP limits

- Peers are added manually; there is no public seed service, NAT traversal, or
  automatic internet discovery yet.
- P2P messages use HTTP without transport encryption or peer authentication.
- The chain uses a compact leading-zero-bit difficulty rule, not Bitcoin's
  production target encoding or a battle-tested fast-chain DAA.
- State is a replay-validated JSON file. Ten years of 10-second blocks requires
  a database, indexes, snapshots, and pruning before public launch.
- Coinbase outputs have no maturity period in this version.
- Wallet keys use standard-library P-256 rather than Bitcoin's secp256k1.
- There is no wallet encryption, recovery phrase, transaction fee market,
  hostile-network hardening, protocol upgrade mechanism, or security audit.
- Release binaries are not Authenticode-signed in this MVP.

See [docs/architecture.md](docs/architecture.md) for the design and
[docs/roadmap.md](docs/roadmap.md) for the path from MVP to a public testnet.
