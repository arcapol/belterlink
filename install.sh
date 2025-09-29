#!/usr/bin/env bash
set -euo pipefail

# BelterLink installer (Option A: user binary + root-owned symlink)
# - Downloads latest release matching your OS/arch
# - Installs real binary to $USER_BIN (default: ~/.local/bin)
# - Creates a root-owned symlink at $SYS_BIN/belterlink (default: /usr/local/bin)
# - Requires sudo once for the symlink; future self-updates do NOT need sudo

REPO="${REPO:-arcapol/belterlink}"   # <-- set to your GitHub owner/repo
BIN="${BIN:-belterlink}"

USER_BIN="${USER_BIN:-$HOME/.local/bin}"
SYS_BIN="${SYS_BIN:-/usr/local/bin}"

have() { command -v "$1" >/dev/null 2>&1; }
die() { echo "ERROR: $*" >&2; exit 1; }

normalize_arch() {
  case "$1" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *) die "Unsupported arch: $1" ;;
  esac
}

normalize_os() {
  case "$1" in
    Darwin|darwin) echo "darwin" ;;
    Linux|linux) echo "linux" ;;
    *) die "Unsupported OS: $1" ;;
  esac
}

download() {
  local url="$1" out="$2"
  if have curl; then curl -fL "$url" -o "$out"
  elif have wget; then wget -O "$out" "$url"
  else die "Need curl or wget to download"
  fi
}

# Detect platform
OS="$(normalize_os "$(uname -s)")"
ARCH="$(normalize_arch "$(uname -m)")"
ASSET="${BIN}-${OS}-${ARCH}"

# Query latest release
API="https://api.github.com/repos/${REPO}/releases/latest"
if have curl; then JSON="$(curl -fsSL "$API")" || die "GitHub API query failed"
else JSON="$(wget -qO- "$API")" || die "GitHub API query failed"
fi

# Find asset URL (avoid jq dependency)
ASSET_URL="$(printf '%s\n' "$JSON" | grep -oE "https://[^\"]*/${ASSET}" | head -n1)"
# Fallback to redirect URL if grep missed
[ -n "${ASSET_URL:-}" ] || ASSET_URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"

echo "Installing ${BIN} (${OS}/${ARCH})"
echo "From:  $ASSET_URL"
echo "To:    $USER_BIN/${BIN}  (real binary)"
echo "Link:  $SYS_BIN/${BIN}    (symlink, requires sudo once)"

TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT
download "$ASSET_URL" "$TMP"
chmod +x "$TMP"

mkdir -p "$USER_BIN"
mv "$TMP" "${USER_BIN}/${BIN}"
chmod 755 "${USER_BIN}/${BIN}"

# Create/refresh root-owned symlink
sudo mkdir -p "$SYS_BIN"
sudo ln -sf "${USER_BIN}/${BIN}" "${SYS_BIN}/${BIN}"

# PATH hints
if ! echo ":$PATH:" | grep -q ":${SYS_BIN}:"; then
  echo "ℹ️  ${SYS_BIN} is not in your PATH. Add it or open a new shell."
fi

echo "✅ Installed:"
echo "   - Binary : ${USER_BIN}/${BIN}"
echo "   - Symlink: ${SYS_BIN}/${BIN}"

# Show version if supported
if "${SYS_BIN}/${BIN}" --version >/dev/null 2>&1; then
  "${SYS_BIN}/${BIN}" --version || true
fi

