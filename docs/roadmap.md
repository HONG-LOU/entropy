# From MVP to public testnet

The current repository proves the end-to-end product loop. A public network
needs these changes before real users or value are invited.

## Testnet gate

1. Freeze a versioned binary wire protocol and add header-first block sync.
2. Replace JSON chain storage with Pebble, indexed UTXO, snapshots, and pruning.
3. Replace the simplified difficulty rule with a tested target-based DAA suited
   to 10-second blocks; validate median-time-past and future timestamp drift.
4. Add seed nodes, peer scoring, connection limits, bans, and NAT traversal.
5. Add coinbase maturity, block/transaction size limits, fee policy, and reorg
   tests over competing chains.
6. Encrypt wallet keys, add a recovery phrase, and migrate to secp256k1.
7. Add protocol fuzzing, race tests, resource exhaustion tests, and independent
   security review.

## Mainnet gate

1. Publish deterministic builds, signed binaries, chain parameters, and genesis.
2. Run a long-lived adversarial public testnet with independent operators.
3. Complete consensus and wallet audits and resolve every critical finding.
4. Define upgrades, emergency communication, releases, and seed ownership.
5. Benchmark global propagation and confirm that a 10-second target has an
   acceptable stale-block rate under realistic block sizes.

Until those gates are met, Entropy is a local/LAN engineering prototype.
