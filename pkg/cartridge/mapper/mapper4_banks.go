package mapper

// This file hosts MMC3 PRG/CHR bank mapping. The $8000-$FFFF register-write
// dispatcher (WritePRG) lives here because the majority of its switch arms
// are bank-related ($8000 bank select, $8001 bank data, $A000 mirroring,
// $A001 PRG RAM protect); the IRQ-register arms ($C000/$C001/$E000/$E001)
// delegate to helpers defined in mapper4_irq.go.

import (
	"github.com/yoshiomiyamaegones/pkg/logger"
)

// ReadPRG reads from PRG ROM/RAM address space
func (m *Mapper4) ReadPRG(addr uint16) uint8 {
	switch {
	case addr >= 0x6000 && addr <= 0x7FFF:
		// PRG RAM area - check both read enable and write protect
		if len(m.data.PRGRAM) > 0 && (m.prgRAMProtect&0x80) != 0 {
			return m.data.PRGRAM[addr-0x6000]
		}
		return 0

	case addr >= 0x8000 && addr <= 0xFFFF:
		// Correct MMC3 PRG ROM banking according to NESdev wiki
		var bank uint8
		prgMode := (m.bankSelect >> 6) & 1

		switch {
		case addr >= 0x8000 && addr <= 0x9FFF:
			// $8000-$9FFF: R6 in mode 0, second-to-last in mode 1
			if prgMode == 0 {
				bank = m.bankRegisters[6]
			} else {
				bank = m.prgBankCount - 2 // second-to-last bank
			}

		case addr >= 0xA000 && addr <= 0xBFFF:
			// $A000-$BFFF: Always R7
			bank = m.bankRegisters[7]

		case addr >= 0xC000 && addr <= 0xDFFF:
			// $C000-$DFFF: second-to-last in mode 0, R6 in mode 1
			if prgMode == 0 {
				bank = m.prgBankCount - 2 // second-to-last bank
			} else {
				bank = m.bankRegisters[6]
			}

		case addr >= 0xE000 && addr <= 0xFFFF:
			// $E000-$FFFF: Always last bank (this is crucial for reset vector)
			bank = m.prgBankCount - 1
		}

		// Ensure bank is in valid range
		if bank >= m.prgBankCount {
			bank = m.prgBankCount - 1
		}

		// Calculate offset within PRG ROM (8KB banks)
		offset := uint32(bank)*0x2000 + uint32(addr&0x1FFF)
		if offset < uint32(len(m.data.PRGROM)) {
			return m.data.PRGROM[offset]
		}
	}

	return 0
}

// WritePRG writes to PRG ROM/RAM address space or mapper registers.
// IRQ-register arms ($C000/$C001/$E000/$E001) delegate to helpers in
// mapper4_irq.go to keep IRQ-specific logic out of the bank-mapping file.
func (m *Mapper4) WritePRG(addr uint16, value uint8) {
	switch {
	case addr >= 0x6000 && addr <= 0x7FFF:
		// PRG RAM area - check write protection
		if len(m.data.PRGRAM) > 0 && (m.prgRAMProtect&0x80) != 0 && (m.prgRAMProtect&0x40) == 0 {
			m.data.PRGRAM[addr-0x6000] = value
		}

	case addr >= 0x8000:
		// Enable essential MMC3 registers according to NESdev spec
		switch addr & 0xE001 {
		case 0x8000: // Bank select ($8000-$9FFE, even)
			logger.LogMapper("MMC3 bank select set: %d", value)
			m.bankSelect = value
			m.recalcCHRBanks() // CHR mode bit 7 may have flipped

		case 0x8001: // Bank data ($8001-$9FFF, odd)
			regIndex := m.bankSelect & 0x07
			if regIndex < 8 {
				// Bounds check to prevent invalid bank access
				if regIndex >= 6 {
					// PRG bank registers (R6, R7) - clamp to valid range
					m.bankRegisters[regIndex] = value % m.prgBankCount
				} else {
					// CHR bank registers (R0-R5) - clamp to valid range
					if m.chrBankCount > 0 {
						m.bankRegisters[regIndex] = value % m.chrBankCount
					} else {
						m.bankRegisters[regIndex] = value
					}
				}
			}
			m.recalcCHRBanks() // refresh windows for the just-written CHR reg

		case 0xA000: // Mirroring ($A000-$BFFE, even)
			m.mirroringMode = value & 1

		case 0xA001: // PRG RAM protect ($A001-$BFFF, odd)
			m.prgRAMProtect = value

		case 0xC000: // IRQ latch ($C000-$DFFE, even)
			m.writeIRQLatch(value)

		case 0xC001: // IRQ reload ($C001-$DFFF, odd)
			m.writeIRQReload()

		case 0xE000: // IRQ disable ($E000-$FFFE, even)
			m.writeIRQDisable()

		case 0xE001: // IRQ enable ($E001-$FFFF, odd)
			m.writeIRQEnable()
		}
	}
}

// ReadCHR reads from CHR ROM/RAM address space via the precomputed per-window
// offset table (see recalcCHRBanks). CHR ROM takes priority; CHR RAM is the
// backing store when the cart ships no CHR ROM.
func (m *Mapper4) ReadCHR(addr uint16) uint8 {
	if addr >= 0x2000 {
		return 0
	}
	offset := m.chrWindowOffset[addr>>10] + uint32(addr&0x3FF)
	if int(offset) < len(m.data.CHRROM) {
		return m.data.CHRROM[offset]
	}
	if int(offset) < len(m.data.CHRRAM) {
		return m.data.CHRRAM[offset]
	}
	return 0
}

// recalcCHRBanks rebuilds chrWindowOffset from bankSelect (CHR mode bit 7) and
// the CHR bank registers R0-R5. The per-window bank numbers reproduce exactly
// what the old per-fetch calculateCHRBank produced — including the uint8 bank
// arithmetic and the `bank %= chrBankCount` clamp (which is skipped when
// chrBankCount is 0, e.g. a 256KB-CHR cart whose 1KB-bank count wraps uint8).
// Call after any change to bankSelect or bankRegisters.
func (m *Mapper4) recalcCHRBanks() {
	// R0/R1 are 2KB banks: the low bit is ignored and the pair spans two
	// consecutive 1KB windows (r, r+1).
	r0, r1 := m.bankRegisters[0]&^1, m.bankRegisters[1]&^1
	var banks [8]uint8
	if (m.bankSelect>>7)&1 == 0 {
		// Mode 0: $0000-$0FFF = R0,R1 (2KB each); $1000-$1FFF = R2..R5 (1KB)
		banks = [8]uint8{r0, r0 + 1, r1, r1 + 1,
			m.bankRegisters[2], m.bankRegisters[3], m.bankRegisters[4], m.bankRegisters[5]}
	} else {
		// Mode 1: $0000-$0FFF = R2..R5 (1KB); $1000-$1FFF = R0,R1 (2KB each)
		banks = [8]uint8{m.bankRegisters[2], m.bankRegisters[3], m.bankRegisters[4], m.bankRegisters[5],
			r0, r0 + 1, r1, r1 + 1}
	}
	for w := 0; w < 8; w++ {
		b := banks[w]
		if m.chrBankCount > 0 {
			b %= m.chrBankCount
		}
		m.chrWindowOffset[w] = uint32(b) * 0x400
	}
}

// WriteCHR writes to CHR RAM (CHR ROM is read-only) via the precomputed
// per-window offset table.
func (m *Mapper4) WriteCHR(addr uint16, value uint8) {
	if addr >= 0x2000 {
		return
	}
	if len(m.data.CHRRAM) > 0 {
		offset := m.chrWindowOffset[addr>>10] + uint32(addr&0x3FF)
		if int(offset) < len(m.data.CHRRAM) {
			m.data.CHRRAM[offset] = value
		}
	}
}
