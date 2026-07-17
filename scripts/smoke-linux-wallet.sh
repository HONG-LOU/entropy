#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
    echo "usage: $0 ENTROPY_CLI TEMP_HOME" >&2
    exit 2
fi

cli=$1
test_home=$2
port=47911

rm -rf "$test_home"
mkdir -p "$test_home"
export HOME=$test_home
eval "$(printf 'entropy-ci-keyring' | gnome-keyring-daemon --unlock --components=secrets)"

"$cli" node \
    --data "$test_home/node" \
    --listen "127.0.0.1:$port" \
    --no-discovery \
    --no-bootstrap >"$test_home/node.log" 2>&1 &
pid=$!
cleanup() {
    if kill -0 "$pid" 2>/dev/null; then
        kill -TERM "$pid" 2>/dev/null || true
        wait "$pid" || true
    fi
}
trap cleanup EXIT

ready=false
for _ in $(seq 1 20); do
    if curl -fsS "http://127.0.0.1:$port/v2/status" >"$test_home/status.json"; then
        ready=true
        break
    fi
    sleep 1
done
if [[ $ready != true ]]; then
    cat "$test_home/node.log" >&2
    exit 1
fi

kill -TERM "$pid"
wait "$pid"
trap - EXIT

"$cli" status --data "$test_home/node" >"$test_home/wallet-status.txt"
grep -q '"protection": "linux-secret-service-user-v1"' "$test_home/node/wallet.vault"
grep -q '"algorithm": "xchacha20-poly1305"' "$test_home/node/wallet.vault"
if grep -Eq 'mnemonic|private_key' "$test_home/node/wallet.vault"; then
    echo "wallet.vault contains plaintext secret fields" >&2
    exit 1
fi
[[ $(stat -c '%a' "$test_home/node/wallet.vault") == 600 ]]
grep -q '^peers:     0$' "$test_home/wallet-status.txt"

[[ $(stat -c '%a' "$test_home/node/wallets") == 700 ]]
mapfile -t profile_vaults < <(find "$test_home/node/wallets" -maxdepth 1 -type f -name '*.vault' -print)
[[ ${#profile_vaults[@]} -eq 1 ]]
profile_vault=${profile_vaults[0]}
grep -q '"protection": "linux-secret-service-user-v1"' "$profile_vault"
grep -q '"algorithm": "xchacha20-poly1305"' "$profile_vault"
if grep -Eq 'mnemonic|private_key' "$profile_vault"; then
    echo "wallet profile contains plaintext secret fields" >&2
    exit 1
fi
[[ $(stat -c '%a' "$profile_vault") == 600 ]]

cat "$test_home/status.json"
cat "$test_home/wallet-status.txt"
