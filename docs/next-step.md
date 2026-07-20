# Entcoin next steps

## P0: security readiness

- Publish a normative consensus specification and cross-implementation test
  vectors for encoding, signatures, proof of work, difficulty adjustment,
  issuance, coinbase maturity, fork choice, and reorganization behavior.
- Expand fuzz, race, malformed-database, resource-exhaustion, eclipse,
  partition, timestamp, difficulty, and deep-reorganization testing. Run
  long-lived adversarial networks with independent miners and operators.
- Commission independent consensus, cryptography, wallet, P2P, persistence,
  installer, and desktop-boundary audits. Publish findings and resolve every
  critical issue before ENT is used to carry real-world value.
- Establish miner and operator diversity, reorganization and hash-rate alerts,
  public incident channels, and a versioned emergency upgrade, rollback, and
  governance process.
- Make release artifacts independently verifiable: reproducible builds where
  practical, signed checksums, detached signatures, provenance, an SBOM,
  Windows Authenticode, Linux package signing, and documented release-key
  rotation and compromise response.
- Operate at least three archive seeds across independent operators, providers,
  and jurisdictions. Add external health monitoring, sustained-load evidence,
  signed and versioned bootstrap data, and a DNS-based fallback.
- Add eclipse-resistant peer management with tried/new address tables,
  network-group diversity, feeler probes, reachability checks, peer scoring,
  and explicit inbound/outbound selection rules.
- Add explicit wallet lock/unlock sessions and memory locking where supported.
  Support offline or hardware signing, watch-only descriptors, and
  multisignature before positioning the wallet for material balances.

## P1: daily-use maturity

- Introduce versioned HD receive/change descriptors, address rotation, labels,
  and deterministic recovery.
- Add recent-block fee estimation, replace-by-fee policy,
  child-pays-for-parent handling, coin selection, and coin control.
- Improve initial synchronization with parallel header/body scheduling,
  compact block relay, and an optional independently verifiable UTXO snapshot
  that never bypasses header or accumulated-work validation.
- Add opt-in UPnP, NAT-PMP, or PCP mapping, plus Tor and I2P proxy support.
- Provide a local authenticated metrics/RPC surface, peer direction and
  transport counters, exportable privacy-conscious diagnostics, database
  growth forecasts, and multi-region seed monitoring.
- Validate schema migration, pruning, bootstrap, and archive serving against
  projected multi-year chain growth and realistic public traffic.
- Add CI coverage for race detection, sustained fuzzing, dependency and source
  vulnerability scanning, frontend checks, and multi-node network scenarios.

## P2: optional ecosystem work

- Build a separate read-only explorer and indexer.
- Add QR payment requests, an address book, wallet labels, CSV export, and
  richer transaction and coin-provenance details.
- Add signed update notifications with tested rollback behavior while keeping
  installation an explicit user action.
- Build a mobile watch-only wallet or remote signer that does not require the
  phone to run a full chain database.
