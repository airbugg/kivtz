#!/bin/sh
# kivtz bootstrap — curl -fsSL airbugg.com/kivtz | sh
set -e

REPO="airbugg/kivtz"
INSTALL_DIR="$HOME/.local/bin"

echo ""
echo "  kivtz — cross-platform dotfiles manager"
echo ""

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
esac

echo "  platform: $OS/$ARCH"
mkdir -p "$INSTALL_DIR"

LATEST=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST" ]; then
    echo "  no release found — install Go and run: go install github.com/$REPO/cmd/kivtz@latest"
    exit 1
fi

URL="https://github.com/$REPO/releases/download/$LATEST/kivtz_${OS}_${ARCH}.tar.gz"
echo "  downloading kivtz $LATEST..."
curl -fsSL "$URL" | tar xz -C "$INSTALL_DIR" kivtz
chmod +x "$INSTALL_DIR/kivtz"
echo "  installed to $INSTALL_DIR/kivtz"

case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *) echo "  note: add $INSTALL_DIR to your PATH" ;;
esac

echo ""
echo "  run: kivtz init <your-dotfiles-repo-url>"
echo ""
