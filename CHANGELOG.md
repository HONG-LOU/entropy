# Changelog

All notable changes are documented here. The protocol identity is the network
compatibility boundary; a `mainnet` identity is not a security or audit claim.

## [1.0.0] - 2026-07-14

### Added

- Public `entropy-mainnet-v1` network with a new reward-free genesis block.
- Built-in HTTPS bootstrap-manifest discovery through the public repository and
  its CDN mirror, with a verified public archive seed at
  `https://template-chat.xyz`.
- Optional Windows archive-seed deployment package with Caddy HTTPS/WSS
  termination, service accounts, scheduled startup, firewall setup, health
  checks, and uninstall support.
- Explicit Linux/Windows seed mode with archive-only storage, an ephemeral
  non-financial identity, no persistent wallet, and disabled wallet/mining
  operations.
- Release artifacts for the desktop app, installer, headless CLI, public-seed
  deployment package, and SHA-256 checksums.

### Changed

- Desktop and package metadata now report version `1.0.0`.
- Fresh Windows mainnet state is isolated under
  `%LOCALAPPDATA%\Entropy\mainnet-v1`; published testnet directories are never
  selected automatically.
- A fresh desktop database starts pruned with a 20,000-block body horizon and
  then respects the persisted operator choice. CLI nodes remain archive by
  default, and the public-seed deployment enforces archive mode.
- Coinbase maturity is 100 blocks beginning with the first reward block at
  height 1. The target remains one block per 10 seconds, exactly 2,000,000 ENT
  over 31,536,000 reward-bearing heights, approximately ten years.
- HTTP and WebSocket endpoints retain the `/v2` transport path even though the
  network identity is `entropy-mainnet-v1`.

### Compatibility

- `entropy-mainnet-v1` rejects every testnet protocol identity and uses a new
  genesis, so old testnet chains are neither migrated nor replayed.
- Wallet control can be restored on mainnet from a known 24-word recovery phrase
  or verified `.entwallet` backup. Chain/database files must not be copied into
  the mainnet directory.
- The published HTTPS archive seed enables automatic cross-internet discovery;
  manually configured peers remain available for manifest or seed outages.

### Security status

- The source repository is public, but v1.0.0 has not received an independent
  consensus, cryptography, wallet, P2P, persistence, or desktop audit.
- The `mainnet` name does not imply production readiness. ENT must not carry
  real-world value unless and until independent audits and sustained hostile
  network testing establish an appropriate security basis.

## [0.2.0] - 2026-07-13

### Added

- SQLite WAL ledger with indexed UTXOs, wallet history, persistent mempool and
  peers, health records, full-synchronous writes, and startup integrity checks.
- Per-block undo records and atomic cumulative-work chain reorganization with
  orphaned-transaction mempool revalidation.
- Persistent archive/pruned storage policy and irreversible body/undo pruning.
- Header-first incremental HTTP synchronization and requested body batches.
- WebSocket transaction/block relay with keepalive, connection, message, and
  per-IP limits.
- Automatic LAN multicast discovery and persistent exponential peer backoff.
- Windows user-scope DPAPI wallet vault.
- New-wallet 24-word BIP39 recovery using a versioned Entropy P-256 derivation.
- Portable Argon2id/XChaCha20-Poly1305 `.entwallet` export and restore.
- Desktop transaction history, confirmation/spendable state, peer management,
  wallet recovery, database diagnostics, and pruning workflow.
- Headless `history`, `wallet-backup`, and `wallet-migrate` commands.
- NSIS installer build and release SHA-256 checksum generation.
- Single-instance desktop activation, automatic occupied-port fallback, and a
  pinned tag-to-GitHub-Release workflow.

### Changed

- Network identity is now `entropy-testnet-v3`.
- Chain state moved from whole-file JSON replay to incremental SQLite commits.
- Synchronization compares and validates headers before downloading candidate
  bodies and never accepts a serialized remote state wholesale.
- Coinbase outputs require 100-block maturity beginning at spending height 100,
  preserving the earlier published testnet history.
- Retry failures, manual/discovered peer state, and storage policy survive node
  restart.
- Desktop startup treats missing/corrupt backend, wallet, and ledger state as a
  fatal error instead of showing placeholder data.
- Clean Windows installs now store live data under `%LOCALAPPDATA%`; existing
  `%APPDATA%` wallets and chains are detected and reused.
- Mempool relay now enforces aggregate byte/input budgets and a minimum fee,
  while mixed-case hexadecimal addresses remain visible and spendable.
- Peer synchronization now rejects redirects, rate-limits validation work,
  folds replayed local prefixes, bounds disk staging, and cancels stale mining
  templates as soon as the active tip changes.

### Migration

- Valid `chain.json` and `peers.json` are imported and verified before becoming
  `.migrated.bak` files.
- Plaintext `wallet.json` requires an explicit migration that creates and
  verifies both local DPAPI and portable password-encrypted copies before
  deleting plaintext.
- A disagreement between legacy and SQLite chain tips stops migration without
  replacing either copy.

### Removed

- `GET/POST /v1/state` whole-state synchronization.
- New plaintext wallet storage.
- Production frontend mock data and simulated backend success paths.

### Security status

- v0.2 is a public testnet and has not received an independent audit.
- Release binaries are not Authenticode-signed and builds are not yet
  reproducible.
- ENT must not carry real-world value.

## [0.1.0]

- Initial educational MVP with a Wails desktop wallet/node, P-256 signed UTXO
  transfers, proof of work, exact 2,000,000 ENT height-based emission, manual
  HTTP peers, and atomic JSON persistence.
- Superseded by v0.2 and no longer supported.
