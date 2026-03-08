#!/bin/sh
set -e

REPO="sfphinx/tribaloutpost-auto-downloader"
BINARY="tribaloutpost-adl"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

OS="linux"

echo "Installing $BINARY for $OS/$ARCH..."

# Get latest release tag
LATEST=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | head -1 | cut -d'"' -f4)
if [ -z "$LATEST" ]; then
    echo "Error: could not determine latest release"
    exit 1
fi
echo "Latest version: $LATEST"

# Build download URL
VERSION=$(echo "$LATEST" | sed 's/^v//')
PROJECT="tribaloutpost-auto-downloader"
ARCHIVE="${PROJECT}-v${VERSION}-${OS}-${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/$LATEST/$ARCHIVE"

# Download and extract
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

echo "Downloading $URL..."
curl -fsSL "$URL" -o "$TMPDIR/$ARCHIVE"

echo "Extracting..."
tar -xzf "$TMPDIR/$ARCHIVE" -C "$TMPDIR"

# Install binary
mkdir -p "$INSTALL_DIR"
mv "$TMPDIR/$BINARY" "$INSTALL_DIR/$BINARY"
chmod +x "$INSTALL_DIR/$BINARY"

echo "Installed $BINARY to $INSTALL_DIR/$BINARY"

# Check if install dir is in PATH
case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *)
        echo ""
        echo "NOTE: $INSTALL_DIR is not in your PATH."
        echo "Add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
        echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
        ;;
esac

# Offer to enable autostart
echo ""
if [ -t 0 ]; then
    TTY=/dev/stdin
elif [ -e /dev/tty ]; then
    TTY=/dev/tty
else
    TTY=""
fi

if [ -n "$TTY" ]; then
    printf "Enable automatic startup on login? [y/N] "
    read -r REPLY < "$TTY"
else
    REPLY=""
fi
case "$REPLY" in
    [yY]*)
        "$INSTALL_DIR/$BINARY" autostart enable
        echo "Autostart enabled."
        ;;
    *)
        echo "You can enable autostart later with: $BINARY autostart enable"
        ;;
esac

echo ""
echo "Installation complete! Run '$BINARY' to start."
