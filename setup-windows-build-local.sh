#!/bin/bash

# Setup script for Windows cross-compilation with SDL2 (local installation)
set -e

echo "Setting up Windows cross-compilation environment for GoNES (local)..."

# Create local directories
LOCAL_PREFIX="$HOME/.local/mingw64"
mkdir -p "$LOCAL_PREFIX"/{include,lib,lib/pkgconfig}

# Create temp directory
mkdir -p /tmp/sdl2-windows
cd /tmp/sdl2-windows

# Download SDL2 development libraries for Windows
SDL2_VERSION="2.28.5"
SDL2_MINGW_URL="https://www.libsdl.org/release/SDL2-devel-${SDL2_VERSION}-mingw.tar.gz"

echo "Downloading SDL2 development libraries..."
wget -O SDL2-devel-mingw.tar.gz "$SDL2_MINGW_URL"

echo "Extracting SDL2 libraries..."
tar -xzf SDL2-devel-mingw.tar.gz

# Install to local directories
echo "Installing SDL2 headers and libraries to $LOCAL_PREFIX..."
cp -r SDL2-*/x86_64-w64-mingw32/include/* "$LOCAL_PREFIX/include/"
cp -r SDL2-*/x86_64-w64-mingw32/lib/* "$LOCAL_PREFIX/lib/"

# Create pkg-config file
cat > "$LOCAL_PREFIX/lib/pkgconfig/sdl2.pc" << EOF
prefix=$LOCAL_PREFIX
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
echo "SDL2 installed to: $LOCAL_PREFIX"
echo ""
echo "You can now build for Windows with:"
echo "  PKG_CONFIG_PATH=\"$LOCAL_PREFIX/lib/pkgconfig\" make build-windows-local"
echo ""
echo "Or export the environment variable:"
echo "  export PKG_CONFIG_PATH=\"$LOCAL_PREFIX/lib/pkgconfig\""
echo "  make build-windows-local"