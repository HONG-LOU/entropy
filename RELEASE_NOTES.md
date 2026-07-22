# Entcoin v1.1.0

Entcoin v1.1.0 is the first security-hardening release of the
`entropy-mainnet-v1` line. It preserves the genesis block, consensus rules,
wallet derivation, addresses, ledger schema, and P2P compatibility. Existing
v1.0.x nodes can upgrade in place without moving chain or wallet data.

## Security

- Removed update mirrors from the checksum trust path. Installers may still be
  downloaded from the Asia or website mirrors, but their expected SHA-256 now
  comes only from the official GitHub Release. A compromised mirror can no
  longer replace both an artifact and its checksum.
- Added a regression test that serves a malicious artifact and matching
  malicious mirror checksum, then verifies that the updater rejects it and
  installs only bytes matching the GitHub checksum.
- Upgraded `golang.org/x/crypto` to `v0.52.0` and `golang.org/x/sys` to
  `v0.45.0`. `govulncheck` reports no vulnerabilities in reachable code.
- Pinned every GitHub Action to an immutable commit, reduced release-token
  permissions, added race and vulnerability gates, and enabled npm dependency
  auditing in CI.
- Added GitHub build-provenance attestations for every published artifact.
- Fixed Linux package-directory modes at `0755`, added a build-time rejection
  for group/other-writable package directories, and stripped release CLI debug
  symbols.

## Verification

- Re-audited issuance arithmetic, transaction signing, proof-of-work,
  difficulty adjustment, timestamp rules, coinbase maturity, cumulative-work
  fork choice, atomic reorganization, pruning boundaries, P2P decoding and
  resource limits, wallet encryption, update delivery, and release automation.
- Confirmed the reward schedule emits exactly `200,000,000,000,000` atomic
  units over `31,536,000` reward-bearing blocks and never exceeds the cap.
- Passed all Go tests, the full Go race suite, `go vet`, headless CLI build,
  frontend and website tests, frontend production build, npm audit, and
  reachable-code vulnerability scanning with Go 1.26.5.

## Documentation

- Added complete English and Simplified Chinese README entry points.
- Added a navigable documentation index and the v1.1.0 internal security audit
  report, including verified properties, fixed findings, residual risks, and
  the exact release gates.
- Updated architecture, protocol, operations, website, updater, package, and
  build metadata to v1.1.0.
- Added a tag-aware release metadata gate so package, lockfile, Wails, updater,
  website, README, changelog, and release-note versions cannot diverge.

## Known Boundaries

This is a project-led internal audit, not an independent third-party security
assessment. P2P traffic is unauthenticated and unencrypted unless an operator
adds TLS, peer discovery is not Sybil-resistant, release checksums remain rooted
in the GitHub repository account, and Windows signing depends on the release
certificate being configured. Read `SECURITY.md` and
`docs/security-audit-v1.1.0.md` before using Entcoin with material value.

## Upgrade

Stop Entcoin cleanly, retain a verified recovery phrase or `.entwallet` backup,
install v1.1.0, and start normally. The application reuses the existing
`mainnet-v1` data directory. Verify the checksum from the GitHub Release before
running a downloaded artifact.
