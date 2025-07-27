package memory

import (
	"github.com/yoshiomiyamaegones/pkg/logger"
)

// Memory represents the NES memory map
type Memory struct {
	// CPU RAM (2KB, mirrored to fill 8KB)
	RAM [2048]uint8

	// Test memory for high addresses (for testing purposes)
	HighMem [0xA000]uint8 // 0x6000-0xFFFF

	// PPU interface
	PPU interface {
		ReadRegister(addr uint16) uint8
		WriteRegister(addr uint16, value uint8)
	}

	// APU interface
	APU interface {
		ReadRegister(addr uint16) uint8
		WriteRegister(addr uint16, value uint8)
	}

	// Cartridge interface
	Cartridge interface {
		ReadPRG(addr uint16) uint8
		WritePRG(addr uint16, value uint8)
	}

	// Input interface
	Input interface {
		Read() uint8
		Write(value uint8)
	}
}

// New creates a new Memory instance
func New() *Memory {
	return &Memory{}
}

// SetCartridge sets the cartridge reference
func (m *Memory) SetCartridge(cart interface {
	ReadPRG(addr uint16) uint8
	WritePRG(addr uint16, value uint8)
}) {
	m.Cartridge = cart
}

// SetPPU sets the PPU reference
func (m *Memory) SetPPU(ppu interface {
	ReadRegister(addr uint16) uint8
	WriteRegister(addr uint16, value uint8)
}) {
	m.PPU = ppu
}

// SetAPU sets the APU reference
func (m *Memory) SetAPU(apu interface {
	ReadRegister(addr uint16) uint8
	WriteRegister(addr uint16, value uint8)
}) {
	m.APU = apu
}

// SetInput sets the input reference
func (m *Memory) SetInput(input interface {
	Read() uint8
	Write(value uint8)
}) {
	m.Input = input
}

// Read reads a byte from the given address with optimized path for common cases
func (m *Memory) Read(addr uint16) uint8 {

	// Fast path for most common accesses (CPU RAM and cartridge)
	if addr < 0x2000 {
		// CPU RAM (0x0000-0x1FFF, mirrored every 0x800 bytes)
		return m.RAM[addr&0x7FF] // Use bitwise AND for faster modulo
	}

	if addr >= 0x6000 {
		// Cartridge PRG ROM space (0x8000-0xFFFF) - most frequent after RAM
		if m.Cartridge != nil {
			return m.Cartridge.ReadPRG(addr)
		}
		// For testing: use HighMem when no cartridge is present
		index := addr - 0x6000
		if index >= 0xA000 {
			// Index out of bounds - this shouldn't happen
			return 0
		}
		return m.HighMem[index]
	}

	// Less frequent accesses
	if addr < 0x4000 {
		// PPU registers (0x2000-0x3FFF, mirrored every 8 bytes)
		if m.PPU != nil {
			return m.PPU.ReadRegister(0x2000 + (addr & 0x7))
		}
		return 0
	}

	if addr == 0x4016 {
		// Controller 1
		if m.Input != nil {
			return m.Input.Read()
		}
		return 0
	}

	if addr == 0x4017 {
		// Controller 2 / APU frame counter
		if m.APU != nil {
			return m.APU.ReadRegister(addr)
		}
		return 0
	}

	if addr < 0x4020 {
		// APU and I/O registers (0x4000-0x401F)
		if m.APU != nil {
			return m.APU.ReadRegister(addr)
		}
		return 0
	}

	// Unmapped addr > 0x4020 && addr < 0x6000
	return 0
}

// Write writes a byte to the given address
func (m *Memory) Write(addr uint16, value uint8) {

	switch {
	case addr < 0x2000:
		// CPU RAM (0x0000-0x1FFF, mirrored every 0x800 bytes)
		m.RAM[addr%0x800] = value

	case addr < 0x4000:
		// PPU registers (0x2000-0x3FFF, mirrored every 8 bytes)
		if m.PPU != nil {
			ppuAddr := 0x2000 + (addr & 0x7)
			// Debug: Log $2006/$2007 writes specifically
			if ppuAddr == 0x2006 || ppuAddr == 0x2007 {
				logger.LogCPU("Memory Write PPU $%04X: value=$%02X", ppuAddr, value)
			}
			m.PPU.WriteRegister(ppuAddr, value)
		}

	case addr == 0x4014:
		// OAM DMA
		m.performOAMDMA(value)

	case addr == 0x4016:
		// Controller 1
		if m.Input != nil {
			m.Input.Write(value)
		}

	case addr < 0x4020:
		// APU and I/O registers (0x4000-0x401F)
		if m.APU != nil {
			m.APU.WriteRegister(addr, value)
		}
	case addr >= 0x6000:
		// Cartridge PRG ROM space (0x8000-0xFFFF)
		if m.Cartridge != nil {
			m.Cartridge.WritePRG(addr, value)
		} else {
			// For testing: use HighMem when no cartridge is present
			index := addr - 0x6000
			if index >= 0xA000 {
				// Index out of bounds - this shouldn't happen
				return
			}
			m.HighMem[index] = value
		}

	default:
		// Unmapped addr > 0x4020 && addr < 0x6000
	}
}

// performOAMDMA performs OAM DMA transfer
func (m *Memory) performOAMDMA(page uint8) {
	// Transfer 256 bytes from CPU memory to PPU OAM
	baseAddr := uint16(page) << 8

	for i := 0; i < 256; i++ {
		value := m.Read(baseAddr + uint16(i))
		if m.PPU != nil {
			m.PPU.WriteRegister(0x2004, value)
		}
	}
}
