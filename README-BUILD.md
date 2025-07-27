# GoNES Build Instructions

## Prerequisites

### Linux
```bash
# Ubuntu/Debian
sudo apt-get install libsdl2-dev

# Fedora/CentOS
sudo dnf install SDL2-devel

# Arch Linux
sudo pacman -S sdl2
```

### macOS
```bash
brew install sdl2
```

### Windows
SDL2 libraries are included with the go-sdl2 package for Windows builds.

## Building

### Quick Build (Current Platform)
```bash
go build -o gones ./cmd/gones
```

### Using Makefile
```bash
# Build for current platform
make build

# Build for Linux
make build-linux

# Build for Windows (requires mingw-w64)
make build-windows

# Build for all platforms
make build-all

# Run tests
make test

# Clean builds
make clean
```

### Using Build Script
```bash
# Make executable
chmod +x build.sh

# Run build script
./build.sh
```

### Using Docker (Recommended for Windows builds)
```bash
# Build both Linux and Windows binaries with Docker
./build-docker.sh
```

## Cross-Compilation Setup

### For Windows Cross-Compilation on Linux

#### Method 1: Automatic Setup (Recommended)
```bash
# Install mingw-w64 first
sudo apt-get install gcc-mingw-w64

# Run the setup script
./setup-windows-build.sh

# Now you can build for Windows
make build-windows
```

#### Method 2: Docker Build (Universal)
```bash
# Build both Linux and Windows binaries with Docker
# No additional setup required - everything is handled in Docker
./build-docker.sh
```

#### Method 3: Manual Setup
```bash
# Install cross-compiler
# Ubuntu/Debian
sudo apt-get install gcc-mingw-w64

# Download and install SDL2 Windows development libraries
wget https://www.libsdl.org/release/SDL2-devel-2.28.5-mingw.tar.gz
tar -xzf SDL2-devel-2.28.5-mingw.tar.gz
sudo mkdir -p /usr/x86_64-w64-mingw32/{include,lib}
sudo cp -r SDL2-*/x86_64-w64-mingw32/include/* /usr/x86_64-w64-mingw32/include/
sudo cp -r SDL2-*/x86_64-w64-mingw32/lib/* /usr/x86_64-w64-mingw32/lib/

# Create pkg-config file (see setup-windows-build.sh for details)
```

### Manual Cross-Compilation
```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o gones-linux ./cmd/gones

# Windows (after SDL2 setup)
PKG_CONFIG_PATH="/usr/x86_64-w64-mingw32/lib/pkgconfig" \
CGO_ENABLED=1 \
GOOS=windows \
GOARCH=amd64 \
CC=x86_64-w64-mingw32-gcc \
go build -o gones-windows.exe ./cmd/gones
```

## Running

```bash
# Linux
./gones <rom_file.nes>

# Windows
gones.exe <rom_file.nes>

# Test with included ROM
make run-test
```

### Controls
- **Z** - A button
- **X** - B button  
- **A** - Select
- **S** - Start
- **Arrow Keys** - D-pad
- **ESC** - Quit

## Supported ROM Formats
- iNES format (.nes files)
- Mappers 0, 1, 2, 3, 4 (NROM, MMC1, UxROM, CNROM, MMC3)

## Troubleshooting

### SDL2 Not Found
Make sure SDL2 development libraries are installed on your system.

### Windows Build Fails
Ensure mingw-w64 is installed for cross-compilation.

### ROM Not Loading
- Check if the ROM file exists
- Verify it's a valid .nes file
- Make sure the mapper is supported (0-4)