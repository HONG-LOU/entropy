# Entropy node operations

This guide covers the v0.2 Windows desktop node and the headless CLI. Entropy
is a public testnet. These procedures do not turn it into an audited or
production-safe monetary system.

## Install and launch

Release builds provide two Windows artifacts:

- `Entropy.exe`: portable Wails desktop application;
- `*installer*.exe`: NSIS per-user installer.

Verify the artifact against `SHA256SUMS.txt` from the same release before
running it. Current binaries are not Authenticode-signed, so a checksum proves
only that the file matches the published release artifact, not that a trusted
certificate authority verified its publisher.

Double-click the installed shortcut or portable EXE. A clean first launch
creates the wallet and SQLite ledger under `%LOCALAPPDATA%\Entropy`, starts the
TCP listener, and searches the LAN for peers. Mining remains off until
explicitly enabled. Existing `%APPDATA%\Entropy` v0.1/v0.2 data is detected and
reused during upgrade.

The desktop prefers TCP `47821`. If another process already owns that port it
starts on a free dynamic port instead and shows the actual listener plus a
diagnostic warning. LAN discovery advertises the selected port. CLI nodes keep
strict `--listen` behavior so operator mistakes fail visibly.

Microsoft WebView2 Runtime is required. It is normally present on current
Windows 10/11 systems; the NSIS build can install the bootstrapper when needed.

## Uninstall and retained data

The NSIS uninstaller removes the installed application and shortcuts. It does
not delete `%LOCALAPPDATA%\Entropy` or a reused `%APPDATA%\Entropy` directory;
wallet keys, the recovery marker, peer records, and the chain database remain
available for a later reinstall. Remove active data manually only after
confirming that the recovery phrase or a portable `.entwallet` backup works.

On managed Windows systems the legacy `%APPDATA%` location can roam. Exclude a
reused Entropy directory from profile replication because copying SQLite WAL
files between computers is not a supported backup method.

## Node count and topologies

### One computer, one node

One node is a complete isolated testnet. It can mine blocks and create locally
confirmed transactions, but there is no independent peer to observe relay,
synchronization, or forks.

### One computer, two nodes

Use separate data directories and TCP ports. Never point two processes at the
same data directory.

```powershell
go build -o .\build\bin\entropy-cli.exe .\cmd\entropy

.\build\bin\entropy-cli.exe node `
  --data .\data\node-a `
  --listen 127.0.0.1:47821 `
  --mine `
  --no-discovery
```

In another terminal:

```powershell
.\build\bin\entropy-cli.exe node `
  --data .\data\node-b `
  --listen 127.0.0.1:47822 `
  --peer http://127.0.0.1:47821 `
  --no-discovery
```

Node B validates and adopts node A's higher-work chain. Mining both nodes is a
useful reorg test but increases short forks.

### LAN nodes

The desktop listener defaults to `0.0.0.0:47821`. LAN discovery sends a small
announcement every five seconds to `239.255.78.21:47822/UDP`. Nodes on the same
multicast-capable subnet store discovered peer URLs and begin synchronization.

LAN discovery can fail on guest Wi-Fi, networks with client isolation,
multicast-disabled switches, VPN adapters, or restrictive firewalls. Add the
peer manually in that case:

```text
http://192.168.1.20:47821
```

The peer URL is the base URL only. Do not include `/v2/status`, credentials, a
query string, or fragment.

### Public nodes

At least one node must accept internet connections for nodes behind NAT to
bootstrap. v0.2 has no bundled public seed service, so operators exchange and
add reachable peer URLs manually.

An outbound-only node still downloads and independently validates headers,
blocks, transactions, signatures, proof of work, and monetary rules. It also
relays to its established outbound peers. It simply cannot accept a new inbound
connection initiated from the internet.

## Windows Firewall, router, and NAT

The two default ports have different purposes:

| Port | Transport | Scope | Purpose |
| --- | --- | --- | --- |
| `47821` | TCP | LAN or internet | HTTP synchronization and WebSocket relay |
| `47822` | UDP multicast | LAN only | Automatic local peer discovery |

To make a home node publicly reachable:

1. Give the computer a stable LAN address or DHCP reservation.
2. Allow inbound TCP `47821` for Entropy in Windows Defender Firewall.
3. Forward router TCP `47821` to that computer's TCP `47821`.
4. Test from a different internet connection, not from the same LAN.
5. Give peers `http://<public-ip-or-dns>:47821`.

Do not forward the multicast UDP port. Public discovery does not use it.

| Connection type | Can accept inbound? | What to do |
| --- | --- | --- |
| Public IPv4 directly on host | Yes | Open TCP firewall rule |
| Home NAT with port forwarding | Yes | Forward TCP `47821` and open firewall |
| Carrier-grade NAT | Usually no | Connect outbound to a reachable public peer |
| Corporate/hotel network | Usually no | Use outbound peers if policy allows |
| Public VPS | Yes | Bind TCP listener and restrict unrelated services |

Use an external status check against:

```text
http://<public-host>:47821/v2/status
```

The direct node server uses plain HTTP and WebSocket and has no peer
authentication. A TLS reverse proxy may provide HTTPS/WSS if it passes WebSocket
upgrades and preserves the bounded endpoint behavior. Do not expose unrelated
administrative services on the same port.

## CLI reference

Build once:

```powershell
go build -o .\build\bin\entropy-cli.exe .\cmd\entropy
```

Each command accepts `--data <directory>`. When omitted, Windows uses
the active default data directory. Each data directory has an exclusive lock, so stop the
desktop app or existing CLI node before opening the same directory.

### `node`

```text
entropy-cli node [--data DIR] [--listen HOST:PORT] [--peer URL]
                 [--mine] [--prune-depth BLOCKS] [--no-discovery]
```

| Option | Meaning |
| --- | --- |
| `--listen` | TCP P2P bind address; default `0.0.0.0:47821` |
| `--peer` | Add one persistent HTTP(S) peer at startup |
| `--mine` | Begin CPU mining immediately |
| `--prune-depth` | Explicitly persist `0` for archive, or retain `120..31,536,000` recent bodies |
| `--no-discovery` | Disable LAN multicast send and receive for this run |

When `--prune-depth` is omitted, the node keeps the storage policy already
recorded in its SQLite database. A fresh database defaults to archive mode.

### `status`

```powershell
.\build\bin\entropy-cli.exe status --data .\data\node-a
```

Prints address, confirmed balance, height, pending count, peers, and issued
supply. It opens the data directory directly and cannot inspect a concurrently
running node.

### `mine-one`

```powershell
.\build\bin\entropy-cli.exe mine-one --data .\data\node-a
```

Builds one candidate, searches proof of work, commits through normal consensus
validation, prints the block height/hash, and exits.

### `history`

```powershell
.\build\bin\entropy-cli.exe history --data .\data\node-a --limit 50
```

Prints pending and confirmed wallet transactions with received/sent amounts and
confirmation counts.

### `wallet-backup`

The backup password is read from `ENTROPY_WALLET_PASSWORD` so it does not appear
as a command-line argument. Passwords must contain 12 to 1024 UTF-8 bytes.

```powershell
$env:ENTROPY_WALLET_PASSWORD = Read-Host "Backup password"
.\build\bin\entropy-cli.exe wallet-backup `
  --data .\data\node-a `
  --output E:\Backups\entropy-wallet.entwallet
Remove-Item Env:\ENTROPY_WALLET_PASSWORD
```

The command appends `.entwallet` when the destination has another extension.
It refuses to silently replace malformed wallet state.

### `wallet-migrate`

Use this only when a v0.1 plaintext `wallet.json` prevents startup:

```powershell
$env:ENTROPY_WALLET_PASSWORD = Read-Host "Backup password"
.\build\bin\entropy-cli.exe wallet-migrate `
  --data "$env:LOCALAPPDATA\Entropy" `
  --output E:\Backups\entropy-legacy-wallet.entwallet
Remove-Item Env:\ENTROPY_WALLET_PASSWORD
```

The migration preserves the old P-256 key and address. There is intentionally
no CLI restore command in v0.2; restore backups or recovery phrases from the
desktop Wallet view.

## Data directory

The default Windows layout is:

```text
%LOCALAPPDATA%\Entropy\
  entropy.db                       SQLite ledger and indexes
  entropy.db-wal                   live WAL, normally checkpointed on shutdown
  entropy.db-shm                   live SQLite shared-memory file
  wallet.vault                     DPAPI-protected active wallet
  wallet.recovery-confirmed        local backup acknowledgement marker
  node.lock                        exclusive process lock
  chain.json.migrated.bak          retained v0.1 chain after migration, if any
  peers.json.migrated.bak          retained v0.1 peers after migration, if any
```

Peer records, mempool, UTXOs, history indexes, health events, prune policy, and
prune horizon are inside `entropy.db`.

The database is reconstructable from archive peers. `wallet.vault`, the
24-word phrase, or a `.entwallet` file is required to recover spending control.
Do not treat a database-only copy as a wallet backup.

For a consistent full-directory maintenance copy, stop the node first and copy
the directory while no `node.lock` process is active. Copying only `entropy.db`
while the node is live can omit committed data still present in its WAL.

## Wallet backup and recovery

### New v0.2 wallet

1. Open **Wallet** and reveal the 24-word recovery phrase.
2. Record the words in order on offline media. Do not store a screenshot or
   paste them into chat, email, or an unencrypted cloud note.
3. Confirm recovery in the app so the warning clears.
4. Also export a `.entwallet` file with a long, unique password.
5. Store the phrase, backup file, and backup password in separate locations.

The BIP39 words use an Entropy-specific P-256 derivation and an empty BIP39
passphrase. Generic Bitcoin or Ethereum wallets will not derive this address.

### Local DPAPI vault

`wallet.vault` unlocks automatically through Windows user-scope DPAPI. It is
bound to the Windows protection context and is not a portable recovery method.
Changing computers, reinstalling Windows, deleting the Windows profile, or
losing account protection may make the copied local vault unusable.

### Portable `.entwallet` backup

Portable backups encrypt authenticated wallet material with Argon2id and
XChaCha20-Poly1305. They support both mnemonic v0.2 wallets and migrated legacy
wallets. Keep the file and password separate. A forgotten password cannot be
reset by the project.

### Restore from backup

1. Stop mining.
2. Open **Wallet** and choose **Restore encrypted backup**.
3. Select the `.entwallet` file, enter its password, and confirm replacement.
4. Verify the displayed address against your known address.
5. Allow the existing ledger to refresh history and balances.

Restore atomically replaces the active DPAPI vault. It does not merge two
wallets and does not delete the chain database.

### Restore from 24 words

1. Stop mining.
2. Open **Wallet** and choose **Restore recovery phrase**.
3. Enter exactly 24 words and confirm wallet replacement.
4. Verify the restored address.
5. Export a fresh encrypted backup.

Migrated v0.1 wallets do not have a phrase. Do not invent one for them; restore
their `.entwallet` backup instead.

## Legacy v0.1 migration

Before upgrading, stop v0.1 and copy the whole data directory to offline
storage. That pre-migration copy contains a plaintext private key, so protect it
accordingly and securely retire it after verifying the encrypted recovery
copies. Start v0.2 once.

The chain migrator validates `chain.json`, imports it into SQLite, verifies the
new height and tip hash, and then renames the old file. Peers follow the same
import-before-rename rule. A mismatch between a pre-existing SQLite tip and the
legacy tip is fatal and leaves both copies in place.

Plaintext `wallet.json` is handled separately because chain data may be
recreated but a private key may not. Startup remains blocked until the user
selects a portable backup destination and supplies a valid password. Migration
then:

1. Validates the old wallet and records its expected address.
2. Creates and verifies the portable `.entwallet` backup.
3. Creates and verifies the DPAPI `wallet.vault` with the same address.
4. Removes plaintext `wallet.json` only after both encrypted copies verify.

The operation is restartable after interruption and refuses mismatched partial
copies.

## Archive and pruning operations

Archive is the fresh-node default and is required for a node intended to serve
new peers from genesis. It consumes increasing disk space over the ten-year
schedule but preserves maximum reorg and diagnostic history.

Pruned mode keeps all headers, UTXOs, address summaries, and current operation
state while deleting old bodies and undo records. The retention setting is
persistent. The minimum is 120 blocks so the retained window covers timestamp
and initial difficulty context.

The desktop **Database** panel previews the irreversible cutoff before applying
pruning. The equivalent CLI startup policy is:

```powershell
.\build\bin\entropy-cli.exe node --prune-depth 20000
```

Set `--prune-depth 0` explicitly to stop future pruning. This changes policy
only; it cannot reconstruct data already deleted.

Do not use a pruned node as the only public bootstrap peer. It responds with
`410 Gone` when another node requests a deleted block body. A reorg whose common
ancestor is below the prune horizon is also rejected rather than guessed.

### Recover historical bodies or a deep reorg

1. Stop the node and verify a wallet phrase or `.entwallet` backup.
2. Move `entropy.db`, `entropy.db-wal`, and `entropy.db-shm` to a quarantine
   directory. Do not delete `wallet.vault`.
3. Start a fresh archive ledger and add a trusted archive peer.
4. Wait for full synchronization and verify the wallet address, height, tip,
   balance, and history.
5. Retain the quarantined database until verification is complete.

## Mining and confirmations

The built-in miner uses CPU workers. It is suitable for a small testnet, not an
efficient production mining deployment. Closing the app or stopping mining
cancels active work. A found block is committed only if its snapshot tip is
still current.

Treat mempool receipt as propagation, not settlement. The 10-second interval is
a target, not a service guarantee. Wait more confirmations for larger testnet
experiments and expect occasional short reorganizations.

New coinbase output maturity is 100 blocks from activation height 100. The app
can show a confirmed balance that is larger than the currently spendable
balance while recent mining rewards mature.

## Troubleshooting

### `listen tcp ... address already in use`

Another process owns the port. Stop it or choose another `--listen` port. Every
node on one computer needs a unique TCP port.

### Data directory is locked

Close the desktop and CLI process using that directory. Do not delete
`node.lock` while a process is alive. If every process is stopped, relaunch and
let the operating-system lock determine whether the stale file is reusable.

### `legacy wallet requires encrypted migration`

Complete the desktop migration dialog or run `wallet-migrate`. Keep the chosen
`.entwallet` file and password; a migrated wallet has no recovery phrase.

### Protected wallet is missing or cannot decrypt

The node deliberately does not create a replacement wallet. Restore the known
24-word phrase or `.entwallet` backup. Silent replacement would display a new,
empty address and risk permanent loss.

### Peer remains offline

Check the base URL, remote TCP firewall, router forwarding, and `/v2/status`.
Retry delay grows exponentially to five minutes after repeated failures and
resets after a successful contact.

### Sync cannot fetch old blocks

The selected peer may be pruned and return `410 Gone`. Add an archive peer. If
your own prune horizon blocks a required reorg, follow the database resync
procedure above.

### LAN peers are not discovered

Confirm both nodes are on the same subnet, permit UDP multicast `47822`, and
disable Wi-Fi client isolation. Otherwise add peers manually. Use
`--no-discovery` when multicast is intentionally prohibited.

### Desktop window is blank or does not start

Install or repair Microsoft WebView2 Runtime. The node backend treats startup
failures as fatal and does not show fake balances or simulated success.

## Operational security boundary

Opening TCP `47821` exposes a deliberately public P2P parser to untrusted input.
The implementation applies message, connection, and timeout limits, but has not
been independently audited. Keep Windows patched, run as a normal user, keep
wallet backups offline, and read [the security policy](../SECURITY.md).
