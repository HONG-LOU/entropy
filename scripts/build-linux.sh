#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 || ! $1 =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "usage: $0 VERSION" >&2
    exit 2
fi

version=$1
project=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
bin="$project/build/bin"
stage=$(mktemp -d "${TMPDIR:-/tmp}/entcoin-linux-package.XXXXXX")
trap 'rm -rf "$stage"' EXIT

command -v go >/dev/null || { echo "Go 1.26.5 is required" >&2; exit 1; }
go version | grep -q 'go1\.26\.5' || { echo "Go 1.26.5 is required" >&2; exit 1; }
command -v wails >/dev/null || { echo "Wails v2.13.0 is required" >&2; exit 1; }
wails version | grep -q 'v2\.13\.0' || { echo "Wails v2.13.0 is required" >&2; exit 1; }
command -v dpkg-deb >/dev/null || { echo "dpkg-deb is required" >&2; exit 1; }

cd "$project"
wails build -clean -trimpath -platform linux/amd64 -tags webkit2_41 -o entcoin-linux-amd64
go build -trimpath -o "$bin/entcoin-cli-linux-amd64" ./cmd/entcoin

install -d \
    "$stage/DEBIAN" \
    "$stage/usr/bin" \
    "$stage/usr/share/applications" \
    "$stage/usr/share/icons/hicolor/512x512/apps"
install -m 0755 "$bin/entcoin-linux-amd64" "$stage/usr/bin/entcoin"
install -m 0755 "$bin/entcoin-cli-linux-amd64" "$stage/usr/bin/entcoin-cli"
install -m 0644 "$project/deploy/linux-desktop/entcoin.desktop" \
    "$stage/usr/share/applications/entcoin.desktop"
install -m 0644 "$project/build/appicon.png" \
    "$stage/usr/share/icons/hicolor/512x512/apps/entcoin.png"
installed_size=$(du -sk "$stage/usr" | cut -f1)
sed \
    -e "s/@VERSION@/$version/g" \
    -e "s/@INSTALLED_SIZE@/$installed_size/g" \
    "$project/deploy/linux-desktop/control" > "$stage/DEBIAN/control"

package="$bin/entcoin_${version}_amd64.deb"
dpkg-deb --build --root-owner-group "$stage" "$package"

(
    cd "$bin"
    sha256sum \
        entcoin-linux-amd64 \
        entcoin-cli-linux-amd64 \
        "$(basename "$package")" > SHA256SUMS-linux.txt
)

echo "Built: $bin/entcoin-linux-amd64"
echo "Built: $bin/entcoin-cli-linux-amd64"
echo "Built: $package"
