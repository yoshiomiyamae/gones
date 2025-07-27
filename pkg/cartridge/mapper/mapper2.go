package mapper

// Mapper2 (UxROM) - 16KB PRG bank switching, CHR RAM
type Mapper2 struct {
	cartridge *CartridgeData
	
	// Bank selection
	prgBank      uint8 // Current PRG bank (0-15)
	prgBankCount uint8 // Number of 16KB PRG banks
}

// NewMapper2 creates a new Mapper2 instance
func NewMapper2(data *CartridgeData) *Mapper2 {
	m := &Mapper2{
		cartridge: data,
		prgBank:   0, // Start with bank 0
	}
	
	// Calculate PRG bank count (16KB banks)
	m.prgBankCount = uint8(len(data.PRGROM) / 16384)
	
	return m
}

// ReadPRG reads from PRG space
func (m *Mapper2) ReadPRG(addr uint16) uint8 {
	if addr >= 0x8000 {
		// PRG ROM area
		if addr < 0xC000 {
			// $8000-$BFFF: Switchable 16KB bank
			bank := m.prgBank % m.prgBankCount
			offset := addr - 0x8000
			finalAddr := uint32(bank)*16384 + uint32(offset)
			
			if finalAddr < uint32(len(m.cartridge.PRGROM)) {
				return m.cartridge.PRGROM[finalAddr]
			}
		} else {
			// $C000-$FFFF: Fixed to last 16KB bank
			lastBank := m.prgBankCount - 1
			offset := addr - 0xC000
			finalAddr := uint32(lastBank)*16384 + uint32(offset)
			
			if finalAddr < uint32(len(m.cartridge.PRGROM)) {
				return m.cartridge.PRGROM[finalAddr]
			}
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

// WritePRG writes to PRG space (handles bank switching)
func (m *Mapper2) WritePRG(addr uint16, value uint8) {
	if addr >= 0x8000 {
		// Bank register write - any write to $8000-$FFFF changes PRG bank
		m.prgBank = value & 0x0F // Only lower 4 bits used for bank selection
	} else if addr >= 0x6000 && addr < 0x8000 && len(m.cartridge.PRGRAM) > 0 {
		// PRG RAM write
		addr -= 0x6000
		if int(addr) < len(m.cartridge.PRGRAM) {
			m.cartridge.PRGRAM[addr] = value
		}
	}
}

// ReadCHR reads from CHR space (CHR RAM only for UxROM)
func (m *Mapper2) ReadCHR(addr uint16) uint8 {
	// UxROM uses CHR RAM, not CHR ROM
	if len(m.cartridge.CHRRAM) > 0 {
		if int(addr) < len(m.cartridge.CHRRAM) {
			return m.cartridge.CHRRAM[addr]
		}
	} else if len(m.cartridge.CHRROM) > 0 {
		// Some UxROM variants may have CHR ROM
		if int(addr) < len(m.cartridge.CHRROM) {
			return m.cartridge.CHRROM[addr]
		}
	}
	
	return 0
}

// WriteCHR writes to CHR space (CHR RAM only)
func (m *Mapper2) WriteCHR(addr uint16, value uint8) {
	// UxROM typically uses CHR RAM
	if len(m.cartridge.CHRRAM) > 0 {
		if int(addr) < len(m.cartridge.CHRRAM) {
			m.cartridge.CHRRAM[addr] = value
		}
	}
	// CHR ROM writes are ignored
}

// Step does nothing for UxROM (no special timing requirements)
func (m *Mapper2) Step() {
	// UxROM doesn't require per-cycle updates
}

// GetCurrentPRGBank returns the current PRG bank for debugging
func (m *Mapper2) GetCurrentPRGBank() uint8 {
	return m.prgBank
}

// IsIRQPending returns false for Mapper2 (no IRQ support)
func (m *Mapper2) IsIRQPending() bool {
	return false
}

// ClearIRQ does nothing for Mapper2 (no IRQ support)
func (m *Mapper2) ClearIRQ() {
	// No IRQ to clear
}