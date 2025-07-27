# Makefile for GoNES emulator

.PHONY: build build-linux build-windows test clean install-deps help

# Default target
all: build

# Build for current platform
build:
	@echo "Building GoNES for current platform..."
	go build -o gones ./cmd/gones
	@echo "✓ Build complete: ./gones"

# Build for Linux
build-linux:
	@echo "Building for Linux (amd64)..."
	@mkdir -p build
	GOOS=linux GOARCH=amd64 go build -o build/gones-linux-amd64 ./cmd/gones
	@echo "✓ Linux build complete: build/gones-linux-amd64"

# Build for Windows (requires mingw-w64 and SDL2 setup)
build-windows:
	@echo "Building for Windows (amd64)..."
	@mkdir -p build
	@echo "Checking for SDL2 Windows development libraries..."
	@if [ ! -f "/usr/x86_64-w64-mingw32/include/SDL2/SDL.h" ]; then \
		echo "❌ SDL2 Windows development libraries not found."; \
		echo ""; \
		echo "Please run: ./setup-windows-build.sh"; \
		echo ""; \
		echo "Or install manually:"; \
		echo "  1. Download SDL2 development libraries from https://www.libsdl.org/download-2.0.php"; \
		echo "  2. Extract to /usr/x86_64-w64-mingw32/"; \
		echo ""; \
		exit 1; \
	fi
	PKG_CONFIG_PATH="/usr/x86_64-w64-mingw32/lib/pkgconfig" \
	CGO_ENABLED=1 \
	GOOS=windows \
	GOARCH=amd64 \
	CC=x86_64-w64-mingw32-gcc \
	go build -o build/gones-windows-amd64.exe ./cmd/gones
	@echo "✓ Windows build complete: build/gones-windows-amd64.exe"

# Build for Windows using local SDL2 installation
build-windows-local:
	@echo "Building for Windows (amd64) with local SDL2..."
	@mkdir -p build
	@LOCAL_PREFIX="$$HOME/.local/mingw64"; \
	if [ ! -f "$$LOCAL_PREFIX/include/SDL2/SDL.h" ]; then \
		echo "❌ SDL2 Windows development libraries not found in $$LOCAL_PREFIX"; \
		echo ""; \
		echo "Please run: ./setup-windows-build-local.sh"; \
		echo ""; \
		exit 1; \
	fi; \
	PKG_CONFIG_PATH="$$LOCAL_PREFIX/lib/pkgconfig" \
	CGO_ENABLED=1 \
	GOOS=windows \
	GOARCH=amd64 \
	CC=x86_64-w64-mingw32-gcc \
	go build -o build/gones-windows-amd64.exe ./cmd/gones
	@echo "✓ Windows build complete: build/gones-windows-amd64.exe"

# Build for both platforms
build-all: build-linux build-windows
	@echo "✓ All builds complete!"
	@ls -la build/

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf build/
	rm -f gones gones.exe
	@echo "✓ Clean complete"

# Install dependencies
install-deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download
	@echo "✓ Dependencies installed"

# Install build tools for cross-compilation
install-build-tools:
	@echo "Installing cross-compilation tools..."
	@echo "On Ubuntu/Debian:"
	@echo "  sudo apt-get install gcc-mingw-w64"
	@echo "On macOS:"
	@echo "  brew install mingw-w64"

# Run with a test ROM
run-test:
	@if [ -f "test/roms/nestest.nes" ]; then \
		./gones test/roms/nestest.nes; \
	else \
		echo "Test ROM not found. Please place a .nes file in the test/roms/ directory"; \
	fi

# Help
help:
	@echo "GoNES Emulator Build System"
	@echo ""
	@echo "Available targets:"
	@echo "  build            - Build for current platform"
	@echo "  build-linux      - Build for Linux (amd64)"
	@echo "  build-windows    - Build for Windows (amd64) - system-wide SDL2"
	@echo "  build-windows-local - Build for Windows (amd64) - local SDL2"
	@echo "  build-all        - Build for all platforms"
	@echo "  test             - Run tests"
	@echo "  clean            - Clean build artifacts"
	@echo "  install-deps     - Install Go dependencies"
	@echo "  install-build-tools - Show commands to install cross-compilation tools"
	@echo "  run-test         - Run with test ROM"
	@echo "  help             - Show this help"
	@echo ""
	@echo "Requirements for Windows cross-compilation:"
	@echo "  - mingw-w64 (gcc-mingw-w64 package)"
	@echo ""
	@echo "SDL2 Requirements:"
	@echo "  - Linux: libsdl2-dev"
	@echo "  - macOS: sdl2 (via Homebrew)"
	@echo "  - Windows: Included with go-sdl2"