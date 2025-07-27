#!/bin/bash

# Setup script for Windows cross-compilation with SDL2
set -e

echo "Setting up Windows cross-compilation environment for GoNES..."

# Create directories
mkdir -p /tmp/sdl2-windows
cd /tmp/sdl2-windows

# Download SDL2 development libraries for Windows
SDL2_VERSION="2.28.5"
SDL2_MINGW_URL="https://www.libsdl.org/release/SDL2-devel-${SDL2_VERSION}-mingw.tar.gz"

echo "Downloading SDL2 development libraries..."
wget -O SDL2-devel-mingw.tar.gz "$SDL2_MINGW_URL"

echo "Extracting SDL2 libraries..."
tar -xzf SDL2-devel-mingw.tar.gz

# Install to system directories
sudo mkdir -p /usr/x86_64-w64-mingw32/include
sudo mkdir -p /usr/x86_64-w64-mingw32/lib

echo "Installing SDL2 headers and libraries..."
sudo cp -r SDL2-*/x86_64-w64-mingw32/include/* /usr/x86_64-w64-mingw32/include/
sudo cp -r SDL2-*/x86_64-w64-mingw32/lib/* /usr/x86_64-w64-mingw32/lib/

# Create pkg-config file
sudo mkdir -p /usr/x86_64-w64-mingw32/lib/pkgconfig
sudo tee /usr/x86_64-w64-mingw32/lib/pkgconfig/sdl2.pc > /dev/null << EOF
prefix=/usr/x86_64-w64-mingw32
exec_prefix=\${prefix}
libdir=\${exec_prefix}/lib
includedir=\${prefix}/include

Name: sdl2
Description: Simple DirectMedia Layer
Version: $SDL2_VERSION
Requires:
Conflicts:
Libs: -L\${libdir} -lSDL2main -lSDL2 -mwindows
Libs.private: -lm -ldinput8 -ldxguid -ldxerr8 -luser32 -lgdi32 -lwinmm -limm32 -lole32 -loleaut32 -lshell32 -lsetupapi -lversion -luuid
Cflags: -I\${includedir}/SDL2 -Dmain=SDL_main
EOF

# Cleanup
cd /
rm -rf /tmp/sdl2-windows

echo "✓ Windows cross-compilation setup complete!"
echo ""
echo "You can now build for Windows with:"
echo "  make build-windows"