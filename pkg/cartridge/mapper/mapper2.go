package mapper

import (
	"encoding/binary"
	"io"
)

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
	} else if addr >= 0x6000 {
		return readPRGRAM(m.cartridge, addr)
	}

	return 0
}

// WritePRG writes to PRG space (handles bank switching)
func (m *Mapper2) WritePRG(addr uint16, value uint8) {
	if addr >= 0x8000 {
		// Bank register write - any write to $8000-$FFFF changes PRG bank
		m.prgBank = value & 0x0F // Only lower 4 bits used for bank selection
		return
	}
	// PRG RAM write ($6000-$7FFF). Out-of-range addresses are ignored.
	writePRGRAM(m.cartridge, addr, value)
}

// ReadCHR reads from CHR space (CHR RAM only for UxROM)
func (m *Mapper2) ReadCHR(addr uint16) uint8 {
	// UxROM normally uses CHR RAM; some variants ship CHR ROM instead.
	// readCHRROMOrRAM prefers ROM when present, falling back to RAM.
	return readCHRROMOrRAM(m.cartridge, addr)
}

// WriteCHR writes to CHR space (CHR RAM only)
func (m *Mapper2) WriteCHR(addr uint16, value uint8) {
	// CHR ROM writes are ignored.
	writeCHRRAM(m.cartridge, addr, value)
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

// SaveState writes the currently selected PRG bank.
func (m *Mapper2) SaveState(w io.Writer) error {
	return binary.Write(w, binary.LittleEndian, m.prgBank)
}

// LoadState restores the selected PRG bank.
func (m *Mapper2) LoadState(r io.Reader) error {
	return binary.Read(r, binary.LittleEndian, &m.prgBank)
}