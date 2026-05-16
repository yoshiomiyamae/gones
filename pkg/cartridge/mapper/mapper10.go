package mapper

import (
	"encoding/binary"
	"io"
)

// Mapper10 implements MMC4 (FxROM). Used by Fire Emblem Gaiden, Famicom
// Wars, etc. The defining feature is the CHR "latch" system: each pattern
// table has two selectable 4KB banks, and which one is active flips
// automatically when the PPU fetches tile $FD or $FE — letting games swap
// CHR data mid-frame without explicit register writes.
//
// Registers (write):
//
//	$A000-$AFFF: PRG bank (16KB at $8000); last 16KB fixed at $C000
//	$B000-$BFFF: CHR bank for $0000-$0FFF when latch0 == $FD
//	$C000-$CFFF: CHR bank for $0000-$0FFF when latch0 == $FE
//	$D000-$DFFF: CHR bank for $1000-$1FFF when latch1 == $FD
//	$E000-$EFFF: CHR bank for $1000-$1FFF when latch1 == $FE
//	$F000-$FFFF: Mirroring (bit 0: 0=vertical, 1=horizontal)
//
// Latch transitions (on PPU CHR read):
//
//	$0FD8-$0FDF -> latch0 = $FD
//	$0FE8-$0FEF -> latch0 = $FE
//	$1FD8-$1FDF -> latch1 = $FD
//	$1FE8-$1FEF -> latch1 = $FE
type Mapper10 struct {
	cartridge *CartridgeData

	prgBank      uint8 // lower 4 bits select 16KB bank at $8000
	prgBankCount uint8

	chrBank0FD uint8 // $0000-$0FFF when latch0 == $FD
	chrBank0FE uint8 // $0000-$0FFF when latch0 == $FE
	chrBank1FD uint8 // $1000-$1FFF when latch1 == $FD
	chrBank1FE uint8 // $1000-$1FFF when latch1 == $FE
	chrBankCount uint8

	latch0 uint8 // $FD or $FE — controls $0000-$0FFF bank selection
	latch1 uint8 // $FD or $FE — controls $1000-$1FFF bank selection

	mirroring uint8 // raw MMC4 bit (0=vertical, 1=horizontal)
}

// NewMapper10 creates a new MMC4 mapper.
func NewMapper10(data *CartridgeData) *Mapper10 {
	m := &Mapper10{
		cartridge: data,
		// Initial latch state is undefined on real hardware; $FE is what
		// most documentation defaults to and matches MMC4 boot ROM patterns.
		latch0: 0xFE,
		latch1: 0xFE,
	}
	m.prgBankCount = uint8(len(data.PRGROM) / 16384)
	if len(data.CHRROM) > 0 {
		m.chrBankCount = uint8(len(data.CHRROM) / 4096)
	}
	return m
}

// ReadPRG reads from PRG space. $8000-$BFFF is switchable; $C000-$FFFF is
// fixed to the last 16KB bank. $6000-$7FFF is PRG RAM if present.
func (m *Mapper10) ReadPRG(addr uint16) uint8 {
	switch {
	case addr >= 0xC000:
		// Last 16KB bank fixed at $C000-$FFFF
		offset := uint32(m.prgBankCount-1)*16384 + uint32(addr-0xC000)
		if offset < uint32(len(m.cartridge.PRGROM)) {
			return m.cartridge.PRGROM[offset]
		}
	case addr >= 0x8000:
		// Switchable 16KB at $8000-$BFFF
		bank := m.prgBank
		if m.prgBankCount > 0 {
			bank %= m.prgBankCount
		}
		offset := uint32(bank)*16384 + uint32(addr-0x8000)
		if offset < uint32(len(m.cartridge.PRGROM)) {
			return m.cartridge.PRGROM[offset]
		}
	case addr >= 0x6000:
		return readPRGRAM(m.cartridge, addr)
	}
	return 0
}

// WritePRG dispatches register writes ($A000-$FFFF) and handles PRG RAM
// writes ($6000-$7FFF). Writes to $8000-$9FFF are ignored (MMC4 reserves
// that range for the audio coprocessor in the FxROM-W variant, which we
// don't emulate).
func (m *Mapper10) WritePRG(addr uint16, value uint8) {
	switch {
	case addr >= 0xF000:
		m.mirroring = value & 1
	case addr >= 0xE000:
		m.chrBank1FE = value & 0x1F
	case addr >= 0xD000:
		m.chrBank1FD = value & 0x1F
	case addr >= 0xC000:
		m.chrBank0FE = value & 0x1F
	case addr >= 0xB000:
		m.chrBank0FD = value & 0x1F
	case addr >= 0xA000:
		m.prgBank = value & 0x0F
	case addr >= 0x6000 && addr < 0x8000:
		writePRGRAM(m.cartridge, addr, value)
	}
}

// ReadCHR fetches a CHR byte and then updates the latch state if this
// address falls in one of the trigger ranges. The fetch uses the *current*
// latch — the trigger applies to subsequent reads (per NESdev MMC4 docs).
func (m *Mapper10) ReadCHR(addr uint16) uint8 {
	value := m.chrFetch(addr)

	switch {
	case addr >= 0x0FD8 && addr <= 0x0FDF:
		m.latch0 = 0xFD
	case addr >= 0x0FE8 && addr <= 0x0FEF:
		m.latch0 = 0xFE
	case addr >= 0x1FD8 && addr <= 0x1FDF:
		m.latch1 = 0xFD
	case addr >= 0x1FE8 && addr <= 0x1FEF:
		m.latch1 = 0xFE
	}

	return value
}

// chrFetch picks the active 4KB bank using the current latch state and
// returns the byte. Address bit 12 selects which pattern table half and
// thus which latch / register pair to consult.
func (m *Mapper10) chrFetch(addr uint16) uint8 {
	var bank uint8
	if addr < 0x1000 {
		if m.latch0 == 0xFD {
			bank = m.chrBank0FD
		} else {
			bank = m.chrBank0FE
		}
	} else {
		if m.latch1 == 0xFD {
			bank = m.chrBank1FD
		} else {
			bank = m.chrBank1FE
		}
	}
	if m.chrBankCount > 0 {
		bank %= m.chrBankCount
	}
	offset := uint32(bank)*4096 + uint32(addr&0x0FFF)
	if offset < uint32(len(m.cartridge.CHRROM)) {
		return m.cartridge.CHRROM[offset]
	}
	if int(offset) < len(m.cartridge.CHRRAM) {
		return m.cartridge.CHRRAM[offset]
	}
	return 0
}

// WriteCHR writes to CHR RAM if present. MMC4 carts usually have CHR ROM
// (read-only), so this is mostly a no-op.
func (m *Mapper10) WriteCHR(addr uint16, value uint8) {
	if len(m.cartridge.CHRRAM) == 0 || addr >= 0x2000 {
		return
	}
	if int(addr) < len(m.cartridge.CHRRAM) {
		m.cartridge.CHRRAM[addr] = value
	}
}

// Step is a no-op: MMC4 has no IRQ counter or scanline timing.
func (m *Mapper10) Step() {}

// IsIRQPending always false: MMC4 has no IRQ source.
func (m *Mapper10) IsIRQPending() bool { return false }

// ClearIRQ is a no-op.
func (m *Mapper10) ClearIRQ() {}

// GetMirroringMode returns the PPU-encoded mirroring (0=horizontal,
// 1=vertical). MMC4 stores the inverse: bit 0 = 0 means vertical.
func (m *Mapper10) GetMirroringMode() uint8 {
	if m.mirroring == 0 {
		return 1 // vertical
	}
	return 0 // horizontal
}

type mapper10State struct {
	PrgBank                                    uint8
	ChrBank0FD, ChrBank0FE, ChrBank1FD, ChrBank1FE uint8
	Latch0, Latch1                             uint8
	Mirroring                                  uint8
}

// SaveState persists bank registers, latches, and mirroring.
func (m *Mapper10) SaveState(w io.Writer) error {
	return binary.Write(w, binary.LittleEndian, mapper10State{
		PrgBank:    m.prgBank,
		ChrBank0FD: m.chrBank0FD, ChrBank0FE: m.chrBank0FE,
		ChrBank1FD: m.chrBank1FD, ChrBank1FE: m.chrBank1FE,
		Latch0: m.latch0, Latch1: m.latch1,
		Mirroring: m.mirroring,
	})
}

// LoadState restores state written by SaveState.
func (m *Mapper10) LoadState(r io.Reader) error {
	var s mapper10State
	if err := binary.Read(r, binary.LittleEndian, &s); err != nil {
		return err
	}
	m.prgBank = s.PrgBank
	m.chrBank0FD, m.chrBank0FE = s.ChrBank0FD, s.ChrBank0FE
	m.chrBank1FD, m.chrBank1FE = s.ChrBank1FD, s.ChrBank1FE
	m.latch0, m.latch1 = s.Latch0, s.Latch1
	m.mirroring = s.Mirroring
	return nil
}
