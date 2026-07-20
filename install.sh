#!/bin/bash
# prism-switch installer
# Usage: curl -fsSL https://raw.githubusercontent.com/chiga0/prism-switch/main/install.sh | bash
set -e

REPO="chiga0/prism-switch"
INSTALL_DIR="/usr/local/bin"
BINARY="prism"

# Detect OS and architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
  darwin|linux) ;;
  *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Get latest version
echo "==> Detecting latest version..."
VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed 's/.*"v\(.*\)".*/\1/')
if [ -z "$VERSION" ]; then
  echo "Error: could not determine latest version"
  exit 1
fi
echo "==> Latest version: v$VERSION"

# Download
FILENAME="prism-switch_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/v${VERSION}/$FILENAME"
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

echo "==> Downloading $FILENAME..."
curl -fsSL "$URL" -o "$TMP_DIR/$FILENAME"

# Extract and install
echo "==> Installing to $INSTALL_DIR/$BINARY..."
tar -xzf "$TMP_DIR/$FILENAME" -C "$TMP_DIR"

if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP_DIR/$BINARY" "$INSTALL_DIR/$BINARY"
  chmod +x "$INSTALL_DIR/$BINARY"
else
  echo "==> Requires sudo to install to $INSTALL_DIR"
  sudo mv "$TMP_DIR/$BINARY" "$INSTALL_DIR/$BINARY"
  sudo chmod +x "$INSTALL_DIR/$BINARY"
fi

echo ""
echo "✓ prism v$VERSION installed to $INSTALL_DIR/$BINARY"
echo ""
echo "Get started:"
echo "  prism init          # create config"
echo "  prism detect        # auto-detect installed agents"
echo "  prism sync          # sync providers to agents"
