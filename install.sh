#!/bin/bash
# Aurora CLI Installer — Mad God Inc.
# Usage: curl -sL https://raw.githubusercontent.com/madgodinc/aurora-cli/main/install.sh | bash

set -e

REPO="madgodinc/aurora-cli"
BIN_DIR="$HOME/bin"
GO_VERSION="1.24.3"

echo ""
echo "  ♥ Aurora CLI Installer"
echo "  ──────────────────────"
echo ""

# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$OS" in
  mingw*|msys*|cygwin*) OS="windows"; EXT=".exe" ;;
  linux) OS="linux"; EXT="" ;;
  darwin) OS="darwin"; EXT="" ;;
  *) echo "  Unsupported OS: $OS"; exit 1 ;;
esac
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "  Unsupported arch: $ARCH"; exit 1 ;;
esac
echo "  Platform: ${OS}/${ARCH}"

# ── Check/Install Go ──
install_go() {
  echo "  Installing Go ${GO_VERSION}..."
  GO_URL="https://go.dev/dl/go${GO_VERSION}.${OS}-${ARCH}"

  if [ "$OS" = "windows" ]; then
    GO_URL="${GO_URL}.zip"
    TMPFILE=$(mktemp).zip
    curl -sL "$GO_URL" -o "$TMPFILE"
    echo "  Extracting Go..."
    unzip -q -o "$TMPFILE" -d /c/ 2>/dev/null
    rm -f "$TMPFILE"
    export PATH="/c/go/bin:$PATH"
  else
    GO_URL="${GO_URL}.tar.gz"
    TMPFILE=$(mktemp).tar.gz
    curl -sL "$GO_URL" -o "$TMPFILE"
    echo "  Extracting Go (needs sudo)..."
    sudo tar -C /usr/local -xzf "$TMPFILE"
    rm -f "$TMPFILE"
    export PATH="/usr/local/go/bin:$PATH"
  fi
  echo "  Go $(go version | awk '{print $3}') installed"
}

GO_BIN=""
if command -v go &>/dev/null; then
  GO_BIN="go"
elif [ -f "/c/go/bin/go.exe" ]; then
  export PATH="/c/go/bin:$PATH"
  GO_BIN="go"
elif [ -f "/usr/local/go/bin/go" ]; then
  export PATH="/usr/local/go/bin:$PATH"
  GO_BIN="go"
fi

if [ -z "$GO_BIN" ] || ! command -v go &>/dev/null; then
  echo "  Go not found."
  read -p "  Install Go ${GO_VERSION}? [Y/n]: " yn
  case "$yn" in
    [nN]*) echo "  Go required to build Aurora CLI."; exit 1 ;;
    *) install_go ;;
  esac
fi

echo "  Go: $(go version | awk '{print $3}')"

# ── Check Git ──
if ! command -v git &>/dev/null; then
  echo "  Error: Git not installed. Install Git for Windows from https://git-scm.com"
  exit 1
fi

# ── Clone and build ──
TMPDIR=$(mktemp -d)
echo "  Cloning aurora-cli..."
git clone --depth 1 "https://github.com/${REPO}.git" "$TMPDIR/aurora-cli" 2>/dev/null

cd "$TMPDIR/aurora-cli"
echo "  Building (this may take a minute on first run)..."
go build -ldflags="-s -w" -o "aurora${EXT}" . 2>&1

# ── Install binary ──
mkdir -p "$BIN_DIR"
cp "aurora${EXT}" "$BIN_DIR/aurora${EXT}"
chmod +x "$BIN_DIR/aurora${EXT}"

# ── Cleanup ──
cd "$HOME"
rm -rf "$TMPDIR"

# ── Add to PATH ──
if ! echo "$PATH" | grep -q "$HOME/bin"; then
  SHELL_RC=""
  if [ -f "$HOME/.bashrc" ]; then
    SHELL_RC="$HOME/.bashrc"
  elif [ -f "$HOME/.zshrc" ]; then
    SHELL_RC="$HOME/.zshrc"
  elif [ -f "$HOME/.profile" ]; then
    SHELL_RC="$HOME/.profile"
  fi

  if [ -n "$SHELL_RC" ]; then
    if ! grep -q 'HOME/bin' "$SHELL_RC" 2>/dev/null; then
      echo 'export PATH="$HOME/bin:$PATH"' >> "$SHELL_RC"
      echo "  Added ~/bin to PATH in $(basename $SHELL_RC)"
    fi
  fi
  export PATH="$HOME/bin:$PATH"
fi

# ── Verify ──
echo ""
INSTALLED_VERSION=$(aurora${EXT} --version 2>&1 | head -1)
echo "  ✓ $INSTALLED_VERSION"
echo ""
echo "  To start Aurora:"
echo "    aurora              (interactive TUI)"
echo "    aurora -p \"hello\"   (single query)"
echo ""
echo "  First run will ask for server connection settings."
echo "  ♥ Enjoy! — Mad God Inc."
echo ""
