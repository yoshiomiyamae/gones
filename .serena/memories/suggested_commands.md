# Suggested Commands for GoNES

## Build Commands
```bash
# Build for current platform
make build
# or
go build -o gones ./cmd/gones

# Build for Linux
make build-linux

# Build for Windows (system-wide SDL2)
make build-windows

# Build for Windows (local SDL2 in ~/.local/mingw64)
make build-windows-local

# Build for all platforms
make build-all
```

## Test Commands
```bash
# Run all tests
make test
# or
go test ./...

# Run tests for specific package
go test ./pkg/cpu
go test ./pkg/ppu
go test ./pkg/cartridge/mapper
```

## Dependency Management
```bash
# Install/update dependencies
make install-deps
# or
go mod tidy
go mod download
```

## Running the Emulator
```bash
# Run with a ROM file
./gones <rom_file.nes>

# Run with test ROM (if available)
make run-test
```

## Clean Build Artifacts
```bash
make clean
```

## Cross-compilation Tools
```bash
# Show installation commands for cross-compilation tools
make install-build-tools

# On Ubuntu/Debian:
sudo apt-get install gcc-mingw-w64 libsdl2-dev

# On macOS:
brew install mingw-w64 sdl2
```

## Utility Commands (Linux)
- `ls` - List files
- `cd` - Change directory
- `grep` - Search text
- `find` - Find files
- `git` - Version control