package mapper

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
	} else if addr >= 0x6000 && len(m.cartridge.PRGRAM) > 0 {
		// PRG RAM area - check if enabled
		if (m.prgBank & 0x10) == 0 { // PRG RAM enabled when bit 4 is 0
			addr -= 0x6000
			if int(addr) < len(m.cartridge.PRGRAM) {
				return m.cartridge.PRGRAM[addr]
			}
		}
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
	} else if addr >= 0x6000 && len(m.cartridge.PRGRAM) > 0 {
		// PRG RAM write - check if enabled
		if (m.prgBank & 0x10) == 0 { // PRG RAM enabled when bit 4 is 0
			addr -= 0x6000
			if int(addr) < len(m.cartridge.PRGRAM) {
				m.cartridge.PRGRAM[addr] = value
			}
		}
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
	} else if len(m.cartridge.CHRRAM) > 0 {
		// CHR RAM - typically 8KB, no banking
		if int(addr) < len(m.cartridge.CHRRAM) {
			return m.cartridge.CHRRAM[addr]
		}
	}
	return 0
}

// WriteCHR writes to CHR RAM
func (m *Mapper1) WriteCHR(addr uint16, value uint8) {
	if len(m.cartridge.CHRRAM) > 0 {
		if int(addr) < len(m.cartridge.CHRRAM) {
			m.cartridge.CHRRAM[addr] = value
		}
	}
	// CHR ROM writes are ignored
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

// GetMirroring returns the current mirroring mode
func (m *Mapper1) GetMirroring() uint8 {
	return m.mirroring
}