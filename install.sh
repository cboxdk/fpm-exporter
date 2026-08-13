#!/bin/sh
set -e

# Cbox FPM Exporter Installer
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/cboxdk/fpm-exporter/main/install.sh | sh
#
# With version pinning:
#   curl -fsSL https://raw.githubusercontent.com/cboxdk/fpm-exporter/main/install.sh | VERSION=v1.2.0 sh
#
# Custom install directory:
#   curl -fsSL ... | INSTALL_DIR=/opt/bin sh

REPO="cboxdk/fpm-exporter"
BINARY="fpm-exporter"
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

# Build download URLs
if [ "$VERSION" = "latest" ]; then
    BASE_URL="https://github.com/${REPO}/releases/latest/download"
else
    BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
fi

URL="${BASE_URL}/${FILENAME}"
CHECKSUM_URL="${BASE_URL}/checksums.txt"

# A private temporary directory, so nothing can race us to a predictable path.
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

download() {
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$1" -o "$2"
    elif command -v wget >/dev/null 2>&1; then
        wget -q "$1" -O "$2"
    else
        echo "Error: curl or wget required"
        exit 1
    fi
}

echo "Downloading ${BINARY} ${VERSION} for ${OS}/${ARCH}..."
download "$URL" "${TMP_DIR}/${FILENAME}"

# Verify the download against the release checksums. Releases before v3.2.0 do
# not publish them, so a missing file is a warning rather than a failure.
if download "$CHECKSUM_URL" "${TMP_DIR}/checksums.txt" 2>/dev/null; then
    if command -v sha256sum >/dev/null 2>&1; then
        SHA_CMD="sha256sum"
    elif command -v shasum >/dev/null 2>&1; then
        SHA_CMD="shasum -a 256"
    fi

    if [ -n "${SHA_CMD:-}" ]; then
        EXPECTED=$(grep " ${FILENAME}\$" "${TMP_DIR}/checksums.txt" | awk '{print $1}')
        ACTUAL=$(cd "$TMP_DIR" && $SHA_CMD "$FILENAME" | awk '{print $1}')

        if [ -z "$EXPECTED" ]; then
            echo "Warning: no checksum listed for ${FILENAME}, skipping verification"
        elif [ "$EXPECTED" != "$ACTUAL" ]; then
            echo "Error: checksum mismatch for ${FILENAME}"
            echo "  expected: $EXPECTED"
            echo "  actual:   $ACTUAL"
            exit 1
        else
            echo "Checksum verified."
        fi
    else
        echo "Warning: no sha256sum or shasum available, skipping verification"
    fi
else
    echo "Warning: no checksums published for ${VERSION}, skipping verification"
fi

chmod +x "${TMP_DIR}/${FILENAME}"

# Install (may need sudo)
if [ -w "$INSTALL_DIR" ]; then
    mv "${TMP_DIR}/${FILENAME}" "${INSTALL_DIR}/${BINARY}"
else
    echo "Installing to ${INSTALL_DIR} (requires sudo)..."
    sudo mv "${TMP_DIR}/${FILENAME}" "${INSTALL_DIR}/${BINARY}"
fi

echo "Installed ${BINARY} to ${INSTALL_DIR}/${BINARY}"
echo ""
echo "Run: ${BINARY} serve"
