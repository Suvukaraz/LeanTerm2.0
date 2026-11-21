#!/usr/bin/env bash
# Builds the Windows installer (NSIS, amd64) by cross-compiling from Linux:
# leanterm.exe and leanedit.exe, the same program with their own icon and
# version information (LeanEdit is an application of its own in Explorer,
# "Open with" and the taskbar; leanedit.exe runs the editor).
#
# Needs Go, go-winres (GO_WINRES, default: $(go env GOPATH)/bin/go-winres)
# and makensis (Debian/Ubuntu: apt install nsis). With the libghostty engine
# (the default, LEANTERM_TAGS=ghostty) also Zig (ZIG, default: zig; the CI
# installs it with pip) and PKG_CONFIG_PATH pointing to the Windows
# libghostty-vt, built with
#   build/gio/build-libghostty.sh x86_64-windows-gnu <prefix>
# LEANTERM_TAGS= builds with the pure Go engine only, without cgo.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

VERSION="${1:-$(tr -d '[:space:]' < "$ROOT_DIR/VERSION")}"
SETUP_NAME="leanterm_${VERSION}_windows_amd64_setup.exe"
ZIG="${ZIG:-zig}"
GO_WINRES="${GO_WINRES:-$(go env GOPATH)/bin/go-winres}"
NSIS_DIR="$ROOT_DIR/build/windows/nsis"
TAGS="${LEANTERM_TAGS-ghostty}"

if ! command -v makensis >/dev/null 2>&1; then
    echo "makensis not found. Install NSIS first, e.g.: sudo apt install nsis" >&2
    exit 1
fi
if [ ! -x "$GO_WINRES" ]; then
    echo "go-winres not found: go install github.com/tc-hib/go-winres@v0.3.3" >&2
    exit 1
fi
if [ -n "$TAGS" ] && [ -z "${PKG_CONFIG_PATH:-}" ]; then
    echo "PKG_CONFIG_PATH must point to the Windows libghostty-vt (see build/gio/build-libghostty.sh)" >&2
    exit 1
fi
case "$VERSION" in
    [0-9]*.[0-9]*.[0-9]*) ;;
    *) echo "VERSION must be X.Y.Z (got '$VERSION')" >&2; exit 1 ;;
esac

cd "$ROOT_DIR"
WORK_DIR="$(mktemp -d)"
# The resource object must sit in the package folder while Go links it.
SYSO="$ROOT_DIR/cmd/leanterm/rsrc_windows_amd64.syso"
trap 'rm -rf "$WORK_DIR" "$SYSO"' EXIT

# winres writes the resources of one executable: icon 1 is its own (the
# window class and Explorer take it), icon 2 is LeanEdit's (LeanEdit's
# windows inside LeanTerm and pinned LeanEdit windows use it), the manifest
# and the version information.
winres() { # name description own-icon
    # go-winres takes file names relative to the JSON file.
    cp "$3" "$WORK_DIR/own.ico"
    cp "$ROOT_DIR/build/windows/leanedit.ico" "$WORK_DIR/leanedit.ico"
    cat > "$WORK_DIR/winres.json" <<EOF
{
  "RT_GROUP_ICON": {
    "#1": {"0000": "own.ico"},
    "#2": {"0000": "leanedit.ico"}
  },
  "RT_MANIFEST": {
    "#1": {"0409": {
      "description": "$2",
      "minimum-os": "win10",
      "execution-level": "as invoker",
      "dpi-awareness": "per monitor v2",
      "long-path-aware": true,
      "use-common-controls-v6": true
    }}
  },
  "RT_VERSION": {
    "#1": {"0000": {
      "fixed": {"file_version": "$VERSION.0", "product_version": "$VERSION.0"},
      "info": {"0409": {
        "CompanyName": "LeanTerm Project",
        "FileDescription": "$2",
        "FileVersion": "$VERSION",
        "InternalName": "$1",
        "LegalCopyright": "© 2026 LeanTerm Project, GPL-3.0",
        "OriginalFilename": "$1.exe",
        "ProductName": "$2",
        "ProductVersion": "$VERSION"
      }}
    }}
  }
}
EOF
    "$GO_WINRES" make --in "$WORK_DIR/winres.json" --arch amd64 --out "$ROOT_DIR/cmd/leanterm/rsrc"
}

build() { # name
    if [ -n "$TAGS" ]; then
        GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
            CC="$ZIG cc -target x86_64-windows-gnu" CXX="$ZIG c++ -target x86_64-windows-gnu" \
            CGO_LDFLAGS_ALLOW='.*[.]lib$' \
            go build -tags "$TAGS" -trimpath -buildvcs=false \
            -ldflags="-w -s -H windowsgui" -o "$WORK_DIR/$1.exe" ./cmd/leanterm
    else
        GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
            go build -trimpath -buildvcs=false \
            -ldflags="-w -s -H windowsgui" -o "$WORK_DIR/$1.exe" ./cmd/leanterm
    fi
    rm -f "$SYSO"
}

echo "Building leanterm.exe (${TAGS:-vt only})..."
winres leanterm "LeanTerm" "$ROOT_DIR/build/windows/icon.ico"
build leanterm
echo "Building leanedit.exe..."
winres leanedit "LeanEdit" "$ROOT_DIR/build/windows/leanedit.ico"
build leanedit

echo "Creating installer..."
mkdir -p "$ROOT_DIR/bin"
(cd "$NSIS_DIR" && makensis -V2 \
    -DINFO_PRODUCTVERSION="$VERSION" \
    -DARG_BINARY="$WORK_DIR/leanterm.exe" \
    -DARG_LEANEDIT="$WORK_DIR/leanedit.exe" \
    -DOUTFILE="$ROOT_DIR/bin/$SETUP_NAME" \
    gio.nsi)

echo "Windows installer ready: $ROOT_DIR/bin/$SETUP_NAME"
