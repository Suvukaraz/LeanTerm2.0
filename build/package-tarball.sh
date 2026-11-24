#!/usr/bin/env bash
# Builds the generic Linux archive used by install.sh on distributions without
# .deb packages (Fedora, openSUSE, Arch, ...): binary, the pureedit launcher,
# desktop entries (PureTerm and PureEdit), icons.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

VERSION="${1:-$(tr -d '[:space:]' < "$ROOT_DIR/VERSION")}"
ARCH="amd64"
NAME="pureterm_${VERSION}_linux_${ARCH}"

cd "$ROOT_DIR"
# The Gio GUI with the libghostty engine: build libghostty first
# (build/gio/build-libghostty.sh) and set PKG_CONFIG_PATH to its prefix.
# PURETERM_TAGS= builds without libghostty (the vt engine only).
TAGS="${PURETERM_TAGS-ghostty}"
echo "Building PureTerm binary..."
go build ${TAGS:+-tags "$TAGS"} -trimpath -ldflags="-s -w" -o "$ROOT_DIR/bin/pureterm" ./cmd/pureterm

STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT
DIR="$STAGE/$NAME"
mkdir -p "$DIR/icons"

install -m 0755 "$ROOT_DIR/bin/pureterm" "$DIR/pureterm"
install -m 0755 "$ROOT_DIR/build/linux/pureedit" "$DIR/pureedit"
for size in 16 24 32 48 64 128 256 512; do
    install -m 0644 "$ROOT_DIR/build/linux/icons/pureterm-${size}.png" "$DIR/icons/pureterm-${size}.png"
    install -m 0644 "$ROOT_DIR/build/linux/icons/pureedit-${size}.png" "$DIR/icons/pureedit-${size}.png"
done
install -m 0644 "$ROOT_DIR/PureTerm_Icon.svg" "$DIR/icons/pureterm.svg"
install -m 0644 "$ROOT_DIR/PureEdit_Icon.svg" "$DIR/icons/pureedit.svg"
install -m 0644 "$ROOT_DIR/LICENSE" "$DIR/LICENSE"

# The same entries as in the .deb package; install.sh sets the absolute Exec paths.
install -m 0644 "$ROOT_DIR/build/linux/org.pureterm.PureTerm.desktop" "$DIR/org.pureterm.PureTerm.desktop"
# PureEdit, the text editor on its own (its Exec is pureedit); install.sh sets the absolute path.
install -m 0644 "$ROOT_DIR/build/linux/org.pureterm.PureEdit.desktop" "$DIR/org.pureterm.PureEdit.desktop"

mkdir -p "$ROOT_DIR/bin"
tar -C "$STAGE" --owner=0 --group=0 -czf "$ROOT_DIR/bin/$NAME.tar.gz" "$NAME"

echo "Linux archive ready: $ROOT_DIR/bin/$NAME.tar.gz"
