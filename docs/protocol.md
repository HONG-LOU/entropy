# Entropy public-testnet protocol v3

This document describes the v0.2 implementation. The consensus/network identity
is `entropy-testnet-v3`. The HTTP path prefix remains `/v2` because it denotes
the second transport design; endpoint numbering and network identity are not
the same version counter.

Protocol v3 is a testnet protocol and may change incompatibly in a later
release. It is not an authenticated payment API.

## Transport overview

```text
LAN discovery       UDP multicast announcement
Catch-up sync       bounded HTTP JSON requests
Live relay          WebSocket JSON messages
Fallback relay      HTTP transaction/block POST
```

The built-in server listens on plain TCP, normally port `47821`. A peer base URL
is `http://host:port` or `https://host:port` without a path. HTTPS peers map to
WSS for the real-time channel, but TLS must be supplied by an operator's reverse
proxy because the node does not terminate TLS itself.

All JSON is size-limited and decoded against explicit structures. Unknown,
duplicate, trailing, and excessively nested data is rejected. Amounts and
nonces are JSON integers; hashes and cumulative work are encoded as strings in
the fields shown below.

## Chain status

`GET /v2/status` returns:

```json
{
  "protocol": "entropy-testnet-v3",
  "name": "Entropy",
  "symbol": "ENT",
  "height": 123,
  "tip_hash": "...",
  "chain_work": "...",
  "listen_port": 47821
}
```

`chain_work` is an unsigned base-10 integer. A node rejects status with a
different protocol/name/symbol, empty tip hash, or malformed work. Status is
informational; it never causes direct state replacement.

## Block locator and headers

`POST /v2/headers` accepts one to 64 active-chain hashes, newest first, plus a
batch limit from 1 to 2,000:

```json
{
  "locator": ["newest-known-hash", "older-hash", "genesis-hash"],
  "limit": 2000
}
```

The normal locator contains the newest ten consecutive ancestors, then steps
back exponentially, and always ends with genesis. The server selects the first
hash it has on its active chain and returns:

```json
{
  "protocol": "entropy-testnet-v3",
  "common_height": 120,
  "common_hash": "...",
  "headers": []
}
```

Each item in `headers` uses the normal block JSON fields but has no transaction
body. The receiver checks continuity, expected difficulty, median-time-past,
future-time bound, Merkle commitment, header hash, proof of work, and cumulative
work before requesting bodies. A response that changes its common point during
pagination is rejected. One synchronization attempt validates at most 20,000
headers. A peer that starts from an unnecessarily old locator cannot force a
redownload: headers identical to the local active prefix are folded into the
effective common ancestor.

## Block bodies

`POST /v2/blocks` accepts one to 8 unique hashes:

```json
{
  "hashes": ["block-hash-1", "block-hash-2"]
}
```

The response preserves request order:

```json
{
  "protocol": "entropy-testnet-v3",
  "blocks": []
}
```

Each body must exactly match its requested/validated header. A missing hash
returns `404 Not Found`; a body deleted by pruning returns `410 Gone`. Archive
nodes are therefore required for genesis-to-tip bootstrap.

The synchronizer fetches body batches only after the candidate header chain can
beat local cumulative work. It then disconnects the old branch, validates and
connects the candidate bodies, checks higher cumulative work, rebuilds the
mempool, and commits in one SQLite transaction. Any invalid body or database
error rolls back the entire replacement.

A direct extension commits at most one 8-block body batch per sync round, so a
slow peer still makes resumable progress. A non-direct candidate is staged in a
single protected temporary file with a 512 MiB ceiling before the atomic reorg.
Candidates beyond the 20,000-header or 512 MiB safety boundary are rejected;
operators must use a closer archive peer or rebuild the local database.

## Mempool catch-up

`GET /v2/mempool?limit=N&offset=M` returns up to 64 validated pending transactions.
Clients advance the zero-based offset across bounded sync rounds and reset it
when a short page is returned:

```json
{
  "protocol": "entropy-testnet-v3",
  "transactions": []
}
```

The receiver runs each transaction through its own UTXO, signature, maturity,
fee, conflict, and resource validation. A sender cannot force an entry into the
remote mempool by including it in this response. Local relay policy limits the
entire mempool to 5,000 transactions, 32 MiB, and 20,000 total inputs. The
minimum relay fee is 1,000 atomic units per started KiB; miners may still include
otherwise consensus-valid transactions under their own policy.

## HTTP relay

`POST /v2/transactions` accepts one signed transaction. Success returns HTTP
`202 Accepted` and its transaction ID. Invalid, duplicate, or conflicting data
returns `409 Conflict`.

`POST /v2/block` accepts one complete block. Success returns HTTP `202 Accepted`
and its block hash. Invalid or nonconnecting data returns `409 Conflict`.

These endpoints are the fallback and fan-out path. They call the same ledger
validation used by local wallet submissions, mining, and synchronization.

The v0.1 `GET/POST /v1/state` endpoint is intentionally absent. A v3 peer never
accepts another node's serialized whole-chain/whole-mempool state.

## WebSocket live relay

`GET /v2/p2p` upgrades to WebSocket. Direct peer clients omit the browser
`Origin` header. The connection begins with a `hello` message:

```json
{
  "type": "hello",
  "protocol": "entropy-testnet-v3",
  "node_id": "ent1...",
  "listen_port": 47821,
  "status": {
    "protocol": "entropy-testnet-v3",
    "name": "Entropy",
    "symbol": "ENT",
    "height": 123,
    "tip_hash": "...",
    "chain_work": "...",
    "listen_port": 47821
  }
}
```

`node_id` is currently the node wallet address and is not an authenticated peer
identity. A receiver rejects an empty/self node ID or an incompatible protocol.
For an inbound connection, the observed remote IP plus advertised listen port
may be recorded as a discovered peer.

Supported message types are:

| Type | Payload | Behavior |
| --- | --- | --- |
| `hello` | node ID, listen port, optional status | establish compatibility and peer address |
| `status` | status object | update peer height and liveness |
| `transaction` | complete transaction | validate, persist, and relay if accepted |
| `block` | complete block | validate and connect; trigger catch-up if needed |
| `ping` | none | respond with protocol `pong` |
| `pong` | none | keepalive acknowledgement |

Unknown types, binary frames, oversized messages, malformed JSON, and protocol
mismatches close the connection. The implementation sends WebSocket control
pings every 20 seconds and requires activity/pong within the read deadline.
The bounded send queue may drop a live relay message for a slow peer instead of
blocking consensus or wallet actions; periodic HTTP synchronization repairs
missed chain and mempool data.

## LAN discovery

Nodes listen and announce on:

```text
239.255.78.21:47822/UDP
```

An announcement is at most 2 KiB:

```json
{
  "protocol": "entropy-testnet-v3",
  "node_id": "ent1...",
  "port": 47821
}
```

The receiver uses the datagram's source IP, not an advertised host string, and
combines it with the validated port. It ignores self, incompatible, malformed,
and out-of-range announcements. Discovery is limited to multicast-capable LANs
and is disabled by `--no-discovery`.

## Peer lifecycle and limits

Manual and discovered peers persist in SQLite. Manual status survives later LAN
rediscovery. A failed peer becomes offline and uses exponential retry:

```text
1s, 2s, 4s, 8s, ... up to 5 minutes
```

A successful request resets failures, last error, and next-attempt time. The
current node caps the stored peer set and live WebSocket set at 64 each.

Inbound HTTP work is limited to 32 concurrent requests globally and eight per
source IP. WebSockets are limited to 32 globally and four per source IP. Server
header/read/write/idle timeouts and per-message limits bound stalled clients.
The limits are implementation policy, not consensus, and may change without a
network reset.

## Security and privacy properties

- Every received transaction, body, and header is locally validated.
- Peer status and node ID are untrusted hints.
- P2P has no bearer token, mutual authentication, or built-in encryption.
- The wallet private key and recovery phrase never appear in protocol messages.
- The node wallet address is exposed as `node_id`, so peers can correlate node
  connectivity with that address.
- IP addresses, manually configured URLs, failure times, and errors persist in
  the local database.
- No public seed deployment, NAT traversal, or anonymity layer is included.

Connection limits and strict decoding are defensive controls, not audit
evidence. Report sensitive issues through the process in
[SECURITY.md](../SECURITY.md).
