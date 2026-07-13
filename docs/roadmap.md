# Entropy public-testnet roadmap

v0.2 provides the complete local product loop: protected wallet, SQLite UTXO
ledger, undo-based cumulative-work reorg, incremental header/body sync,
WebSocket relay, LAN discovery, mining, transaction history, legacy migration,
pruning, desktop packaging, and a headless node.

The remaining work is about public infrastructure, adversarial assurance, and
release trust. None of the items below is permission to assign real-world value
to ENT.

## Public bootstrap infrastructure

- Deploy multiple always-on archive seed nodes on independent networks and
  publish their ownership, retention, monitoring, and replacement policy.
- Add a resilient bootstrap mechanism, such as signed seed lists or DNS seeds,
  without making one operator a consensus authority.
- Exercise full genesis-to-tip bootstrap and archive serving under realistic
  public traffic and database growth.

## NAT traversal and reachability

- Add explicit, user-controlled UPnP IGD, NAT-PMP, or PCP mapping with clear
  firewall and privacy behavior.
- Add public address discovery and reachability tests without trusting a single
  peer's view.
- Improve peer address exchange, stale-address expiry, diversity selection, and
  behavior on IPv6 and carrier-grade NAT.

## Independent review and hostile-network testing

- Commission independent consensus, wallet/cryptography, P2P, persistence, and
  desktop-boundary audits; publish findings and fixes.
- Expand fuzzing, race, malformed database, resource exhaustion, eclipse,
  partition, timestamp, difficulty, and deep-reorg test campaigns.
- Run a long-lived adversarial testnet with independent miners and operators;
  measure propagation, stale-block rate, DAA behavior, and recovery from loss of
  major peers.
- Define and test a versioned consensus/protocol upgrade and rollback process.

## Release supply chain

- Produce reproducible Windows and CLI builds from a documented clean
  environment.
- Authenticode-sign Windows binaries and installers, publish detached release
  signatures, checksums, provenance, and a software bill of materials.
- Establish release-key rotation, compromise response, supported-version, and
  vulnerability-disclosure procedures.

## Operational maturity

- Establish public testnet status and incident channels, seed health
  monitoring, explicit release support windows, and measured database sizing
  guidance.
- Add privacy-conscious node diagnostics and exportable health reports without
  collecting wallet secrets.
- Validate multi-year schema migration and pruning behavior against projected
  31,536,000-block growth.

There is no scheduled mainnet milestone. A separate decision would require all
critical audit findings resolved, long-running independent testnet evidence,
stable governance and upgrades, and a new explicit risk review. Until then,
Entropy remains a public testnet and ENT must remain valueless.
