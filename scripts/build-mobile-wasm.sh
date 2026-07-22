#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
output_dir="$repo_root/mobile/public/wasm"
go_root="$(go env GOROOT)"
mkdir -p "$output_dir"
cd "$repo_root"
GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o "$output_dir/entcoin.wasm" ./cmd/entcoin-wasm
cp "$go_root/lib/wasm/wasm_exec.js" "$output_dir/wasm_exec.js"
