package mapper

// Mapper0 (NROM) - No mapping
type Mapper0 struct {
	cartridge *CartridgeData
}

// NewMapper0 creates a new Mapper0 instance
func NewMapper0(data *CartridgeData) *Mapper0 {
	return &Mapper0{cartridge: data}
}

// ReadPRG reads from PRG ROM
func (m *Mapper0) ReadPRG(addr uint16) uint8 {
	if addr >= 0x8000 {
		addr -= 0x8000
		if len(m.cartridge.PRGROM) == 16384 {
			// 16KB ROM, mirror at 0xC000
			addr = addr % 16384
		}
		if int(addr) < len(m.cartridge.PRGROM) {
			return m.cartridge.PRGROM[addr]
		}
	} else if addr >= 0x6000 && len(m.cartridge.PRGRAM) > 0 {
		// PRG RAM
		addr -= 0x6000
		if int(addr) < len(m.cartridge.PRGRAM) {
			return m.cartridge.PRGRAM[addr]
		}
	}
	return 0
}

// WritePRG writes to PRG space
func (m *Mapper0) WritePRG(addr uint16, value uint8) {
	if addr >= 0x6000 && addr < 0x8000 && len(m.cartridge.PRGRAM) > 0 {
		// PRG RAM
		addr -= 0x6000
		if int(addr) < len(m.cartridge.PRGRAM) {
			m.cartridge.PRGRAM[addr] = value
		}
	}
	// ROM writes are ignored
}

// ReadCHR reads from CHR ROM/RAM
func (m *Mapper0) ReadCHR(addr uint16) uint8 {
	if len(m.cartridge.CHRROM) > 0 {
		if int(addr) < len(m.cartridge.CHRROM) {
			return m.cartridge.CHRROM[addr]
		} else {
			return 0
		}
	} else if len(m.cartridge.CHRRAM) > 0 {
		if int(addr) < len(m.cartridge.CHRRAM) {
			return m.cartridge.CHRRAM[addr]
		}
	}
	return 0
}

// WriteCHR writes to CHR RAM
func (m *Mapper0) WriteCHR(addr uint16, value uint8) {
	if len(m.cartridge.CHRRAM) > 0 {
		if int(addr) < len(m.cartridge.CHRRAM) {
			m.cartridge.CHRRAM[addr] = value
		}
	}
	// CHR ROM writes are ignored
}

// Step does nothing for Mapper0
func (m *Mapper0) Step() {
	// No special timing for NROM
}

// IsIRQPending returns false for Mapper0 (no IRQ support)
func (m *Mapper0) IsIRQPending() bool {
	return false
}

// ClearIRQ does nothing for Mapper0 (no IRQ support)
func (m *Mapper0) ClearIRQ() {
	// No IRQ to clear
}