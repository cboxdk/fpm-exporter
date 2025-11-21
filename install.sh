#!/bin/sh
set -e

# PHPeek PHP-FPM Exporter Installer
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/gophpeek/phpeek-fpm-exporter/main/install.sh | sh
#
# With version pinning:
#   curl -fsSL https://raw.githubusercontent.com/gophpeek/phpeek-fpm-exporter/main/install.sh | VERSION=v1.2.0 sh
#
# Custom install directory:
#   curl -fsSL ... | INSTALL_DIR=/opt/bin sh

REPO="gophpeek/phpeek-fpm-exporter"
BINARY="phpeek-fpm-exporter"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="${VERSION:-latest}"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    linux) OS="linux" ;;
    darwin) OS="darwin" ;;
    *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

FILENAME="${BINARY}-${OS}-${ARCH}"

# Build download URL
if [ "$VERSION" = "latest" ]; then
    URL="https://github.com/${REPO}/releases/latest/download/${FILENAME}"
else
    URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}"
fi

echo "Downloading ${BINARY} ${VERSION} for ${OS}/${ARCH}..."
if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$URL" -o "/tmp/${BINARY}"
elif command -v wget >/dev/null 2>&1; then
    wget -q "$URL" -O "/tmp/${BINARY}"
else
    echo "Error: curl or wget required"
    exit 1
fi

chmod +x "/tmp/${BINARY}"

# Install (may need sudo)
if [ -w "$INSTALL_DIR" ]; then
    mv "/tmp/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
    echo "Installing to ${INSTALL_DIR} (requires sudo)..."
    sudo mv "/tmp/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

echo "Installed ${BINARY} to ${INSTALL_DIR}/${BINARY}"
echo ""
echo "Run: ${BINARY} serve"
