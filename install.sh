#!/usr/bin/env bash
#
# monkeytui installer.
#
# One-line install (no Go required — downloads a prebuilt binary):
#   curl -fsSL https://raw.githubusercontent.com/ricardojparram/monkeytui/main/install.sh | bash
#
# Downloads the prebuilt binary for your OS/arch from the latest GitHub
# release and puts it on your PATH. If no prebuilt binary matches your
# platform, it falls back to building from source with `go install`.

set -euo pipefail

REPO="ricardojparram/monkeytui"
NAME="monkeytui"
BASE="https://github.com/${REPO}/releases/latest/download"

YELLOW=$'\033[1;33m'; RED=$'\033[1;31m'; GREEN=$'\033[1;32m'; RESET=$'\033[0m'
info() { printf '%s==>%s %s\n' "$YELLOW" "$RESET" "$1"; }
ok()   { printf '%s✓%s %s\n'   "$GREEN"  "$RESET" "$1"; }
die()  { printf '%serror:%s %s\n' "$RED" "$RESET" "$1" >&2; exit 1; }

DEST=""

# --- detect platform ---------------------------------------------------------
os="$(uname -s)"; arch="$(uname -m)"
case "$os" in
  Linux)  os="linux"  ;;
  Darwin) os="darwin" ;;
  *)      os="unsupported" ;;
esac
case "$arch" in
  x86_64|amd64)  arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *)             arch="unsupported" ;;
esac

# --- pick an install directory (no sudo needed) ------------------------------
install_dir() {
  if [ -w "/usr/local/bin" ]; then
    printf '%s' "/usr/local/bin"
  else
    mkdir -p "$HOME/.local/bin"
    printf '%s' "$HOME/.local/bin"
  fi
}

fetch() { # url out
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$1" -o "$2"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$2" "$1"
  else
    die "need curl or wget to download."
  fi
}

# --- try prebuilt binary first ----------------------------------------------
download_prebuilt() {
  [ "$os" != "unsupported" ] && [ "$arch" != "unsupported" ] || return 1
  asset="${NAME}_${os}_${arch}"
  url="${BASE}/${asset}"
  tmp="$(mktemp)"
  info "Downloading prebuilt ${asset}…"
  if fetch "$url" "$tmp"; then
    dir="$(install_dir)"
    chmod +x "$tmp"
    mv "$tmp" "${dir}/${NAME}"
    DEST="${dir}/${NAME}"
    return 0
  fi
  rm -f "$tmp"
  return 1
}

# --- fall back to building from source with Go ------------------------------
build_from_source() {
  command -v go >/dev/null 2>&1 || return 1
  info "No prebuilt binary for ${os}/${arch}; building from source with Go…"
  gobin="$(go env GOBIN)"; [ -n "$gobin" ] || gobin="$(go env GOPATH)/bin"
  GOFLAGS="" go install "github.com/${REPO}@latest"
  DEST="${gobin}/${NAME}"
  [ -x "$DEST" ] || return 1
  return 0
}

if download_prebuilt || build_from_source; then
  ok "Installed ${NAME} → ${DEST}"
else
  die "No prebuilt binary for ${os}/${arch} and Go is not installed.
  Install Go from https://go.dev/dl, or grab a binary from
  https://github.com/${REPO}/releases"
fi

# --- PATH hint ---------------------------------------------------------------
case ":${PATH}:" in
  *":$(dirname "$DEST"):"*) : ;;
  *) info "Add this to your shell profile:  export PATH=\"$(dirname "$DEST"):\$PATH\"" ;;
esac

printf '\n%sRun it:%s %s\n' "$GREEN" "$RESET" "$NAME"
