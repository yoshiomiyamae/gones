package mapper

// Mapper3 (CNROM) - Fixed PRG, 8KB CHR bank switching
type Mapper3 struct {
	cartridge *CartridgeData
	
	// Bank selection
	chrBank      uint8 // Current CHR bank (0-3)
	chrBankCount uint8 // Number of 8KB CHR banks
	
	// Bus conflict behavior
	busConflictMode uint8 // 0=unknown, 1=no conflicts, 2=AND-type conflicts
}

// NewMapper3 creates a new Mapper3 instance
func NewMapper3(data *CartridgeData) *Mapper3 {
	m := &Mapper3{
		cartridge:       data,
		chrBank:         0, // Start with bank 0
		busConflictMode: 1, // Default to no conflicts for compatibility (submapper 1)
	}
	
	// Calculate CHR bank count (8KB banks)
	if len(data.CHRROM) > 0 {
		m.chrBankCount = uint8(len(data.CHRROM) / 8192)
	}
	
	return m
}

// ReadPRG reads from PRG space (32KB fixed mapping)
func (m *Mapper3) ReadPRG(addr uint16) uint8 {
	if addr >= 0x8000 {
		// PRG ROM area - 32KB fixed mapping (CNROM specification)
		addr -= 0x8000
		if int(addr) < len(m.cartridge.PRGROM) {
			return m.cartridge.PRGROM[addr]
		}
	} else if addr >= 0x6000 && len(m.cartridge.PRGRAM) > 0 {
		// PRG RAM area
		addr -= 0x6000
		if int(addr) < len(m.cartridge.PRGRAM) {
			return m.cartridge.PRGRAM[addr]
		}
	}
	
	return 0
}

// WritePRG writes to PRG space (handles CHR bank switching with bus conflicts)
func (m *Mapper3) WritePRG(addr uint16, value uint8) {
	if addr >= 0x8000 {
		// CHR bank register write - any write to $8000-$FFFF changes CHR bank
		effectiveValue := value
		
		// Handle bus conflicts according to submapper
		if m.busConflictMode == 2 {
			// AND-type bus conflicts: effective value is AND of written value and PRG ROM content
			prgValue := m.ReadPRG(addr)
			effectiveValue = value & prgValue
		}
		// busConflictMode 1 = no conflicts, use value as-is
		// busConflictMode 0 = unknown behavior, default to no conflicts
		
		m.chrBank = effectiveValue & 0x03 // Only lower 2 bits used for bank selection (4 banks max)
	} else if addr >= 0x6000 && addr < 0x8000 && len(m.cartridge.PRGRAM) > 0 {
		// PRG RAM write
		addr -= 0x6000
		if int(addr) < len(m.cartridge.PRGRAM) {
			m.cartridge.PRGRAM[addr] = value
		}
	}
}

// ReadCHR reads from CHR space with bank switching
func (m *Mapper3) ReadCHR(addr uint16) uint8 {
	if len(m.cartridge.CHRROM) > 0 {
		// CHR ROM with banking
		bank := m.chrBank % m.chrBankCount
		finalAddr := uint32(bank)*8192 + uint32(addr)
		
		if finalAddr < uint32(len(m.cartridge.CHRROM)) {
			return m.cartridge.CHRROM[finalAddr]
		}
	} else if len(m.cartridge.CHRRAM) > 0 {
		// CHR RAM - no banking typically, but some variants might have it
		if int(addr) < len(m.cartridge.CHRRAM) {
			return m.cartridge.CHRRAM[addr]
		}
	}
	
	return 0
}

// WriteCHR writes to CHR space
func (m *Mapper3) WriteCHR(addr uint16, value uint8) {
	// CNROM typically uses CHR ROM which is read-only
	// CHR RAM variants don't use banking - direct access only
	if len(m.cartridge.CHRRAM) > 0 {
		if int(addr) < len(m.cartridge.CHRRAM) {
			m.cartridge.CHRRAM[addr] = value
		}
	}
	// CHR ROM writes are ignored
}

// Step does nothing for CNROM (no special timing requirements)
func (m *Mapper3) Step() {
	// CNROM doesn't require per-cycle updates
}

// GetCurrentCHRBank returns the current CHR bank for debugging
func (m *Mapper3) GetCurrentCHRBank() uint8 {
	return m.chrBank
}

// IsIRQPending returns false for Mapper3 (no IRQ support)
func (m *Mapper3) IsIRQPending() bool {
	return false
}

// ClearIRQ does nothing for Mapper3 (no IRQ support)
func (m *Mapper3) ClearIRQ() {
	// No IRQ to clear
}

// SetBusConflictMode sets the bus conflict behavior based on submapper
// 0 = unknown behavior, 1 = no conflicts, 2 = AND-type conflicts
func (m *Mapper3) SetBusConflictMode(mode uint8) {
	if mode <= 2 {
		m.busConflictMode = mode
	}
}