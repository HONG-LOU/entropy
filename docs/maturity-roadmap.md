# Lightweight mainnet maturity roadmap

Entropy is a compact independent proof-of-work network, not a Bitcoin fork and
not a hosted account service. Its desktop application combines a validating
node, one shared chain database, and locally protected wallet profiles. There
is no server login: possession of a recovery phrase or encrypted backup is the
wallet authorization boundary.

The reference point for this roadmap is the operational shape of mature UTXO
networks such as Bitcoin Core: conservative consensus changes, outbound-first
connectivity, reachability-tested address management, descriptor/HD wallet
separation, reproducible releases, and independently operated infrastructure.
Matching those properties does not require matching Bitcoin Core's size.

## Implemented lightweight baseline

- Full local validation, cumulative-work fork choice, persistent UTXO state,
  mempool policy, atomic reorganization, and archive/pruned storage.
- Outbound HTTPS/WSS bootstrap and bidirectional reconciliation, so NAT clients
  can transact and synchronize without accepting inbound connections.
- Reachability-safe peer discovery: an inbound source IP is not treated as a
  dialable node, and never-reachable discovered records are retired.
- Multiple local wallet profiles over one chain database. Create, import,
  switch, export, and guarded removal do not duplicate or resynchronize the
  ledger.
- OS-protected local vaults, 24-word recovery, portable password-encrypted
  backups, and fail-closed startup when protected key material is unavailable.

## P0: required before meaningful real-world value

1. **Independent consensus and cryptography audit.** The implementation needs
   an external review, a written consensus specification, more cross-client
   vectors, fuzzing, and long hostile-network/reorg tests. Internal tests cannot
   establish monetary safety.
2. **Economic resistance to chain takeover.** A small SHA-256 network has very
   little hash power relative to Bitcoin. Six fast blocks are not six Bitcoin
   confirmations. Monitoring, miner diversity, reorg alerting, and an explicit
   emergency/governance policy are still missing.
3. **Release authenticity.** Windows and Linux artifacts need reproducible
   builds where practical, signed checksums, protected release keys, Windows
   Authenticode, and Linux repository/package signing. HTTPS alone is not a
   complete software-supply-chain trust model.
4. **Bootstrap and eclipse resistance.** Deploy at least three independently
   operated seeds across providers and jurisdictions. Replace the unsigned
   moving manifest with signed, versioned bootstrap data and add an addrman-like
   tried/new table, network-group diversity, feeler probes, and inbound/outbound
   anti-eclipse rules.
5. **Wallet hardening.** Add an explicit wallet lock/unlock session, memory
   locking where supported, hardware-wallet/offline signing, watch-only
   descriptors, and multisignature before positioning the wallet for material
   balances.

## P1: mature daily-use behavior

- **HD receive/change addresses and labels.** One address per wallet is simple
  but weak for privacy and bookkeeping. A versioned HD descriptor model should
  rotate receive/change addresses while keeping recovery deterministic.
- **Fee market tools.** Add recent-block fee estimation, replace-by-fee policy,
  child-pays-for-parent handling, coin selection, and coin control. A fixed
  default fee is acceptable only while block space is mostly empty.
- **Faster initial sync.** Add parallel header/body scheduling, compact block
  relay, bounded peer scoring, and an optional independently verifiable UTXO
  snapshot. A snapshot must never bypass header and accumulated-work checks.
- **Private connectivity.** Optional UPnP/NAT-PMP should be opt-in and clearly
  scoped. Tor/I2P proxy support provides a better mature-network path than
  forcing every household user to expose TCP 47821.
- **Operational visibility.** Add local authenticated metrics/RPC, peer
  direction and transport counters, reorg alerts, database growth forecasts,
  and public multi-region seed monitoring without exposing wallet APIs.

## P2: useful but not required for a compact core

- Read-only explorer/indexer as a separate optional process.
- QR payment requests, address book, wallet labels, CSV export, and improved
  transaction detail/coin provenance.
- Signed automatic update notifications. Installation should remain an
  explicit user action until release signing and rollback behavior are proven.
- Mobile watch-only wallet or remote signer. A phone should not be required to
  run the full desktop chain database.

## Deliberate non-goals

- No central username/password wallet accounts or custody server.
- No external database service for a normal desktop node.
- No smart-contract VM, token factory, staking layer, or web admin dependency.
- No trusted fast-sync checkpoint that can silently replace proof-of-work
  validation.

These constraints keep a normal installation self-contained. Infrastructure
redundancy, audits, release signing, and optional indexes can grow around the
node without turning the wallet or consensus engine into a hosted service.
