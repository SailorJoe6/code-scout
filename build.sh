#!/bin/bash
set -e

# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    arm64) ARCH="arm64" ;;
esac

# Set CGO flags
export CGO_CFLAGS="-I$(pwd)/include"

if [[ "$OS" == "darwin" ]]; then
    export CGO_LDFLAGS="-L$(pwd)/lib/${OS}_${ARCH} -llancedb_go -framework Security -framework CoreFoundation"
else
    export CGO_LDFLAGS="-L$(pwd)/lib/${OS}_${ARCH} -llancedb_go"
fi

# Build
echo "Building for ${OS}_${ARCH}..."
go build -o code-scout ./cmd/code-scout

echo "✓ Build complete: ./code-scout"
