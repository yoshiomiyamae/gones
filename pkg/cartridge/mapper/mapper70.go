package mapper

import (
	"encoding/binary"
	"io"
)

// Mapper70 (Bandai) — UxROM-like, but the bank register also selects an
// 8 KiB CHR ROM bank. Used by Family Trainer: Manhattan Police, Kamen
// Rider Club, and a handful of other Bandai titles.
//
// Bus map:
//
//	$8000-$BFFF  switchable 16 KiB PRG bank
//	$C000-$FFFF  fixed to last 16 KiB PRG bank
//	$0000-$1FFF  switchable 8 KiB CHR bank
//
// Bank register ($8000-$FFFF write, .BBB CCCC):
//
//	bits 0-3  CHR ROM bank (8 KiB)
//	bits 4-6  PRG ROM bank (16 KiB)
//	bit  7    unused on mapper 70 (single-screen on mapper 152)
//
// Mirroring is fixed by the iNES header — no mapper control.
type Mapper70 struct {
	cartridge *CartridgeData

	prgBank      uint8
	chrBank      uint8
	prgBankCount uint8
	chrBankCount uint8
}

// NewMapper70 creates a new Mapper70 instance.
func NewMapper70(data *CartridgeData) *Mapper70 {
	m := &Mapper70{cartridge: data}
	if len(data.PRGROM) > 0 {
		m.prgBankCount = uint8(len(data.PRGROM) / 16384)
	}
	if len(data.CHRROM) > 0 {
		m.chrBankCount = uint8(len(data.CHRROM) / 8192)
	}
	return m
}

// ReadPRG reads from PRG space.
func (m *Mapper70) ReadPRG(addr uint16) uint8 {
	if addr >= 0x8000 {
		var bank uint8
		var offset uint16
		if addr < 0xC000 {
			bank = m.prgBank % m.prgBankCount
			offset = addr - 0x8000
		} else {
			bank = m.prgBankCount - 1
			offset = addr - 0xC000
		}
		finalAddr := uint32(bank)*16384 + uint32(offset)
		if finalAddr < uint32(len(m.cartridge.PRGROM)) {
			return m.cartridge.PRGROM[finalAddr]
		}
		return 0
	}
	if addr >= 0x6000 {
		return readPRGRAM(m.cartridge, addr)
	}
	return 0
}

// WritePRG writes to PRG space (bank register at $8000-$FFFF).
func (m *Mapper70) WritePRG(addr uint16, value uint8) {
	if addr >= 0x8000 {
		m.chrBank = value & 0x0F
		m.prgBank = (value >> 4) & 0x07
		return
	}
	writePRGRAM(m.cartridge, addr, value)
}

// ReadCHR reads from CHR space with 8 KiB banking.
func (m *Mapper70) ReadCHR(addr uint16) uint8 {
	if m.chrBankCount > 0 {
		bank := m.chrBank % m.chrBankCount
		finalAddr := uint32(bank)*8192 + uint32(addr)
		if finalAddr < uint32(len(m.cartridge.CHRROM)) {
			return m.cartridge.CHRROM[finalAddr]
		}
		return 0
	}
	return readCHRROMOrRAM(m.cartridge, addr)
}

// WriteCHR writes to CHR space (CHR RAM only — most mapper-70 carts are CHR ROM).
func (m *Mapper70) WriteCHR(addr uint16, value uint8) {
	writeCHRRAM(m.cartridge, addr, value)
}

func (m *Mapper70) Step()              {}
func (m *Mapper70) IsIRQPending() bool { return false }
func (m *Mapper70) ClearIRQ()          {}

func (m *Mapper70) SaveState(w io.Writer) error {
	return binary.Write(w, binary.LittleEndian, [2]uint8{m.prgBank, m.chrBank})
}

func (m *Mapper70) LoadState(r io.Reader) error {
	var b [2]uint8
	if err := binary.Read(r, binary.LittleEndian, &b); err != nil {
		return err
	}
	m.prgBank, m.chrBank = b[0], b[1]
	return nil
}
