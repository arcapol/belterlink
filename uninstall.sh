#!/usr/bin/env bash
set -euo pipefail

# BelterLink uninstaller (Option A)
# - Removes root-owned symlink at /usr/local/bin/belterlink (or $SYS_BIN)
# - Removes user binary at ~/.local/bin/belterlink (or $USER_BIN)
# - Optionally removes config at ~/.config/belterlink

BIN_NAME="${BIN_NAME:-belterlink}"
USER_BIN="${USER_BIN:-$HOME/.local/bin}"
SYS_BIN="${SYS_BIN:-/usr/local/bin}"
CONFIG_DIR="${CONFIG_DIR:-$HOME/.config/$BIN_NAME}"

echo "🔗 Uninstalling ${BIN_NAME} ..."

# Remove symlink (may require sudo)
SYMLINK_PATH="${SYS_BIN}/${BIN_NAME}"
if [ -L "$SYMLINK_PATH" ] || [ -f "$SYMLINK_PATH" ]; then
  echo "Removing symlink: $SYMLINK_PATH"
  sudo rm -f "$SYMLINK_PATH"
else
  echo "ℹ️  Symlink not found at ${SYMLINK_PATH}"
fi

# Remove user binary
USER_BIN_PATH="${USER_BIN}/${BIN_NAME}"
if [ -f "$USER_BIN_PATH" ]; then
  echo "Removing binary : $USER_BIN_PATH"
  rm -f "$USER_BIN_PATH"
else
  echo "ℹ️  Binary not found at ${USER_BIN_PATH}"
fi

# Optional: remove config
if [ -d "$CONFIG_DIR" ]; then
  read -rp "Do you also want to remove config at ${CONFIG_DIR}? [y/N]: " confirm
  if [[ "${confirm:-N}" =~ ^[Yy]$ ]]; then
    rm -rf "$CONFIG_DIR"
    echo "🗑️  Config removed."
  else
    echo "ℹ️  Config preserved."
  fi
fi

echo "✅ Uninstall complete."
