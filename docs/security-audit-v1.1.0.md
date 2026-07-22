# Entcoin v1.1.0 internal security audit

**Date:** 2026-07-22

**Scope:** repository source and release automation through v1.1.0

**Network:** `entropy-mainnet-v1`

**Assessment type:** project-led internal review, not an independent audit

## Executive summary

The review found one high-severity update-chain weakness, two medium-severity
supply-chain weaknesses, and documentation/release-control gaps. All findings
accepted for v1.1.0 were fixed and covered by automated checks.

No inflation, signature-bypass, invalid-proof-of-work, lower-work reorganization,
coinbase-maturity, cross-network replay, or non-atomic ledger transition was
found in the reviewed implementation. That statement describes review evidence;
it is not proof that no undiscovered defect exists.

v1.1.0 does not change consensus. Genesis, network ID, deterministic encoding,
issuance, difficulty, maturity, addresses, wallet derivation, database schema,
and wire protocol remain compatible with v1.0.x.

## Findings

| ID | Severity | Finding | Resolution |
| --- | --- | --- | --- |
| ENT-2026-01 | High | Update mirrors supplied both artifacts and expected checksums. A compromised mirror could replace both and pass SHA-256 verification. | Checksums now come only from the official GitHub Release. Mirrors remain untrusted byte sources. An adversarial regression test covers the exact attack. |
| ENT-2026-02 | Medium | `golang.org/x/crypto v0.51.0` contained disclosed vulnerabilities in unused SSH/OpenPGP packages. They were not reachable, but future imports could expose them. | Upgraded to `v0.52.0`; `x/sys` moved to `v0.45.0`; CI now runs pinned `govulncheck v1.6.0`. Reachable vulnerabilities: 0. |
| ENT-2026-03 | Medium | GitHub Actions used mutable major-version tags and release build jobs inherited repository write permission. | Every Action is pinned to a full commit SHA and JavaScript Actions use maintained Node 24 runtimes. Build jobs have read-only contents permission; only publish receives contents, OIDC, and attestation writes. |
| ENT-2026-04 | Low | CI did not continuously run the race detector, Go vulnerability scanner, or npm audit. | Added all three gates; release artifacts also receive build-provenance attestations. |
| ENT-2026-05 | Low | Release text was generated generically and did not preserve a reviewed, complete security log. | Added a controlled `RELEASE_NOTES.md`, bilingual entry points, documentation index, and this audit record. |
| ENT-2026-06 | Low | Linux package parent-directory modes inherited the build host umask and could be recorded as `0775`; release CLI binaries retained unnecessary debug data. | Every package directory is explicitly `0755`, writable directory modes fail the build, and release CLIs use stripped symbols. |

## Consensus and mathematical review

### Issuance

All monetary arithmetic is unsigned integer arithmetic. The constants are:

```text
UnitsPerENT = 100,000,000
MaxSupply   = 2,000,000 * UnitsPerENT
            = 200,000,000,000,000

EmissionBlocks = 10 * 365 * 24 * 60 * 60 / 10
               = 31,536,000

BaseSubsidy        = floor(MaxSupply / EmissionBlocks)
                   = 6,341,958
BonusSubsidyBlocks = MaxSupply mod EmissionBlocks
                   = 12,512,000
```

The implementation pays `BaseSubsidy + 1` for the first
`BonusSubsidyBlocks`, then `BaseSubsidy` through `EmissionBlocks`:

```text
12,512,000 * 6,341,959
+ (31,536,000 - 12,512,000) * 6,341,958
= 200,000,000,000,000 atomic units
```

Height zero and heights above `EmissionBlocks` pay zero subsidy. A block must
claim exactly subsidy plus fees; underclaiming and overclaiming are both
rejected. Fees consume existing inputs and cannot create supply. Checked
addition rejects overflow. The amount stored by valid consensus is far below
SQLite's signed 64-bit limit.

### Transactions and signatures

- Transaction and signature encodings begin with `entropy-mainnet-v1`, which
  prevents otherwise identical objects from replaying across network IDs.
- Each signing digest commits to every input outpoint and public key, every
  output amount and address, the signed input index, and that input's referenced
  amount and address.
- Validation reconstructs ownership from the SEC1 P-256 public key, parses one
  complete DER signature, rejects non-positive values and high-S signatures,
  and verifies with Go's standard ECDSA implementation.
- Transaction IDs commit to the complete signed transaction. Duplicate inputs,
  missing outputs, zero outputs, duplicate transaction IDs, overspending, and
  arithmetic overflow are rejected.
- Mempool outpoint uniqueness is also enforced by a SQLite unique constraint,
  providing defense in depth against concurrent double spends.

### Proof of work and fork choice

- A block hash commits to network ID, version, height, timestamp, previous hash,
  Merkle root, difficulty, and nonce.
- Valid work requires at least `difficulty` leading zero bits. For this target
  representation, expected hashes and recorded work are both `2^difficulty`.
- Cumulative work uses arbitrary-precision integers and is stored in explicit
  big-endian bytes, avoiding signed-SQL overflow.
- Chain replacement requires strictly greater cumulative work. Equal-work and
  lower-work candidates are rejected regardless of height.
- Candidate headers are validated before body download; bodies must match those
  headers and are fully validated while applying the replacement.

### Difficulty and time

- Difficulty starts at 22, first adjusts at height 120, and then every 60
  blocks. The comparison window spans median-time samples separated by 60
  heights.
- Median time uses up to the preceding 11 blocks, limiting the effect of a
  single timestamp outlier.
- Adjustment is a consensus step function: `+2` at or below one quarter target,
  `+1` below one half, `-2` at or above four times target, `-1` above twice
  target, otherwise unchanged. Results clamp to `[4, 255]`.
- Every new timestamp must exceed median time past and must not exceed local
  time by more than 120 seconds.

The arithmetic and implementation agree. The remaining concern is economic and
adversarial behavior: this compact, rapidly adjusting algorithm has not been
validated by years of heterogeneous mining or an independent implementation.

### Coinbase maturity and reorganization

Coinbase spending requires `spendingHeight - createdHeight >= 100`, implemented
with overflow-safe comparisons. The height-1 reward is rejected at height 100
and accepted at height 101. The rule is shared by block validation, mempool
acceptance, wallet spendability, mining, and reorganization rebuilds.

A reorganization occurs in one SQLite transaction. The old branch is
disconnected with hash-bound undo data, the candidate is connected forward,
greater cumulative work is required, and orphaned/current mempool transactions
are revalidated before commit. Any error rolls back the complete transition.

## Wallet and cryptography review

- New mnemonic entropy is 256 bits from the operating-system CSPRNG and encoded
  as 24 English BIP39 words.
- The versioned P-256 derivation performs HMAC-SHA-256 rejection sampling into
  `[1, N-1]`; it is deterministic and avoids modular-reduction bias.
- Portable backups use Argon2id v1.3 (`t=3`, `64 MiB`, two threads) and
  XChaCha20-Poly1305. Import bounds password length, envelope size, ciphertext,
  and KDF parameters before expensive allocation. Only one KDF runs per process.
- Windows local storage uses user-scope DPAPI. Linux stores a random 256-bit
  XChaCha20 key in Secret Service. Authentication failures do not silently
  generate replacement wallets.
- Wallet files use restrictive permissions, temporary files, flush/sync, and
  atomic installation. Existing backup creation refuses overwrite.
- Private keys and mnemonic text are cleared from mutable byte slices and
  material structures where practical, but Go strings, garbage collection, and
  the unlocked process prevent a guarantee of complete memory erasure.

## P2P and resource review

- HTTP and WebSocket bodies have explicit byte limits, strict unknown-field
  rejection, duplicate-key rejection, nesting limits, timeouts, and bounded
  response sizes.
- HTTP, WebSocket, heavy requests, sync, staging, per-IP concurrency, handshake
  rates, message bytes, validation work, queued bytes, peers, mempool count,
  mempool bytes, and mempool inputs are bounded.
- Peer URLs reject credentials, paths, query strings, fragments, invalid ports,
  and malformed DNS. Public peer exchange accepts only explicitly ported,
  globally routable IP literals and excludes documented/special ranges.
- The optional reverse-proxy client-IP header is trusted only from a loopback
  TCP peer and must contain one syntactically valid address.
- LAN discovery derives the host from the datagram source rather than advertised
  input. Bootstrap and peer exchange provide locations only; they cannot change
  consensus validation.
- Incoming candidate branches are staged in mode `0600` temporary files, capped
  at 512 MiB, body/header matched, fsynced, and removed after use.

## Persistence review

- Ledger directories are mode `0700`; symlink and non-regular database paths
  are rejected before opening.
- Existing databases are opened read-only and immutable to verify protocol and
  genesis before schema migration or WAL configuration.
- SQLite enables WAL, `synchronous=FULL`, foreign keys, `trusted_schema=OFF`, a
  busy timeout, and one process-level directory lock.
- Clean shutdown checkpoints WAL and records session state. Unclean startup runs
  `PRAGMA quick_check`.
- Numeric, address, body, transaction position, header, work, and undo values
  are validated when crossing database boundaries used by consensus operations.

`quick_check` proves SQLite structural consistency, not a full replay of every
historical consensus rule. Local database tampering remains inside the trusted
host boundary; operators should treat unexpected health or tip state as an
incident and restore from trusted peers/backups.

## Update and release trust

The updater obtains version discovery from the website manifest with a GitHub
feed fallback. Version strings and release URLs are canonicalized and restricted
to this repository. Artifact names are derived locally from version/platform.

For v1.1.0, expected checksums come only from GitHub Release. Mirrors can improve
availability but cannot define expected content. Redirects remain HTTPS-only and
host restricted; downloads are bounded, resumable, written mode `0600`, and
promoted only after SHA-256 matches.

CI pins Actions to immutable commits, locks Go 1.26.5, uses `npm ci`, tests both
platform paths, and attaches GitHub build provenance. The GitHub organization,
repository controls, Actions platform, and TLS PKI remain trust roots.

## Residual risks

These are known limitations, not closed findings:

1. No independent third-party audit or independent consensus implementation has
   validated this report.
2. P2P has no authenticated identity or built-in encryption. An operator proxy
   can protect transport but does not create protocol-level peer identity.
3. Peer selection lacks mature tried/new tables, network-group diversity, and
   robust Sybil/eclipse resistance. Public seeds have limited operator diversity.
4. The project-specific P-256 wallet derivation and fast difficulty algorithm
   have less ecosystem review than widely deployed cryptocurrency standards.
5. Wallet keys exist in process memory while in use. Hardware/offline signing,
   multisignature, watch-only descriptors, and explicit lock sessions are absent.
6. Checksums are authenticated only as strongly as the GitHub release account;
   detached project release signatures are not yet implemented.
7. Authenticode is conditional on release-secret configuration. Linux packages
   are not signed by a distribution repository key.
8. Builds use locked tool versions and provenance but are not yet demonstrated
   bit-for-bit reproducible across independent builders.
9. A pruned node cannot reorganize below its retained undo horizon and must
   resynchronize from an archive node after such a fork.
10. Network value, miner diversity, operator diversity, monitoring, incident
    response, and governance remain materially less mature than established
    monetary networks.

## Verification evidence

The v1.1.0 candidate was checked with Go 1.26.5 and Node.js tooling:

```text
go test -count=1 ./...                         PASS
go test -race -count=1 ./...                   PASS
go vet ./...                                   PASS
go build -trimpath ./cmd/entcoin               PASS
govulncheck ./...                               0 reachable vulnerabilities
npm audit --audit-level=high                    0 vulnerabilities
npm test                                        PASS
npm run build                                   PASS
node --test website/tests/site-core.test.mjs    PASS
git diff --check                                PASS
```

The module-level verbose scan still reports `GO-2026-5932` for the deprecated
`golang.org/x/crypto/openpgp` package. Entcoin neither imports nor calls that
package; the database lists no fixed module version. CI gates on reachable code
and the project will not introduce `openpgp`.

The local candidate additionally built the Ubuntu desktop, stripped CLI, and
`.deb`; verified all package checksums, metadata, contents, and directory modes;
and passed a real D-Bus Secret Service wallet startup/reopen smoke test. The
release workflow repeats that work, builds Windows artifacts, parses Windows
deployment scripts, and publishes provenance attestations. Those tag-triggered
results must be green before v1.1.0 is considered delivered.

## Release decision

The fixed findings justify v1.1.0 as a security-hardening release without a
consensus migration. The project remains suitable for experimental operation
under the disclosed boundaries. It should not be represented as independently
audited or used for value whose loss is unacceptable until the residual P0 work
in [next-step.md](next-step.md) is completed.
