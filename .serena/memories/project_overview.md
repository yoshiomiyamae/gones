# GoNES Project Overview

## Purpose
GoNES is a Nintendo Entertainment System (NES) emulator written in Go and SDL2. It provides full NES compatibility with accurate 6502 CPU, PPU, and APU implementations.

## Tech Stack
- **Language**: Go 1.21+
- **Graphics/Audio**: SDL2 (via go-sdl2 v0.4.35)
- **Platform**: Cross-platform (Windows, macOS, Linux)

## Key Features
- Full NES compatibility (6502 CPU, PPU, APU)
- 60 FPS operation with real-time audio
- Modular architecture for maintainability
- Multiple mapper support (Mapper 0, 1, 2, 3, 4)

## Project Structure
```
pkg/
├── cpu/          # 6502 processor implementation
├── ppu/          # Picture Processing Unit
├── apu/          # Audio Processing Unit
├── memory/       # Memory management
├── cartridge/    # Cartridge and Mapper implementations
├── input/        # Input handling
├── gui/          # GUI layer (SDL2)
├── logger/       # Logging utilities
└── nes/          # Main NES system coordination

cmd/
├── gones/            # Main emulator executable
├── rom_analyzer/     # ROM analysis tool
└── headless_debug/   # Headless debugging tool

test/
└── *.go              # Integration tests
```

## Module Path
`github.com/yoshiomiyamaegones`