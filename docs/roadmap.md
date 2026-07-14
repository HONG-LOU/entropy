# Entropy mainnet roadmap

v1.0.0 establishes the public `entropy-mainnet-v1` compatibility boundary: a
new genesis, an isolated data directory, full desktop validation and relay,
wallet recovery, proof-of-work mining, SQLite persistence, incremental sync,
pruning, HTTPS bootstrap manifests, and an optional Windows archive-seed
deployment package.

The `mainnet` name is not an audit or production-readiness claim. The public
repository and complete local product loop make review possible; they do not
make ENT safe for real-world value. ENT must remain valueless until appropriate
independent audits and sustained hostile-network evidence exist.

## Public bootstrap operations

- Deploy multiple always-on archive seeds on independent networks and publish
  only endpoints that pass genesis-to-tip and external HTTPS/WSS health checks.
- Keep the public HTTPS manifest accurate, remove unhealthy endpoints quickly,
  and publish ownership, retention, monitoring, and replacement policy.
- Exercise full genesis-to-tip bootstrap and archive serving under realistic
  public traffic and projected database growth. Until this is done, the empty
  manifest correctly means that no active public seed is being claimed.
- Add manifest signing or DNS-based fallback without making any seed operator a
  consensus authority.

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
- Run long-lived adversarial network exercises with independent miners and
  operators; measure propagation, stale-block rate, DAA behavior, and recovery
  from loss of major peers.
- Define and test a versioned consensus/protocol upgrade and rollback process.

## Release supply chain

- Produce reproducible Windows and CLI builds from a documented clean
  environment.
- Authenticode-sign Windows binaries and installers, publish detached release
  signatures, checksums, provenance, and a software bill of materials.
- Establish release-key rotation, compromise response, supported-version, and
  vulnerability-disclosure procedures.

## Operational maturity

- Establish public network status and incident channels, seed health
  monitoring, explicit release support windows, and measured database sizing
  guidance.
- Add privacy-conscious node diagnostics and exportable health reports without
  collecting wallet secrets.
- Validate multi-year schema migration and pruning behavior against projected
  31,536,000-block growth.

The network identity is already `entropy-mainnet-v1`, but the security milestone
for real-value use has not been met. That requires all critical audit findings
resolved, long-running independent-network evidence, stable governance and
upgrade procedures, trustworthy releases, and a new explicit risk review.
