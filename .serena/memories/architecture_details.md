# GoNES Architecture Details

## Core Components

### CPU (pkg/cpu)
- 6502 processor emulation
- Handles instructions, addressing modes, interrupts
- Files: `cpu.go`, `instructions.go`, `addressing.go`
- Test coverage: Multiple test files including illegal opcodes

### PPU (pkg/ppu)
- Picture Processing Unit for graphics rendering
- Palette management and rendering
- Files: `ppu.go`, `renderer.go`, `palette.go`

### APU (pkg/apu)
- Audio Processing Unit for sound
- Channel management and register handling
- Files: `apu.go`, `channels.go`, `registers.go`

### Memory (pkg/memory)
- Memory management system
- Handles memory reads/writes and memory mapping

### Cartridge (pkg/cartridge)
- ROM loading and cartridge management
- Mapper implementations:
  - Mapper 0: NROM (no banking)
  - Mapper 1: MMC1
  - Mapper 2: UxROM
  - Mapper 3: CNROM
  - Mapper 4: MMC3
- Each mapper has dedicated tests

### Input (pkg/input)
- Controller input handling
- Keyboard mapping to NES controller buttons

### GUI (pkg/gui)
- SDL2 integration for display and input
- Handles window creation and event loop

### NES (pkg/nes)
- Main system coordinator
- Integrates all components (CPU, PPU, APU, Memory, etc.)
- Manages frame stepping and timing

### Logger (pkg/logger)
- Logging utilities for debugging

## Command-Line Tools

### gones (cmd/gones)
- Main emulator executable
- Usage: `./gones <rom_file.nes>`

### rom_analyzer (cmd/rom_analyzer)
- Tool for analyzing ROM files

### headless_debug (cmd/headless_debug)
- Debugging tool without GUI

## Test Structure
- Unit tests in `pkg/*/` packages (`*_test.go`)
- Integration tests in `test/` directory
- Test helpers in `pkg/cartridge/mapper/test_helpers.go`

## Key Patterns
- Constructor pattern: `New()` or `New<Type>()`
- Pointer receivers for methods that modify state
- Interface-based design for mappers and memory access