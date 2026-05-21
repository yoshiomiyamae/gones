package memory

import (
	"io"

	"github.com/yoshiomiyamaegones/pkg/logger"
)

// PPUBus is the subset of the PPU that the memory bus needs: CPU-visible
// register I/O at $2000-$2007 (and mirrors).
type PPUBus interface {
	ReadRegister(addr uint16) uint8
	WriteRegister(addr uint16, value uint8)
}

// APUBus is the subset of the APU that the memory bus needs: register
// I/O at $4000-$4017.
type APUBus interface {
	ReadRegister(addr uint16) uint8
	WriteRegister(addr uint16, value uint8)
}

// CartridgeBus is the subset of the cartridge that the memory bus needs:
// PRG read/write at $6000-$FFFF (plus $4020-$5FFF for mappers that
// HaveExpansion). CHR access goes through the PPU, not here.
type CartridgeBus interface {
	ReadPRG(addr uint16) uint8
	WritePRG(addr uint16, value uint8)
	HasExpansion() bool
}

// InputBus is the subset of the controller used by the memory bus:
// strobe write to $4016 and serial read.
type InputBus interface {
	Read() uint8
	Write(value uint8)
}

// CheatPatcher is an optional read-time byte patcher. Implementations
// (e.g. Game Genie / PAR) get the address and the value the underlying
// region returned, and may override it.
type CheatPatcher interface {
	Apply(addr uint16, current uint8) uint8
}

// Memory represents the NES memory map
type Memory struct {
	// CPU RAM (2KB, mirrored to fill 8KB)
	RAM [2048]uint8

	// Test memory for high addresses (for testing purposes)
	HighMem [0xA000]uint8 // 0x6000-0xFFFF

	PPU       PPUBus
	APU       APUBus
	Cartridge CartridgeBus
	Input     InputBus

	// Cheats is an optional read-time patcher. When set, every Read result
	// is filtered through Cheats.Apply, letting Game Genie / PAR cheats
	// override ROM and RAM bytes without modifying underlying storage.
	// Hot path: keep the nil check ahead of the call to avoid the
	// interface dispatch when no cheats are loaded.
	Cheats CheatPatcher

	// cpuBus is the most recent byte the CPU saw on its data bus. Reads
	// from "non-driving" addresses (write-only APU ports, unallocated
	// $4018-$401F, etc.) return this stale latched value instead of zero.
	// blargg's cpu_exec_space_apu test runs code at $4000+ and depends on
	// the open-bus value (the high byte of the preceding JMP) decoding as
	// RTI; without proper tracking the CPU fetches $00=BRK and crashes.
	cpuBus uint8
}

// New creates a new Memory instance
func New() *Memory {
	return &Memory{}
}

// SetCartridge sets the cartridge reference
func (m *Memory) SetCartridge(cart CartridgeBus) { m.Cartridge = cart }

// SetPPU sets the PPU reference
func (m *Memory) SetPPU(ppu PPUBus) { m.PPU = ppu }

// SetAPU sets the APU reference
func (m *Memory) SetAPU(apu APUBus) { m.APU = apu }

// SetInput sets the input reference
func (m *Memory) SetInput(input InputBus) { m.Input = input }

// Read reads a byte from the given address. The cheat patcher (when set)
// gets the last word so it can overlay Game Genie / RAM cheats on top of
// whatever the underlying region returned.
func (m *Memory) Read(addr uint16) uint8 {
	v := m.read(addr)
	if m.Cheats != nil {
		return m.Cheats.Apply(addr, v)
	}
	return v
}

// read is the unpatched memory read. Every successful read latches into
// cpuBus; addresses that don't drive the bus (write-only APU ports,
// $4018-$401F, unmapped cartridge space) return the previous latched
// value instead of zero.
func (m *Memory) read(addr uint16) uint8 {
	if addr < 0x2000 {
		v := m.RAM[addr&0x7FF]
		m.cpuBus = v
		return v
	}

	if addr >= 0x6000 {
		if m.Cartridge != nil {
			v := m.Cartridge.ReadPRG(addr)
			m.cpuBus = v
			return v
		}
		v := m.HighMem[addr-0x6000]
		m.cpuBus = v
		return v
	}

	if addr < 0x4000 {
		if m.PPU != nil {
			v := m.PPU.ReadRegister(0x2000 + (addr & 0x7))
			m.cpuBus = v
			return v
		}
		return m.cpuBus
	}

	if addr == 0x4015 {
		if m.APU != nil {
			v := m.APU.ReadRegister(addr)
			m.cpuBus = v
			return v
		}
		return m.cpuBus
	}

	if addr == 0x4016 {
		// Bit 0 from controller; bits 1-7 are open bus on real hardware.
		if m.Input != nil {
			v := (m.cpuBus & 0xFE) | (m.Input.Read() & 0x01)
			m.cpuBus = v
			return v
		}
		return m.cpuBus
	}

	// $4000-$4014 are write-only APU ports, $4017 read is player-2 controller
	// (not modelled yet), $4018-$401F is CPU-test/unallocated.
	if addr < 0x4020 {
		return m.cpuBus
	}

	// $4020-$5FFF is cartridge expansion. Only mappers that opt in
	// (mapper.ExpansionDecoder — currently MMC5) decode this range;
	// everything else leaves it as open bus so the cpu_exec_space
	// test ROM can keep relying on $40xx latching for its
	// JMP-into-APU-space round-trip.
	if m.Cartridge != nil && m.Cartridge.HasExpansion() {
		v := m.Cartridge.ReadPRG(addr)
		m.cpuBus = v
		return v
	}
	return m.cpuBus
}

// oamDMAStallCycles is the cost an OAM DMA adds to the CPU on top of the
// 4-cycle STA $4014: 1 dummy/alignment + 256 reads + 256 writes. Returned
// from Write so the CPU can charge the stall without knowing about $4014.
const oamDMAStallCycles = 513

// Write writes a byte to the given address. Returns the number of extra
// CPU stall cycles the access cost beyond the instruction's normal cycle
// count — non-zero only for OAM DMA at $4014. Other callers may safely
// ignore the return.
func (m *Memory) Write(addr uint16, value uint8) int {
	m.cpuBus = value
	switch {
	case addr < 0x2000:
		// CPU RAM (0x0000-0x1FFF, mirrored every 0x800 bytes)
		m.RAM[addr%0x800] = value

	case addr < 0x4000:
		// PPU registers (0x2000-0x3FFF, mirrored every 8 bytes)
		if m.PPU != nil {
			ppuAddr := 0x2000 + (addr & 0x7)
			if ppuAddr == 0x2006 || ppuAddr == 0x2007 {
				logger.LogCPU("Memory Write PPU $%04X: value=$%02X", ppuAddr, value)
			}
			m.PPU.WriteRegister(ppuAddr, value)
		}

	case addr == 0x4014:
		m.performOAMDMA(value)
		return oamDMAStallCycles

	case addr == 0x4016:
		if m.Input != nil {
			m.Input.Write(value)
		}

	case addr < 0x4020:
		if m.APU != nil {
			m.APU.WriteRegister(addr, value)
		}
	case addr < 0x6000:
		// Cartridge expansion ($4020-$5FFF). Only opt-in mappers (MMC5)
		// receive these writes; for everything else the write is a
		// no-op and the bus latch above stands.
		if m.Cartridge != nil && m.Cartridge.HasExpansion() {
			m.Cartridge.WritePRG(addr, value)
		}
	case addr >= 0x6000:
		if m.Cartridge != nil {
			m.Cartridge.WritePRG(addr, value)
		} else {
			index := addr - 0x6000
			if index >= 0xA000 {
				return 0
			}
			m.HighMem[index] = value
		}
	}
	return 0
}

// SaveState writes the CPU work RAM contents (2KB) to w.
func (m *Memory) SaveState(w io.Writer) error {
	_, err := w.Write(m.RAM[:])
	return err
}

// LoadState restores the CPU work RAM from r.
func (m *Memory) LoadState(r io.Reader) error {
	_, err := io.ReadFull(r, m.RAM[:])
	return err
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
