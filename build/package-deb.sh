#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

VERSION="${1:-$(tr -d '[:space:]' < "$ROOT_DIR/VERSION")}"
ARCH="amd64"
PKG_NAME="pureterm"
DEB_NAME="${PKG_NAME}_${VERSION}_${ARCH}.deb"

cd "$ROOT_DIR"
# The Gio GUI with the libghostty engine: build libghostty first
# (build/gio/build-libghostty.sh) and set PKG_CONFIG_PATH to its prefix.
# PURETERM_TAGS= builds without libghostty (the vt engine only).
TAGS="${PURETERM_TAGS-ghostty}"
echo "Building PureTerm binary..."
go build ${TAGS:+-tags "$TAGS"} -trimpath -ldflags="-s -w" -o "$ROOT_DIR/bin/pureterm" ./cmd/pureterm

DEB_ROOT="$ROOT_DIR/build/deb-root"
rm -rf "$DEB_ROOT"
mkdir -p "$DEB_ROOT/DEBIAN"
mkdir -p "$DEB_ROOT/usr/bin"
mkdir -p "$DEB_ROOT/usr/share/applications"
for size in 16 24 32 48 64 128 256 512; do
    mkdir -p "$DEB_ROOT/usr/share/icons/hicolor/${size}x${size}/apps"
done
mkdir -p "$DEB_ROOT/usr/share/icons/hicolor/scalable/apps"

chmod 0755 "$DEB_ROOT"
chmod 0755 "$DEB_ROOT/DEBIAN"

# 1. Binary
install -m 0755 "$ROOT_DIR/bin/pureterm" "$DEB_ROOT/usr/bin/pureterm"

# 2. Desktop entries. Their names are the application ids of the windows
# (Wayland app_id, X11 WM_CLASS = StartupWMClass): the desktop finds the icon
# and keeps PureTerm and PureEdit apart in the taskbar.
sed "s|^Exec=pureterm|Exec=/usr/bin/pureterm|" "$ROOT_DIR/build/linux/org.pureterm.PureTerm.desktop" \
    > "$DEB_ROOT/usr/share/applications/org.pureterm.PureTerm.desktop"
chmod 0644 "$DEB_ROOT/usr/share/applications/org.pureterm.PureTerm.desktop"

# PureEdit: the text editor on its own (own menu entry, "Open with" for text files)
install -m 0755 "$ROOT_DIR/build/linux/pureedit" "$DEB_ROOT/usr/bin/pureedit"
sed "s|^Exec=pureedit|Exec=/usr/bin/pureedit|" "$ROOT_DIR/build/linux/org.pureterm.PureEdit.desktop" \
    > "$DEB_ROOT/usr/share/applications/org.pureterm.PureEdit.desktop"
chmod 0644 "$DEB_ROOT/usr/share/applications/org.pureterm.PureEdit.desktop"

# 3. Icons
for size in 16 24 32 48 64 128 256 512; do
    install -m 0644 "$ROOT_DIR/build/linux/icons/pureterm-${size}.png" "$DEB_ROOT/usr/share/icons/hicolor/${size}x${size}/apps/pureterm.png"
    install -m 0644 "$ROOT_DIR/build/linux/icons/pureedit-${size}.png" "$DEB_ROOT/usr/share/icons/hicolor/${size}x${size}/apps/pureedit.png"
done
install -m 0644 "$ROOT_DIR/PureTerm_Icon.svg" "$DEB_ROOT/usr/share/icons/hicolor/scalable/apps/pureterm.svg"
install -m 0644 "$ROOT_DIR/PureEdit_Icon.svg" "$DEB_ROOT/usr/share/icons/hicolor/scalable/apps/pureedit.svg"

# 4. Control file
INSTALLED_SIZE=$(du -sk --exclude=DEBIAN "$DEB_ROOT" | cut -f1)
cat <<EOF > "$DEB_ROOT/DEBIAN/control"
Package: ${PKG_NAME}
Version: ${VERSION}
Section: net
Priority: optional
Architecture: ${ARCH}
Maintainer: PureTerm Project <info@pureterm.org>
Depends: libc6 (>= 2.34), libegl1, libgles2, libwayland-client0, libwayland-cursor0, libwayland-egl1, libxkbcommon0, libxkbcommon-x11-0, libx11-6, libx11-xcb1, libxcursor1, libxfixes3, libxrender1, libffi8
Conflicts: linxterm, leanterm
Replaces: linxterm, leanterm
Recommends: xdg-utils, openssh-client, keepassxc
Homepage: https://github.com/Suvukaraz/PureTerm
Installed-Size: ${INSTALLED_SIZE}
Description: PureTerm Connection Manager
 SSH, Telnet and SFTP connection manager for Linux and Windows. Native SSH
 (jump hosts, port forwarding, agent, X11) and Telnet, tabbed terminals, an
 SFTP and local file browser, a built-in text editor that edits remote files
 directly, Expect automation, variables, clusters and KeePassXC integration.
EOF
chmod 0644 "$DEB_ROOT/DEBIAN/control"

# 5. Maintainer scripts
cat <<EOF > "$DEB_ROOT/DEBIAN/postinst"
#!/bin/sh
set -e
if which update-desktop-database >/dev/null 2>&1; then
    update-desktop-database -q || true
fi
if which gtk-update-icon-cache >/dev/null 2>&1; then
    gtk-update-icon-cache -q -t -f /usr/share/icons/hicolor || true
fi
exit 0
EOF
chmod 0755 "$DEB_ROOT/DEBIAN/postinst"

cat <<EOF > "$DEB_ROOT/DEBIAN/postrm"
#!/bin/sh
set -e
if which update-desktop-database >/dev/null 2>&1; then
    update-desktop-database -q || true
fi
if which gtk-update-icon-cache >/dev/null 2>&1; then
    gtk-update-icon-cache -q -t -f /usr/share/icons/hicolor || true
fi
exit 0
EOF
chmod 0755 "$DEB_ROOT/DEBIAN/postrm"

find "$DEB_ROOT" -type d -exec chmod 0755 {} +

mkdir -p "$ROOT_DIR/bin"
dpkg-deb --build --root-owner-group "$DEB_ROOT" "$ROOT_DIR/bin/$DEB_NAME"
rm -rf "$DEB_ROOT"

echo "Debian package successfully built: $ROOT_DIR/bin/$DEB_NAME"
