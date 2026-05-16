package mapper

import (
	"encoding/binary"
	"io"
)

// Mapper1 (MMC1) - Serial port mapper
type Mapper1 struct {
	cartridge *CartridgeData
	
	// Serial port state
	shiftRegister uint8  // 5-bit shift register
	shiftCount    uint8  // Number of bits written
	
	// Internal registers
	control uint8  // Control register ($8000-$9FFF)
	chrBank0 uint8 // CHR bank 0 register ($A000-$BFFF)
	chrBank1 uint8 // CHR bank 1 register ($C000-$DFFF) 
	prgBank  uint8 // PRG bank register ($E000-$FFFF)
	
	// Calculated bank values
	prgMode   uint8  // PRG ROM bank mode (0: 32KB, 1: 16KB)
	chrMode   uint8  // CHR ROM bank mode (0: 8KB, 1: 4KB)
	mirroring uint8  // Nametable mirroring mode
}

// NewMapper1 creates a new Mapper1 instance
func NewMapper1(data *CartridgeData) *Mapper1 {
	return &Mapper1{
		cartridge: data,
		control:   0x0C, // Default: PRG mode 3, CHR mode 0
		prgMode:   3,    // 16KB mode, high bank fixed
		chrMode:   0,    // 8KB mode
		mirroring: 0,    // One-screen lower
	}
}

// ReadPRG reads from PRG ROM/RAM
func (m *Mapper1) ReadPRG(addr uint16) uint8 {
	if addr >= 0x8000 {
		// PRG ROM area
		addr -= 0x8000
		prgSize := len(m.cartridge.PRGROM)
		
		switch m.prgMode {
		case 0, 1: // 32KB mode
			bank := m.prgBank >> 1 // Use bits 4-1 for 32KB banks
			offset := uint32(bank) * 0x8000 + uint32(addr)
			if int(offset) < prgSize {
				return m.cartridge.PRGROM[offset]
			}
		case 2: // 16KB mode, first bank fixed at $8000
			if addr < 0x4000 {
				// Fixed first bank
				if int(addr) < prgSize {
					return m.cartridge.PRGROM[addr]
				}
			} else {
				// Switchable bank at $C000
				bank := m.prgBank & 0x0F
				offset := uint32(bank) * 0x4000 + uint32(addr-0x4000)
				if int(offset) < prgSize {
					return m.cartridge.PRGROM[offset]
				}
			}
		case 3: // 16KB mode, last bank fixed at $C000
			if addr < 0x4000 {
				// Switchable bank at $8000
				bank := m.prgBank & 0x0F
				offset := uint32(bank) * 0x4000 + uint32(addr)
				if int(offset) < prgSize {
					return m.cartridge.PRGROM[offset]
				}
			} else {
				// Fixed last bank
				lastBank := (prgSize / 0x4000) - 1
				offset := uint32(lastBank) * 0x4000 + uint32(addr-0x4000)
				if int(offset) < prgSize {
					return m.cartridge.PRGROM[offset]
				}
			}
		}
	} else if addr >= 0x6000 && (m.prgBank&0x10) == 0 {
		// PRG RAM area - enabled when bit 4 of the PRG bank register is 0.
		return readPRGRAM(m.cartridge, addr)
	}
	return 0
}

// WritePRG handles PRG writes (mapper control and PRG RAM)
func (m *Mapper1) WritePRG(addr uint16, value uint8) {
	if addr >= 0x8000 {
		// Mapper register write via serial port
		if (value & 0x80) != 0 {
			// Reset shift register
			m.shiftRegister = 0
			m.shiftCount = 0
			m.control |= 0x0C // Set PRG mode to 3
			m.prgMode = 3
		} else {
			// Write bit to shift register
			m.shiftRegister = (m.shiftRegister >> 1) | ((value & 1) << 4)
			m.shiftCount++
			
			if m.shiftCount == 5 {
				// Write complete, update register
				m.writeRegister(addr, m.shiftRegister)
				m.shiftRegister = 0
				m.shiftCount = 0
			}
		}
	} else if addr >= 0x6000 && (m.prgBank&0x10) == 0 {
		// PRG RAM write - enabled when bit 4 of the PRG bank register is 0.
		writePRGRAM(m.cartridge, addr, value)
	}
}

// writeRegister writes to internal registers
func (m *Mapper1) writeRegister(addr uint16, value uint8) {
	switch {
	case addr <= 0x9FFF: // Control register
		m.control = value
		m.mirroring = value & 3
		m.prgMode = (value >> 2) & 3
		m.chrMode = (value >> 4) & 1
		
	case addr <= 0xBFFF: // CHR bank 0
		m.chrBank0 = value
		
	case addr <= 0xDFFF: // CHR bank 1
		m.chrBank1 = value
		
	case addr <= 0xFFFF: // PRG bank
		m.prgBank = value
	}
}

// ReadCHR reads from CHR ROM/RAM
func (m *Mapper1) ReadCHR(addr uint16) uint8 {
	if len(m.cartridge.CHRROM) > 0 {
		// CHR ROM
		chrSize := len(m.cartridge.CHRROM)
		var offset uint32
		
		if m.chrMode == 0 {
			// 8KB mode
			bank := m.chrBank0 >> 1 // Use bits 4-1 for 8KB banks
			offset = uint32(bank) * 0x2000 + uint32(addr)
		} else {
			// 4KB mode
			if addr < 0x1000 {
				// First 4KB bank
				offset = uint32(m.chrBank0) * 0x1000 + uint32(addr)
			} else {
				// Second 4KB bank
				offset = uint32(m.chrBank1) * 0x1000 + uint32(addr-0x1000)
			}
		}
		
		if int(offset) < chrSize {
			return m.cartridge.CHRROM[offset]
		}
		return 0
	}
	// CHR RAM fallback — typically 8KB, no banking.
	return readCHRROMOrRAM(m.cartridge, addr)
}

// WriteCHR writes to CHR RAM
func (m *Mapper1) WriteCHR(addr uint16, value uint8) {
	// CHR ROM writes are ignored.
	writeCHRRAM(m.cartridge, addr, value)
}

// Step does nothing for Mapper1 (no IRQ counter)
func (m *Mapper1) Step() {
	// MMC1 has no IRQ functionality
}

// IsIRQPending returns false for Mapper1 (no IRQ support) 
func (m *Mapper1) IsIRQPending() bool {
	return false
}

// ClearIRQ does nothing for Mapper1 (no IRQ support)
func (m *Mapper1) ClearIRQ() {
	// No IRQ to clear
}

// GetMirroringMode translates MMC1 control bits 0-1 into the PPU's mirroring
// code (see pkg/ppu constants). MMC1 mirroring map:
//
//	0 = one-screen lower  -> PPU single-screen lower (2)
//	1 = one-screen upper  -> PPU single-screen upper (3)
//	2 = vertical          -> PPU vertical            (1)
//	3 = horizontal        -> PPU horizontal          (0)
//
// Without this translation, FF2 (which switches to single-screen / vertical
// dynamically) would always render with the iNES header's static mirroring
// because the cartridge layer falls back to the header when no mapper exposes
// GetMirroringMode.
func (m *Mapper1) GetMirroringMode() uint8 {
	switch m.mirroring & 3 {
	case 0:
		return 2 // single-screen lower
	case 1:
		return 3 // single-screen upper
	case 2:
		return 1 // vertical
	case 3:
		return 0 // horizontal
	}
	return 0
}

type mapper1State struct {
	ShiftRegister, ShiftCount uint8
	Control, ChrBank0, ChrBank1, PrgBank uint8
	PrgMode, ChrMode, Mirroring uint8
}

// SaveState writes the full MMC1 register set + serial port state.
func (m *Mapper1) SaveState(w io.Writer) error {
	return binary.Write(w, binary.LittleEndian, mapper1State{
		ShiftRegister: m.shiftRegister, ShiftCount: m.shiftCount,
		Control: m.control,
		ChrBank0: m.chrBank0, ChrBank1: m.chrBank1, PrgBank: m.prgBank,
		PrgMode: m.prgMode, ChrMode: m.chrMode, Mirroring: m.mirroring,
	})
}

// LoadState restores MMC1 state written by SaveState.
func (m *Mapper1) LoadState(r io.Reader) error {
	var s mapper1State
	if err := binary.Read(r, binary.LittleEndian, &s); err != nil {
		return err
	}
	m.shiftRegister, m.shiftCount = s.ShiftRegister, s.ShiftCount
	m.control = s.Control
	m.chrBank0, m.chrBank1, m.prgBank = s.ChrBank0, s.ChrBank1, s.PrgBank
	m.prgMode, m.chrMode, m.mirroring = s.PrgMode, s.ChrMode, s.Mirroring
	return nil
}