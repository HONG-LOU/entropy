#!/usr/bin/env bash
set -euo pipefail

project=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
browser=${CHROMIUM_BIN:-$(command -v chromium || command -v chromium-browser)}
"$browser" --headless --disable-gpu --no-sandbox --hide-scrollbars \
    --window-size=1200,630 --force-device-scale-factor=1 \
    --screenshot="$project/assets/social-preview.png" \
    "file://$project/tools/social-preview.html"
