# Entcoin Website

Dependency-light static website for `https://entcoin.xyz`. English is rendered
without JavaScript; Chinese, live network status, menus, and
the network canvas are progressive enhancements.

## Local development

From the repository root:

```powershell
python -m http.server 4173 --directory website
node --check website/app.js
node --test website/tests/site-core.test.mjs
```

Open `http://localhost:4173`. The local status request is expected to show the
quiet unavailable state unless a compatible `/api/network-status` proxy exists.

Regenerate the social preview with:

```powershell
powershell -ExecutionPolicy Bypass -File website/tools/generate-social-preview.ps1
```

## Content and data boundaries

- Protocol and implementation claims are derived from the repository docs.
- The same-origin status endpoint accepts only `entropy-mainnet-v1` responses.
- Release links are pinned to the verified v1.0.8 assets in the official GitHub repository.
- The open-network section describes the two-region public seed topology and local validation boundary.

## Production layout

Each deployment is uploaded to `/var/www/entcoin/releases/<git-sha>`. The
`/var/www/entcoin/current` symlink points to the active immutable release.
`entcoin.xyz` serves that directory, `www.entcoin.xyz` redirects to the apex,
and only `GET /api/network-status` proxies to the local public archive node at
`http://127.0.0.1:47821/v2/status`.

The website host is `/etc/nginx/sites-available/entcoin-website`. It is separate
from `/etc/nginx/sites-available/entcoin-node`, which must remain enabled and
unchanged. Validate every configuration change with `nginx -t` before reload.

To roll back, atomically repoint the symlink to a known release and reload:

```bash
ln -sfn /var/www/entcoin/releases/<previous-sha> /var/www/entcoin/current
nginx -t && systemctl reload nginx
```

## Production verification

```bash
curl -fsS https://entcoin.xyz/api/network-status
curl -I http://entcoin.xyz
curl -I https://www.entcoin.xyz/path
curl -I https://entcoin.xyz/styles.css
curl -fsS https://node.entcoin.xyz/v2/status
systemctl is-enabled certbot.timer
```

`POST /api/network-status` must be rejected, the node WSS hello must still
succeed, and public TCP port 47821 must remain closed.
