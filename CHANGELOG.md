# Changelog

All notable public-testnet changes are documented here. Entropy follows no
mainnet compatibility promise; protocol identity is the compatibility boundary.

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
