package mapper

import (
	"encoding/binary"
	"io"
)

// Mapper3 (CNROM) - Fixed PRG, 8KB CHR bank switching
type Mapper3 struct {
	cartridge *CartridgeData

	// Bank selection
	chrBank      uint8 // Current CHR bank (0-3)
	chrBankCount uint8 // Number of 8KB CHR banks

	// prgMask masks the $8000-offset address into the PRG ROM. $3FFF for
	// 16KB carts (so $8000-$BFFF and $C000-$FFFF mirror), $7FFF for 32KB.
	// Cached at construction to avoid a length-compare on every CPU fetch.
	prgMask uint16

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

	switch len(data.PRGROM) {
	case 16384:
		m.prgMask = 0x3FFF
	default:
		m.prgMask = 0x7FFF
	}

	return m
}

// ReadPRG reads from PRG space. 16KB CNROM carts mirror their PRG bank at
// $8000-$BFFF and $C000-$FFFF (prgMask=$3FFF, set at construction);
// otherwise the reset/NMI/IRQ vectors at $FFFA-$FFFF would fall outside
// ROM and read as 0.
func (m *Mapper3) ReadPRG(addr uint16) uint8 {
	if addr >= 0x8000 {
		offset := (addr - 0x8000) & m.prgMask
		if int(offset) < len(m.cartridge.PRGROM) {
			return m.cartridge.PRGROM[offset]
		}
		return 0
	}
	if addr >= 0x6000 {
		return readPRGRAM(m.cartridge, addr)
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
		return
	}
	// PRG RAM write ($6000-$7FFF). Out-of-range addresses are ignored.
	writePRGRAM(m.cartridge, addr, value)
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
		return 0
	}
	// CHR RAM fallback — no banking on the few CNROM variants that use it.
	return readCHRROMOrRAM(m.cartridge, addr)
}

// WriteCHR writes to CHR space
func (m *Mapper3) WriteCHR(addr uint16, value uint8) {
	// CNROM typically uses CHR ROM (read-only); CHR RAM variants accept
	// direct, unbanked writes.
	writeCHRRAM(m.cartridge, addr, value)
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

// SaveState writes the CHR bank selection. busConflictMode is config (not
// runtime state) and chrBankCount is derived from ROM size, so neither is
// persisted.
func (m *Mapper3) SaveState(w io.Writer) error {
	return binary.Write(w, binary.LittleEndian, m.chrBank)
}

// LoadState restores the CHR bank selection.
func (m *Mapper3) LoadState(r io.Reader) error {
	return binary.Read(r, binary.LittleEndian, &m.chrBank)
}