#!/usr/bin/env bash
#
# monkeytui installer.
#
# One-line install:
#   curl -fsSL https://raw.githubusercontent.com/ricardojparram/monkeytui/main/install.sh | bash
#
# Builds the binary with `go install` and places it on your PATH (in
# /usr/local/bin when writable, otherwise in your Go bin directory).

set -euo pipefail

MODULE="github.com/ricardojparram/monkeytui"
NAME="monkeytui"
VERSION="${MONKEYTUI_VERSION:-latest}"

YELLOW=$'\033[1;33m'; RED=$'\033[1;31m'; GREEN=$'\033[1;32m'; RESET=$'\033[0m'
info() { printf '%s==>%s %s\n' "$YELLOW" "$RESET" "$1"; }
ok()   { printf '%s✓%s %s\n'   "$GREEN"  "$RESET" "$1"; }
die()  { printf '%serror:%s %s\n' "$RED" "$RESET" "$1" >&2; exit 1; }

# 1. Require the Go toolchain.
command -v go >/dev/null 2>&1 || die "Go not found. Install Go 1.21+ from https://go.dev/dl and re-run."

GOVER="$(go env GOVERSION 2>/dev/null || echo unknown)"
info "Using $GOVER"

# 2. Build & install the binary into the Go bin directory.
info "Installing ${NAME} (${MODULE}@${VERSION})…"
GOBIN_DIR="$(go env GOBIN)"
[ -n "$GOBIN_DIR" ] || GOBIN_DIR="$(go env GOPATH)/bin"
GOFLAGS="" go install "${MODULE}@${VERSION}"

SRC="${GOBIN_DIR}/${NAME}"
[ -x "$SRC" ] || die "build finished but ${SRC} is missing."
ok "Built ${SRC}"

# 3. Try to place it on a system PATH directory for convenience.
DEST="$SRC"
SYS="/usr/local/bin"
if [ -w "$SYS" ]; then
  install -m 0755 "$SRC" "${SYS}/${NAME}" && DEST="${SYS}/${NAME}"
elif command -v sudo >/dev/null 2>&1 && [ -t 0 ]; then
  info "Copying to ${SYS} (needs sudo)…"
  sudo install -m 0755 "$SRC" "${SYS}/${NAME}" && DEST="${SYS}/${NAME}" || DEST="$SRC"
fi

ok "Installed ${NAME} → ${DEST}"

# 4. PATH hint if the install dir isn't already reachable.
case ":${PATH}:" in
  *":$(dirname "$DEST"):"*) : ;;
  *) info "Add this to your shell profile:  export PATH=\"$(dirname "$DEST"):\$PATH\"" ;;
esac

printf '\n%sRun it:%s %s\n' "$GREEN" "$RESET" "$NAME"
