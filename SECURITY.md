# Security policy

## Testnet warning

Entropy v0.2 is an unaudited public testnet. ENT is testnet currency and must
not represent money, deposits, securities, redeemable claims, or other assets
with real-world value. The project makes no mainnet or production-security
claim.

Cryptographic primitives and proof of work do not by themselves make a new
blockchain economically or operationally secure. Independent review, sustained
adversarial operation, trustworthy releases, and a mature response process are
still required.

## Supported versions

| Version | Status |
| --- | --- |
| `0.2.x` | Current public-testnet line; security fixes accepted |
| `0.1.x` | Educational MVP; unsupported, migrate before use |

Protocol identity `entropy-testnet-v3` is intentionally incompatible with the
removed v0.1 whole-state protocol.

## Report a vulnerability

For a vulnerability that could expose wallet secrets, bypass consensus,
inflate supply, cause an invalid reorg, corrupt persistent state, or remotely
exhaust/crash nodes, use GitHub's private vulnerability report for this
repository:

<https://github.com/HONG-LOU/entropy/security/advisories/new>

Do not include wallet phrases, private keys, live host credentials, or a
weaponized public exploit. If private vulnerability reporting is unavailable,
open a minimal issue requesting a private reporting channel without disclosing
technical exploit details.

Include when possible:

- affected commit, version, operating system, and build source;
- whether the desktop, CLI, wallet, ledger, consensus, or P2P path is affected;
- exact preconditions and minimal reproduction steps;
- expected versus observed behavior and security impact;
- logs or test fixtures with all secrets removed;
- whether the issue is already public or actively exploited.

Ordinary bugs, feature requests, documentation problems, and performance
questions can use public GitHub Issues. There is currently no bug-bounty or
guaranteed response-time program.

## Security boundaries

### Wallet

- The active local wallet is protected by Windows user-scope DPAPI and is
  decrypted inside the node process when running.
- A process or user with equivalent Windows-account access may be able to invoke
  DPAPI or inspect process memory. DPAPI is not protection from a compromised
  account or administrator.
- New wallets have a 24-word BIP39 phrase with an Entropy-specific P-256
  derivation. The phrase is the key; anyone who learns it controls the wallet.
- Portable `.entwallet` backups use Argon2id and XChaCha20-Poly1305. Their
  security still depends on a strong, secret password and uncompromised host.
- Migrated v0.1 wallets retain their original key and have no recovery phrase.
  Their verified encrypted backup is indispensable.
- A missing/corrupt vault fails closed and never triggers automatic generation
  of a replacement address.

### P2P listener

- TCP `47821` accepts untrusted HTTP and WebSocket input. Requests have size,
  time, concurrency, peer, and per-IP limits and received objects are locally
  validated.
- Direct P2P is neither encrypted nor authenticated. Peer URL, node ID, status,
  height, and chain-work claims are untrusted metadata.
- The public P2P endpoints do not expose send, restore, recovery phrase, mining,
  or private-key methods. Those are local desktop/CLI actions.
- LAN discovery is unauthenticated multicast and must be treated as a hint, not
  identity proof.
- Rate limits reduce obvious abuse but have not been independently tested as a
  complete denial-of-service defense.

### Consensus and persistence

- Blocks and transactions are validated against local UTXOs, signatures,
  maturity, size/timestamp rules, proof of work, and exact issuance.
- Fork choice uses cumulative work and reorg application is atomic in SQLite.
- SQLite uses WAL and full synchronization, per-block undo data, a process lock,
  and startup integrity checks. Hardware, filesystem, kernel, or storage-driver
  failures remain outside the application's control.
- Pruning intentionally removes historical bodies and undo records. A pruned
  node cannot serve deleted history or accept a reorg below its prune horizon.
- The 10-second target and compact leading-zero difficulty adjustment have not
  received the economic, game-theoretic, or long-duration review of established
  production networks.

## Known security limitations

- No independent consensus, cryptography, wallet, P2P, database, installer, or
  desktop-boundary audit has been completed.
- No built-in TLS, authenticated peer identity, public seed infrastructure,
  eclipse-resistant peer selection, or automatic NAT traversal is available.
- Windows binaries are not Authenticode-signed and builds are not yet
  reproducible. Verify published checksums, while understanding their limit.
- P-256 addresses and mnemonic derivation are Entropy-specific and not Bitcoin
  wallet compatible.
- The node has no hardware-wallet integration, multisignature policy, wallet
  passphrase unlock mode, or process sandbox.
- Network privacy is not a goal of v0.2. Peers observe IP addresses, timing, and
  the wallet address currently used as node ID.
- A single node or a network controlled by one miner/operator provides little
  independent failure or censorship resistance.
- Protocol upgrades and emergency governance are not mature enough for a
  real-value network.

## Operator practices

1. Run as a normal Windows user on a patched system.
2. Record the recovery phrase offline and export a separate encrypted backup
   before receiving or mining ENT.
3. Never paste a phrase, private key, or backup password into an issue, log,
   screenshot, chat, or diagnostic report.
4. Stop the node before copying its full data directory; do not copy only the
   live SQLite main file while WAL data may exist.
5. Verify release checksums and download only from the public repository's
   release page.
6. Expose only the P2P TCP port. Do not expose Windows file sharing, remote
   desktop, development servers, or data directories with it.
7. Use multiple independent archive peers for public experiments and monitor
   unexpected tip, peer, database, and reorg behavior.
8. Keep real funds and real credentials completely separate from this testnet.

See [Node operations](docs/operations.md) for backup, restore, firewall, NAT,
legacy migration, and pruning procedures.
