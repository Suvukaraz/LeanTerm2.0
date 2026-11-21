<p align="center"><img src="LeanTerm_Icon.svg" alt="LeanTerm logo" width="112"></p>

<h1 align="center">LeanTerm</h1>

<p align="center">
  <b>SSH, Telnet and SFTP connection manager for Linux and Windows</b><br>
  Connection tree, tabbed terminals, a file browser and a built-in editor for remote files, in one lean native application.
</p>

<p align="center">
  <a href="https://github.com/Suvukaraz/LeanTerm2.0/releases/latest"><img alt="Latest release" src="https://img.shields.io/github/v/release/Suvukaraz/LeanTerm2.0"></a>
  <img alt="Platforms" src="https://img.shields.io/badge/platform-Linux%20%7C%20Windows-blue">
  <a href="LICENSE"><img alt="License: GPL-3.0" src="https://img.shields.io/github/license/Suvukaraz/LeanTerm2.0"></a>
  <a href="https://github.com/Suvukaraz/LeanTerm2.0/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/Suvukaraz/LeanTerm2.0/actions/workflows/ci.yml/badge.svg"></a>
</p>

LeanTerm is written in Go with the [Gio](https://gioui.org) GUI toolkit. It keeps your saved connections in a tree, opens SSH, Telnet and local shells in tabs, browses the files of the remote or local side next to the terminal, and edits remote files directly in its built-in editor, LeanEdit.

## Installation

**Linux (x86_64)**: one command for Debian, Ubuntu, Fedora, openSUSE, Arch and their derivatives:

```sh
curl -fsSL https://raw.githubusercontent.com/Suvukaraz/LeanTerm2.0/main/install.sh | sh
```

The script installs the `.deb` package on Debian-based systems and the release archive (with the libraries it needs) elsewhere. It checks every download against the release checksums. Running it again updates LeanTerm. Options are passed after `sh -s --`:

```sh
curl -fsSL https://raw.githubusercontent.com/Suvukaraz/LeanTerm2.0/main/install.sh | sh -s -- --version 2.0.0
curl -fsSL https://raw.githubusercontent.com/Suvukaraz/LeanTerm2.0/main/install.sh | sh -s -- --uninstall
```

**Windows 10 (1809) or later**: run `leanterm_2.0.0_windows_amd64_setup.exe` from the [releases page](https://github.com/Suvukaraz/LeanTerm2.0/releases/latest). The installer puts LeanTerm into `C:\Program Files\LeanTerm` and adds Start menu and desktop shortcuts; LeanEdit can be selected as well (`leanedit.exe`, with its own taskbar entry and "Open with" entry). The installer is not code-signed yet, so SmartScreen asks for confirmation on the first start.

**Packages** (all on the [releases page](https://github.com/Suvukaraz/LeanTerm2.0/releases/latest), with `SHA256SUMS`):

| File | Contents |
|---|---|
| `leanterm_2.0.0_amd64.deb` | Debian/Ubuntu package with LeanTerm and LeanEdit |
| `leanterm_2.0.0_linux_amd64.tar.gz` | Archive for other distributions, used by `install.sh` |
| `leanterm_2.0.0_windows_amd64_setup.exe` | Windows installer |

The Linux binaries need glibc 2.34 or newer and the EGL, Wayland, X11 and xkbcommon libraries.

## Features

- **Connections** – SSH (jump hosts, agent, keyboard-interactive and OTP prompts, local, remote and dynamic port forwarding, X11 forwarding), Telnet and local shells, with the protocols implemented in Go. No external `ssh` or `sftp` program is needed.
- **Tree and organisation** – connections in folders with favourites and history, quick connect, bulk edit, Wake-on-LAN, and variables and prompts in connection settings.
- **Terminals** – tabs and split tabs, a find bar, screenshots and session logs. Local shells use a PTY on Linux and ConPTY on Windows.
- **Automation** – Expect-style sequences, pre and post commands, macros, clusters with broadcast input, and piping or teeing a terminal's output to a file or command.
- **File browser** – SFTP for SSH connections and the local file system for local shells, with drag and drop (files from the desktop into the folder under the pointer, rows onto folders to move them), a transfer queue, a folder that follows the terminal (OSC 7) and opening files in the editor.
- **LeanEdit** – the text editor, for local and remote files: syntax highlighting for many languages, code folding, word completion, Go to Symbol, bookmarks, a minimap, and reloading of files changed by other programs. See [LeanEdit](#leanedit).
- **Security** – secrets are encrypted with [age](https://age-encryption.org) and kept in the system keyring or behind a master password, KeePassXC integration, and a GUI lock.

## LeanEdit

LeanEdit is the text editor of LeanTerm as an application of its own. It has a menu entry and an icon, is offered in *Open with* for text files and runs independently of LeanTerm:

```sh
leanedit [file…]        # the same as: leanterm -leanedit [file…]
```

LeanEdit is a separate application in the taskbar (KDE, GNOME and Windows), also when LeanTerm opens it for remote files; files dropped on its window open as tabs. By default LeanEdit runs once: further files open as tabs in the running window, which comes to the front. *File → New Window* opens another window. When the window gets the focus, LeanEdit offers to reload files that were changed or deleted by another program, and saving never overwrites such a change without asking. Started on its own, LeanEdit edits local files and needs no connection.

## Configuration

Settings are edited in the application and stored as TOML in `config.toml` in the configuration folder: `~/.config/leanterm` on Linux, `%APPDATA%\LeanTerm` on Windows (`--config-dir` chooses another one). The window layout is kept in `state/ui.toml`; passwords are kept separately in the encrypted `secrets.age`. All options with their defaults are listed in [docs/config-reference.md](docs/config-reference.md).

Start options such as `--start-uuid`, `--start-shell`, `--edit <file>`, `--leanedit` and `--config-dir` are listed by `leanterm --help`.

## Building from source

Requirements: Go 1.26 or newer. On Linux the GUI needs the development packages of Wayland, X11, EGL, GLES, xkbcommon, Xcursor and libffi (Debian/Ubuntu: `sudo apt install libwayland-dev libx11-dev libx11-xcb-dev libxkbcommon-x11-dev libgles2-mesa-dev libegl1-mesa-dev libffi-dev libxcursor-dev libvulkan-dev`).

```sh
go build -o bin/leanterm ./cmd/leanterm
```

Gio and gvcode are vendored with small changes in `third_party/` (see `third_party/gio/LEANTERM.md`): drag and drop from file managers on X11 and Wayland, per-window application ids and icons, and window activation. This build uses LeanTerm's pure Go terminal engine (`vt`). The packages and the CI use the Ghostty terminal core (libghostty-vt) as well, which is built with Zig 0.17.1 and selected with the `ghostty` build tag:

```sh
build/gio/build-libghostty.sh x86_64-linux-gnu.2.34 "$PWD/ghostty/linux"
PKG_CONFIG_PATH="$PWD/ghostty/linux/lib/pkgconfig" go build -tags ghostty -o bin/leanterm ./cmd/leanterm
```

The script needs Zig 0.17.1 (`pip install ziglang==0.17.1` provides it) and clones the Ghostty source at the commit the bindings are built against. Packages are built by the scripts in `build/`, with the version from the `VERSION` file unless one is given:

| Package | Command |
|---|---|
| Debian package | `build/package-deb.sh` |
| Linux archive | `build/package-tarball.sh` |
| Windows installer with `leanterm.exe` and `leanedit.exe` (cross-compiled from Linux; needs `nsis`, `go-winres` and, for the Ghostty engine, Zig and the Windows libghostty; `LEANTERM_TAGS=` builds without them) | `build/package-windows.sh` |

Tests: `go test ./...` (or `make test`). `make build` builds `bin/leanterm` without libghostty.

## License

LeanTerm is licensed under the [GNU General Public License v3.0](LICENSE).
