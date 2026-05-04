#!/bin/bash
# Aurora CLI Installer — Mad God Inc.
# Usage: curl -sL https://raw.githubusercontent.com/madgodinc/aurora-cli/main/install.sh | bash

set -e

REPO="madgodinc/aurora-cli"
BIN_DIR="$HOME/bin"
AURORA_BIN="$BIN_DIR/aurora.exe"

echo ""
echo "  ♥ Aurora CLI Installer"
echo "  ──────────────────────"
echo ""

# Detect OS/arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$OS" in
  mingw*|msys*|cygwin*) OS="windows" ;;
  linux) OS="linux" ;;
  darwin) OS="darwin" ;;
esac

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
esac

echo "  Platform: ${OS}/${ARCH}"

# Get latest release
echo "  Fetching latest release..."
RELEASE_URL=$(curl -sL "https://api.github.com/repos/${REPO}/releases/latest" | grep "browser_download_url.*${OS}.*${ARCH}" | head -1 | cut -d '"' -f 4)

if [ -z "$RELEASE_URL" ]; then
  # No release yet — build from source
  echo "  No binary release found. Building from source..."

  # Check Go
  if ! command -v go &>/dev/null; then
    GO_PATH="/c/go/bin/go"
    if [ -f "$GO_PATH" ]; then
      export PATH="/c/go/bin:$PATH"
    else
      echo "  Error: Go not installed. Install from https://go.dev/dl/"
      exit 1
    fi
  fi

  TMPDIR=$(mktemp -d)
  echo "  Cloning repository..."
  git clone --depth 1 "https://github.com/${REPO}.git" "$TMPDIR/aurora-cli" 2>/dev/null

  cd "$TMPDIR/aurora-cli"
  echo "  Building..."
  go build -ldflags="-s -w" -o aurora.exe . 2>&1

  mkdir -p "$BIN_DIR"
  cp aurora.exe "$AURORA_BIN"
  rm -rf "$TMPDIR"
else
  # Download binary
  echo "  Downloading: $RELEASE_URL"
  mkdir -p "$BIN_DIR"
  curl -sL "$RELEASE_URL" -o "$AURORA_BIN"
  chmod +x "$AURORA_BIN"
fi

# Add to PATH if not already
if ! grep -q 'HOME/bin' ~/.bashrc 2>/dev/null; then
  echo 'export PATH="$HOME/bin:$PATH"' >> ~/.bashrc
  echo "  Added ~/bin to PATH in .bashrc"
fi

echo ""
echo "  ✓ Aurora CLI installed to $AURORA_BIN"
echo ""
echo "  To start: open a new Git Bash and type: aurora"
echo "  Or run now: export PATH=\"\$HOME/bin:\$PATH\" && aurora"
echo ""
