#!/bin/bash

# Build script for GoNES emulator
# Builds for Linux and Windows

set -e

echo "Building GoNES emulator..."

# Clean previous builds
rm -rf build/
mkdir -p build/

# Build for Linux
echo "Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -o build/gones-linux-amd64 ./cmd/gones
echo "✓ Linux build complete: build/gones-linux-amd64"

# Build for Windows
echo "Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -o build/gones-windows-amd64.exe ./cmd/gones
echo "✓ Windows build complete: build/gones-windows-amd64.exe"

# Check if builds exist
echo ""
echo "Build summary:"
ls -la build/

echo ""
echo "Build complete! You can now run:"
echo "  Linux:   ./build/gones-linux-amd64 <rom_file>"
echo "  Windows: ./build/gones-windows-amd64.exe <rom_file>"