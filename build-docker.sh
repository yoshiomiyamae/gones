#!/bin/bash

# Docker-based cross compilation for GoNES
# This allows building Windows binaries from any platform

set -e

echo "Building GoNES with Docker for cross-platform support..."

# Create Dockerfile
cat > Dockerfile.build <<EOF
FROM ubuntu:22.04 AS builder

# Prevent interactive prompts
ENV DEBIAN_FRONTEND=noninteractive

# Install build dependencies
RUN apt-get update && apt-get install -y \
    golang-1.21 \
    gcc \
    gcc-mingw-w64 \
    pkg-config \
    libsdl2-dev \
    wget \
    curl \
    unzip \
    && rm -rf /var/lib/apt/lists/*

# Add Go to PATH
ENV PATH=/usr/lib/go-1.21/bin:\$PATH

# Setup SDL2 for Windows cross-compilation
RUN mkdir -p /tmp/sdl2-setup && cd /tmp/sdl2-setup && \
    wget https://www.libsdl.org/release/SDL2-devel-2.28.5-mingw.tar.gz && \
    tar -xzf SDL2-devel-2.28.5-mingw.tar.gz && \
    mkdir -p /usr/x86_64-w64-mingw32/include && \
    mkdir -p /usr/x86_64-w64-mingw32/lib && \
    cp -r SDL2-*/x86_64-w64-mingw32/include/* /usr/x86_64-w64-mingw32/include/ && \
    cp -r SDL2-*/x86_64-w64-mingw32/lib/* /usr/x86_64-w64-mingw32/lib/ && \
    rm -rf /tmp/sdl2-setup

# Create pkg-config file for Windows SDL2
RUN mkdir -p /usr/x86_64-w64-mingw32/lib/pkgconfig && \
    echo 'prefix=/usr/x86_64-w64-mingw32' > /usr/x86_64-w64-mingw32/lib/pkgconfig/sdl2.pc && \
    echo 'exec_prefix=\${prefix}' >> /usr/x86_64-w64-mingw32/lib/pkgconfig/sdl2.pc && \
    echo 'libdir=\${exec_prefix}/lib' >> /usr/x86_64-w64-mingw32/lib/pkgconfig/sdl2.pc && \
    echo 'includedir=\${prefix}/include' >> /usr/x86_64-w64-mingw32/lib/pkgconfig/sdl2.pc && \
    echo '' >> /usr/x86_64-w64-mingw32/lib/pkgconfig/sdl2.pc && \
    echo 'Name: sdl2' >> /usr/x86_64-w64-mingw32/lib/pkgconfig/sdl2.pc && \
    echo 'Description: Simple DirectMedia Layer' >> /usr/x86_64-w64-mingw32/lib/pkgconfig/sdl2.pc && \
    echo 'Version: 2.28.5' >> /usr/x86_64-w64-mingw32/lib/pkgconfig/sdl2.pc && \
    echo 'Libs: -L\${libdir} -lSDL2main -lSDL2 -mwindows' >> /usr/x86_64-w64-mingw32/lib/pkgconfig/sdl2.pc && \
    echo 'Cflags: -I\${includedir}/SDL2 -Dmain=SDL_main' >> /usr/x86_64-w64-mingw32/lib/pkgconfig/sdl2.pc

WORKDIR /app
COPY . .

# Download dependencies
RUN go mod download

# Build for Linux
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o gones-linux-amd64 ./cmd/gones

# Build for Windows
RUN PKG_CONFIG_PATH="/usr/x86_64-w64-mingw32/lib/pkgconfig" \
    CGO_ENABLED=1 \
    GOOS=windows \
    GOARCH=amd64 \
    CC=x86_64-w64-mingw32-gcc \
    go build -o gones-windows-amd64.exe ./cmd/gones

FROM scratch AS binaries
COPY --from=builder /app/gones-linux-amd64 /
COPY --from=builder /app/gones-windows-amd64.exe /
EOF

# Build with Docker
docker build -f Dockerfile.build -t gones-builder .

# Extract binaries
mkdir -p build
docker run --rm -v "$(pwd)/build:/out" gones-builder sh -c "cp /gones-* /out/"

# Cleanup
rm Dockerfile.build

echo "✓ Docker build complete!"
echo ""
echo "Binaries available in build/ directory:"
ls -la build/
echo ""
echo "You can now run:"
echo "  Linux:   ./build/gones-linux-amd64 <rom_file>"
echo "  Windows: wine ./build/gones-windows-amd64.exe <rom_file>  # if wine is installed"