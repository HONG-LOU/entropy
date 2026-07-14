# Public archive seed deployment

This runbook deploys one Entropy archive seed behind Caddy. The packaged
automation targets Windows x64; the same CLI seed mode supports a minimal Linux
systemd deployment. The node validates and stores the chain locally. Caddy
terminates TLS on TCP 443 and proxies HTTP plus WebSocket traffic to
`127.0.0.1:47821`.

This package does not create a VPS, domain, DNS record, or public IP. Do not add
an endpoint to the bootstrap manifest until it is deployed, externally
reachable, synchronized, and monitored.

## Topology and trust boundary

```text
internet peers
    |
    | HTTPS / WSS :443
    v
Caddy scheduled task (LOCAL SERVICE)
    |
    | HTTP / WS 127.0.0.1:47821
    | X-Entropy-Client-IP overwritten from the TCP client address
    v
entropy-cli scheduled task (dedicated Windows user)
    |
    v
archive SQLite ledger; ephemeral non-financial identity; no wallet file
```

The node must start with loopback-proxy trust enabled. It may trust
`X-Entropy-Client-IP` only when the immediate TCP `RemoteAddr` is loopback and
the header is one valid IP address. Caddy overwrites that header; it never
passes a caller-supplied value through. Do not put another proxy or CDN in front
without defining an additional trusted-proxy boundary.

The node's protocol currently advertises its listener port, not an HTTPS public
URL. Its actual listener remains `127.0.0.1:47821`, while peers must configure
`https://seed.example.com` from the bootstrap manifest. Do not expose or publish
`http://seed.example.com:47821`.

## Prerequisites

- Windows Server 2022 or Windows 10/11 x64 with a static public IP.
- A dedicated standard Windows user with a non-expiring service password.
- An ASCII DNS hostname with direct A/AAAA records for the host. Use DNS-only
  mode, not a CDN proxy.
- Inbound TCP 443 permitted by the cloud security group and upstream firewall.
  TCP 80 and TCP 47821 stay closed.
- Outbound DNS, NTP, HTTPS, and WSS access.
- `entropy-windows-seed-deploy.zip` from an Entropy release.
- An official Caddy v2 Windows amd64 `caddy.exe`, verified against the
  publisher's checksum.
- Storage outside the application directory for the archive database, Caddy
  certificates, and logs. Capacity requirements depend on real block traffic;
  monitor free space and database growth.

Caddy is configured for the ACME TLS-ALPN-01 challenge, so certificate issuance
uses TCP 443 and does not require port 80.

## Package contents

The release ZIP contains:

- `entropy-cli.exe`;
- `install.ps1` and `uninstall.ps1`;
- startup runners for the node and Caddy;
- `health-check.ps1`;
- a strict `seed.env.example`;
- the Caddy configuration.

Caddy is intentionally not redistributed in the Entropy release. Place the
verified `caddy.exe` beside `install.ps1`.

## Configuration

Copy `seed.env.example` to `seed.env` and replace every example value:

| Variable | Contract |
| --- | --- |
| `ENTROPY_SEED_DOMAIN` | Public DNS hostname only, without scheme or path |
| `ENTROPY_ACME_EMAIL` | ACME account contact |
| `ENTROPY_INSTALL_DIR` | Installed binaries and scripts |
| `ENTROPY_DATA_DIR` | Archive ledger only; outside install dir |
| `ENTROPY_CADDY_DATA_DIR` | Caddy certificate state; outside install dir |
| `ENTROPY_LOG_DIR` | Node, Caddy console, and access logs |
| `ENTROPY_LISTEN_ADDRESS` | Must be `127.0.0.1:47821` |
| `ENTROPY_BOOTSTRAP_PEER` | Optional explicit HTTPS peer base URL |
| `ENTROPY_BOOTSTRAP_MANIFEST_URLS` | 1..8 comma-separated HTTPS manifest URLs |
| `ENTROPY_PRUNE_DEPTH` | Must be `0` for an archive seed |
| `ENTROPY_DISABLE_DISCOVERY` | Must be `true` on a public server |
| `ENTROPY_TRUST_LOOPBACK_PROXY` | Must be `true` with the supplied Caddyfile |

The default manifest delivery paths are:

```text
https://raw.githubusercontent.com/HONG-LOU/entropy/main/network/mainnet.json
https://cdn.jsdelivr.net/gh/HONG-LOU/entropy@main/network/mainnet.json
```

The runner passes both URLs to the node bootstrap subsystem. The node validates
the bounded, versioned `entropy-mainnet-v1` document, refreshes it periodically,
and persists accepted HTTPS peers in SQLite. `ENTROPY_BOOTSTRAP_PEER` may add
one explicit peer for recovery when published manifest sources are unavailable.

The equivalent CLI controls are:

```text
--bootstrap-manifest URL    repeat for independent manifest delivery paths
--no-bootstrap              disable manifests for isolated recovery only
--trust-loopback-proxy      trust the Caddy client-IP header from loopback only
--seed-mode                 archive relay with no persistent wallet
```

Never use `--trust-loopback-proxy` when the node listens on a public interface.

## Install

Open an elevated Windows PowerShell 5.1 session in the extracted directory:

```powershell
$credential = Get-Credential -Message "Dedicated Entropy seed account"
Set-ExecutionPolicy -Scope Process Bypass
.\install.ps1 -NodeCredential $credential
```

The installer:

1. validates every environment value and the Caddyfile;
2. copies binaries and scripts to `ENTROPY_INSTALL_DIR`;
3. applies restrictive ACLs to install, wallet, Caddy, and log directories;
4. registers `EntropySeedNode` under the supplied Windows account;
5. registers `EntropySeedProxy` under `LOCAL SERVICE` with no Caddy admin API;
6. creates one inbound Windows Firewall rule for TCP 443 only;
7. starts the node, verifies local `/v2/status`, then starts Caddy.

Both scheduled tasks run at system startup, restart after failure, have no time
limit, and write bounded or age-pruned logs. Task Scheduler stores the node
account credential; it is not written to `seed.env`.

Group Policy can prohibit password-backed startup tasks. Treat registration or
the 45-second startup check failing as an installation failure. Keep the
dedicated account boundary instead of switching the node task to SYSTEM.

## DNS, firewall, and public verification

Before installation, point the hostname directly at the host public IP. At both
the cloud and Windows layers, permit inbound TCP 443 only. Do not create a rule
for TCP 47821. If the provider has a separate network firewall or NAT gateway,
apply the same rule there.

After DNS propagation and ACME issuance:

```powershell
.\health-check.ps1
```

The check verifies:

- loopback `GET /v2/status`;
- public HTTPS `GET /v2/status`;
- matching height and tip hash through the proxy;
- a WSS upgrade on `/v2/p2p` and a valid Entropy hello message.

For local diagnosis before DNS is ready:

```powershell
.\health-check.ps1 -LocalOnly
Get-ScheduledTask EntropySeedNode, EntropySeedProxy |
  Select-Object TaskName, State
Get-ChildItem D:\EntropySeedLogs
```

External monitoring should request `https://<domain>/v2/status` from another
network at least once per minute. Alert on TLS failure, non-200 responses,
network identity mismatch, sustained height lag, task restart loops, low disk
space, and certificate expiry.

## Seed identity and wallet boundary

The seed runner always supplies `--seed-mode`. The process creates a temporary
P-256 identity in memory, discards it on shutdown, and never creates
`wallet.vault`, a recovery phrase, or a portable backup. Seed mode cannot send
funds or mine. Do not send ENT to the transient address returned by its status
endpoint.

Startup refuses any persistent wallet artifact so an operator cannot
accidentally expose a desktop wallet through the public service. It also
refuses a previously pruned ledger because a bootstrap seed must serve history
from genesis.

Do not copy a live SQLite file without its WAL. For a consistent full data
copy, stop the node task first. A seed database contains no spending key.

## Bootstrap manifest publication

The committed manifest contains only externally verified archive seeds:

```json
{
  "version": 1,
  "protocol": "entropy-mainnet-v1",
  "peers": ["https://template-chat.xyz"]
}
```

Add only HTTPS base URLs without paths. Before merging an endpoint, verify its
HTTPS status, WSS handshake, archive history from genesis, external
reachability, operator independence, and monitoring. Keep no more than 16
entries. The v1 manifest is protected by GitHub HTTPS and repository review; it
is not cryptographically signed.

Deploy at least two archive seeds on independent providers and networks. Each
seed should explicitly bootstrap from another seed so a single empty database
does not become the accidental network authority.

## Upgrade and removal

To upgrade, keep the same `seed.env`, Windows account, and runtime directories.
Run the new package's `install.ps1` again; it stops and replaces the scheduled
tasks and binaries, then reuses the walletless archive ledger.

To remove tasks, firewall rules, binaries, and configuration while retaining
wallet, chain, certificates, and logs:

```powershell
.\uninstall.ps1
```

`-RemoveRuntimeData` permanently removes all retained runtime directories and
must be used only after wallet recovery is verified.

Entropy has not received an independent security or consensus audit. ENT must
not carry real-world value.
