package mapper

import (
	"encoding/binary"
	"io"
)

// Mapper69 (Sunsoft FME-7) — used by Hebereke / Ufouria, Gimmick!,
// Dynamite Headdy and a handful of other late Sunsoft titles.
//
// Bus map:
//
//	$6000-$7FFF  switchable 8 KiB PRG (RAM or ROM, controlled by reg 8)
//	$8000-$9FFF  switchable 8 KiB PRG ROM (reg 9)
//	$A000-$BFFF  switchable 8 KiB PRG ROM (reg A)
//	$C000-$DFFF  switchable 8 KiB PRG ROM (reg B)
//	$E000-$FFFF  fixed to last 8 KiB PRG ROM bank
//
//	$0000-$1FFF  eight 1 KiB CHR ROM banks (regs 0-7)
//
// Two-byte register protocol:
//
//	$8000-$9FFF write: command (low 4 bits) — selects which internal
//	                   register the next $A000-$BFFF write targets
//	$A000-$BFFF write: data for the selected command
//	$C000-$DFFF write: audio register select (not implemented — only
//	                   Gimmick! uses the expansion sound channels)
//	$E000-$FFFF write: audio register data (ditto)
//
// Mirroring is mapper-controlled (reg C). The 16-bit IRQ counter
// (regs D-F) decrements every CPU cycle when enabled and fires on
// underflow; the test most cleanly exercised by Hebereke's intro
// timing.
type Mapper69 struct {
	cartridge *CartridgeData

	command uint8 // last $8000-write selected register

	prgRAMSelect uint8 // raw value written to reg 8 (RAM-enable + bank)
	prgBanks     [3]uint8 // regs 9, A, B → $8000/$A000/$C000
	chrBanks     [8]uint8 // regs 0-7 → $0000/$0400/.../$1C00

	mirroring uint8

	irqControl    uint8
	irqCounter    uint16
	irqPending    bool

	audio fme7Audio

	prgBankCount uint8
	chrBankCount uint16
}

func NewMapper69(data *CartridgeData) *Mapper69 {
	m := &Mapper69{cartridge: data}
	if len(data.PRGROM) > 0 {
		m.prgBankCount = uint8(len(data.PRGROM) / 8192)
	}
	if len(data.CHRROM) > 0 {
		m.chrBankCount = uint16(len(data.CHRROM) / 1024)
	}
	// $E000-$FFFF is hardware-fixed to the last PRG bank, so the reset
	// vector is reachable before the game programs any of regs 9/A/B.
	return m
}

func (m *Mapper69) ReadPRG(addr uint16) uint8 {
	switch {
	case addr >= 0xE000:
		return m.readPRGBank(m.prgBankCount-1, addr-0xE000)
	case addr >= 0xC000:
		return m.readPRGBank(m.prgBanks[2], addr-0xC000)
	case addr >= 0xA000:
		return m.readPRGBank(m.prgBanks[1], addr-0xA000)
	case addr >= 0x8000:
		return m.readPRGBank(m.prgBanks[0], addr-0x8000)
	case addr >= 0x6000:
		// Reg 8: bit 7 = enable (else open bus), bit 6 = RAM (else ROM).
		if m.prgRAMSelect&0x80 == 0 {
			return 0 // disabled — open bus on real hw, harmless 0 here
		}
		if m.prgRAMSelect&0x40 != 0 {
			return readPRGRAM(m.cartridge, addr)
		}
		return m.readPRGBank(m.prgRAMSelect&0x3F, addr-0x6000)
	}
	return 0
}

func (m *Mapper69) readPRGBank(rawBank uint8, offset uint16) uint8 {
	if m.prgBankCount == 0 {
		return 0
	}
	bank := (rawBank & 0x3F) % m.prgBankCount
	finalAddr := uint32(bank)*8192 + uint32(offset)
	if int(finalAddr) < len(m.cartridge.PRGROM) {
		return m.cartridge.PRGROM[finalAddr]
	}
	return 0
}

func (m *Mapper69) WritePRG(addr uint16, value uint8) {
	switch {
	case addr >= 0xE000:
		m.audio.writeAudioData(value)
	case addr >= 0xC000:
		m.audio.writeAudioSelect(value)
	case addr >= 0xA000:
		m.writeParam(value)
	case addr >= 0x8000:
		m.command = value & 0x0F
	case addr >= 0x6000:
		if m.prgRAMSelect&0xC0 == 0xC0 { // enabled + RAM
			writePRGRAM(m.cartridge, addr, value)
		}
	}
}

func (m *Mapper69) writeParam(value uint8) {
	switch m.command {
	case 0, 1, 2, 3, 4, 5, 6, 7:
		m.chrBanks[m.command] = value
	case 8:
		m.prgRAMSelect = value
	case 9, 10, 11:
		m.prgBanks[m.command-9] = value & 0x3F
	case 12:
		m.mirroring = value & 0x03
	case 13:
		// IRQ control. Bit 7 enables the counter, bit 0 enables IRQ
		// generation on underflow. Any write to this register also
		// acknowledges a pending IRQ.
		m.irqControl = value
		m.irqPending = false
	case 14:
		m.irqCounter = (m.irqCounter & 0xFF00) | uint16(value)
	case 15:
		m.irqCounter = (m.irqCounter & 0x00FF) | (uint16(value) << 8)
	}
}

func (m *Mapper69) ReadCHR(addr uint16) uint8 {
	if m.chrBankCount == 0 {
		return readCHRROMOrRAM(m.cartridge, addr)
	}
	region := (addr >> 10) & 0x07
	bank := uint16(m.chrBanks[region]) % m.chrBankCount
	finalAddr := uint32(bank)*1024 + uint32(addr&0x3FF)
	if int(finalAddr) < len(m.cartridge.CHRROM) {
		return m.cartridge.CHRROM[finalAddr]
	}
	return 0
}

func (m *Mapper69) WriteCHR(addr uint16, value uint8) {
	writeCHRRAM(m.cartridge, addr, value)
}

func (m *Mapper69) Step() {}

// TickCPU advances the 16-bit IRQ counter once per CPU cycle while the
// counter-enable bit (D7 of reg 13) is set. Underflow (counter $0000 →
// $FFFF) latches the IRQ if D0 is also set. Also drives the FME-7
// expansion-audio phase counters.
func (m *Mapper69) TickCPU(cycles int) {
	m.audio.tick(cycles)
	if m.irqControl&0x80 == 0 {
		return
	}
	for i := 0; i < cycles; i++ {
		if m.irqCounter == 0 {
			m.irqCounter = 0xFFFF
			if m.irqControl&0x01 != 0 {
				m.irqPending = true
			}
		} else {
			m.irqCounter--
		}
	}
}

// AudioSample exposes the current FME-7 expansion-sound output for the
// APU's mixer to layer on top of the 2A03 channels.
func (m *Mapper69) AudioSample() float32 {
	return m.audio.sample()
}

func (m *Mapper69) IsIRQPending() bool { return m.irqPending }
func (m *Mapper69) ClearIRQ()          { m.irqPending = false }

// IRQCapable marks FME-7 as an IRQ-asserting mapper (CPU-clock counter).
func (m *Mapper69) IRQCapable() {}

// GetMirroringMode reports the current mapper-controlled mirroring in
// the PPU's encoding (0=horizontal, 1=vertical, 2/3=single-screen
// lower/upper). FME-7 reg-12 codes: 0=vert, 1=horiz, 2=lower, 3=upper.
func (m *Mapper69) GetMirroringMode() uint8 {
	switch m.mirroring {
	case 0:
		return 1 // vertical
	case 1:
		return 0 // horizontal
	case 2:
		return 2 // single-screen lower
	default:
		return 3 // single-screen upper
	}
}

type mapper69State struct {
	Command, PRGRAMSelect, Mirroring, IRQControl uint8
	PRGBanks                                     [3]uint8
	CHRBanks                                     [8]uint8
	IRQCounter                                   uint16
	IRQPending                                   bool
}

func (m *Mapper69) SaveState(w io.Writer) error {
	return binary.Write(w, binary.LittleEndian, mapper69State{
		Command:      m.command,
		PRGRAMSelect: m.prgRAMSelect,
		Mirroring:    m.mirroring,
		IRQControl:   m.irqControl,
		PRGBanks:     m.prgBanks,
		CHRBanks:     m.chrBanks,
		IRQCounter:   m.irqCounter,
		IRQPending:   m.irqPending,
	})
}

func (m *Mapper69) LoadState(r io.Reader) error {
	var s mapper69State
	if err := binary.Read(r, binary.LittleEndian, &s); err != nil {
		return err
	}
	m.command = s.Command
	m.prgRAMSelect = s.PRGRAMSelect
	m.mirroring = s.Mirroring
	m.irqControl = s.IRQControl
	m.prgBanks = s.PRGBanks
	m.chrBanks = s.CHRBanks
	m.irqCounter = s.IRQCounter
	m.irqPending = s.IRQPending
	return nil
}
