package mapper

import (
	"encoding/binary"
	"io"
)

// Mapper5 (MMC5 / ExROM) is Nintendo's last and most ambitious official
// NES mapper — used by Castlevania III, Metal Slader Glory, Just Breed,
// Bandit Kings of Ancient China, and a handful of other late JP titles.
//
// Bus map (PRG side):
//
//	$5000-$5015 expansion-audio register file (2 squares + PCM) — TODO
//	$5100-$5107 mode / mirroring / fill / ExRAM-mode control
//	$5113-$5117 PRG bank registers (interpretation depends on $5100)
//	$5120-$5127 CHR 'A' bank registers (sprites in 8x16 mode; all fetches in 8x8)
//	$5128-$512B CHR 'B' bank registers (BG in 8x16 mode)
//	$5130       upper CHR bank bits (for >256 KiB CHR)
//	$5200-$5202 vertical-split-mode control — TODO
//	$5203       IRQ target scanline
//	$5204       IRQ status (read clears pending) / enable (write bit 7)
//	$5205-$5206 8-bit × 8-bit multiplier
//	$5C00-$5FFF 1 KiB ExRAM (mode-dependent)
//	$6000-$7FFF 8 KiB PRG RAM (banked by $5113)
//	$8000-$FFFF PRG banks per $5100 mode
//
// What's implemented today:
//   * PRG modes 0-3 with RAM/ROM bit on each bank register
//   * CHR modes 0-3, dual 'A'/'B' sets (8x16 routes BG→B, sprites→A)
//   * Per-nametable mirroring control via $5105 (NT0/NT1/ExRAM/Fill)
//   * Scanline-match IRQ driven by the PPU's per-scanline tick
//   * 8×8 → 16-bit multiplier
//   * ExRAM at $5C00 in modes 2/3 (CPU R/W backing)
//
// Not implemented yet: expansion audio, vertical split mode,
// ExRAM-mode 0/1 extended attribute lookups, Fill mode rendering.
type Mapper5 struct {
	cartridge *CartridgeData

	prgMode    uint8 // $5100: 0=32KB, 1=16KB, 2=16+8, 3=all-8KB
	chrMode    uint8 // $5101: 0=8KB, 1=4KB, 2=2KB, 3=1KB
	prgRAMW1   uint8 // $5102: write low — $02 unlocks PRG RAM writes
	prgRAMW2   uint8 // $5103: write high — $01 unlocks PRG RAM writes
	exRAMMode  uint8 // $5104
	ntMapping  uint8 // $5105
	fillTile   uint8 // $5106
	fillAttrib uint8 // $5107

	prgBanks [5]uint8  // $5113-$5117 (5113 = $6000 PRG RAM bank)
	chrA     [8]uint16 // $5120-$5127 (sprite-set / 8x8 unified) — full 10-bit bank
	chrB     [4]uint16 // $5128-$512B (BG-set in 8x16 mode)        — full 10-bit bank
	chrHigh  uint8     // $5130: top 2 bits ORed into the next CHR-bank write

	// IRQ scanline counter — driven by NotifyScanline (not A12). NESdev
	// MMC5 page calls these the "in-frame" and "match" flags.
	irqTarget  uint8
	irqEnable  bool
	irqPending bool
	inFrame    bool
	scanline   uint8

	// 8×8 multiplier ($5205 / $5206).
	multA, multB uint8

	exRAM [1024]uint8

	prgBankCount uint8  // count of 8 KiB PRG ROM banks
	chrBankCount uint16 // count of 1 KiB CHR ROM banks

	// sprite8x16 mirrors PPUCTRL bit 5 (propagated via SetSpriteSize).
	// When set, BG fetches route through the 'B' set ($5128-$512B) and
	// sprite fetches through the 'A' set ($5120-$5127); when clear,
	// both share the 'A' set.
	sprite8x16 bool
}

// NewMapper5 constructs a fresh MMC5 around the cartridge's PRG/CHR
// ROM. PRG RAM is allocated by the cartridge layer.
func NewMapper5(data *CartridgeData) *Mapper5 {
	m := &Mapper5{cartridge: data}
	if len(data.PRGROM) > 0 {
		m.prgBankCount = uint8(len(data.PRGROM) / 8192)
	}
	if len(data.CHRROM) > 0 {
		m.chrBankCount = uint16(len(data.CHRROM) / 1024)
	}
	// Power-on PRG layout: hardware boots with mode 3 ($8000-$FFFF =
	// four 8 KiB ROM slots from the last bank). Setting $5117 to the
	// last bank index makes the reset vector reachable before the
	// game programs anything.
	m.prgMode = 3
	if m.prgBankCount > 0 {
		m.prgBanks[4] = (m.prgBankCount - 1) | 0x80 // ROM + last bank
	}
	return m
}

// ReadPRG dispatches CPU reads of $4020-$FFFF through the bank
// registers. $5000-$5FFF holds the MMC5 register file plus ExRAM.
func (m *Mapper5) ReadPRG(addr uint16) uint8 {
	switch {
	case addr >= 0x8000:
		return m.readMappedPRG(addr)
	case addr >= 0x6000:
		bank := m.prgBanks[0] & 0x7F
		offset := uint32(bank)*8192 + uint32(addr-0x6000)
		if len(m.cartridge.PRGRAM) > 0 {
			i := offset % uint32(len(m.cartridge.PRGRAM))
			return m.cartridge.PRGRAM[i]
		}
		return 0
	case addr >= 0x5C00:
		// ExRAM read. In modes 0-1 only the PPU reads it; in mode 2
		// it's general-purpose RAM; in mode 3 it's read-only RAM.
		return m.exRAM[addr-0x5C00]
	case addr == 0x5204:
		v := uint8(0)
		if m.irqPending {
			v |= 0x80
		}
		if m.inFrame {
			v |= 0x40
		}
		m.irqPending = false
		return v
	case addr == 0x5205:
		return uint8(uint16(m.multA) * uint16(m.multB))
	case addr == 0x5206:
		return uint8((uint16(m.multA) * uint16(m.multB)) >> 8)
	}
	return 0
}

func (m *Mapper5) readMappedPRG(addr uint16) uint8 {
	region, bankSize := m.prgRegion(addr)
	raw := m.prgBanks[region]
	isROM := raw&0x80 != 0
	// $5117 always sources ROM; the bit is just ignored there.
	if region == 4 {
		isROM = true
	}
	bank := raw & 0x7F
	switch bankSize {
	case 8192:
		// 8 KiB granularity — bank is already in 8 KiB units.
	case 16384:
		bank &^= 1
	case 32768:
		bank &^= 3
	}
	offset := uint32(bank)*8192 + uint32(addr-m.prgRegionBase(region, bankSize))
	if !isROM {
		if len(m.cartridge.PRGRAM) == 0 {
			return 0
		}
		i := offset % uint32(len(m.cartridge.PRGRAM))
		return m.cartridge.PRGRAM[i]
	}
	if m.prgBankCount == 0 {
		return 0
	}
	mask := uint32(m.prgBankCount)*8192 - 1
	return m.cartridge.PRGROM[offset&mask]
}

// prgRegion returns which $5114-$5117 slot a given address falls into
// and the size of that slot under the current PRG mode.
func (m *Mapper5) prgRegion(addr uint16) (region int, size int) {
	switch m.prgMode {
	case 0:
		// $8000-$FFFF as a single 32 KiB bank via $5117.
		return 4, 32768
	case 1:
		// $8000-$BFFF = $5115 (16 KiB), $C000-$FFFF = $5117 (16 KiB).
		if addr < 0xC000 {
			return 2, 16384
		}
		return 4, 16384
	case 2:
		// $8000-$BFFF = $5115 (16 KiB), $C000-$DFFF = $5116 (8 KiB),
		// $E000-$FFFF = $5117 (8 KiB).
		switch {
		case addr < 0xC000:
			return 2, 16384
		case addr < 0xE000:
			return 3, 8192
		default:
			return 4, 8192
		}
	default:
		// Mode 3: four 8 KiB slots.
		switch {
		case addr < 0xA000:
			return 1, 8192
		case addr < 0xC000:
			return 2, 8192
		case addr < 0xE000:
			return 3, 8192
		default:
			return 4, 8192
		}
	}
}

// prgRegionBase is the CPU address where the slot for `region` starts.
func (m *Mapper5) prgRegionBase(region, size int) uint16 {
	switch size {
	case 32768:
		return 0x8000
	case 16384:
		if region == 2 {
			return 0x8000
		}
		return 0xC000
	default:
		switch region {
		case 1:
			return 0x8000
		case 2:
			return 0xA000
		case 3:
			return 0xC000
		default:
			return 0xE000
		}
	}
}

// WritePRG handles writes to $4020-$FFFF. $5000-$5FFF is the MMC5
// register file; $6000-$7FFF is PRG RAM (write-gated by $5102/$5103);
// $8000-$FFFF is normally ROM and writes are ignored.
func (m *Mapper5) WritePRG(addr uint16, value uint8) {
	switch {
	case addr >= 0x8000:
		// Cartridge ROM — ignore.
		return
	case addr >= 0x6000:
		if m.prgRAMUnlocked() && len(m.cartridge.PRGRAM) > 0 {
			bank := m.prgBanks[0] & 0x7F
			offset := uint32(bank)*8192 + uint32(addr-0x6000)
			i := offset % uint32(len(m.cartridge.PRGRAM))
			m.cartridge.PRGRAM[i] = value
		}
		return
	case addr >= 0x5C00:
		// ExRAM write. Mode 3 = read-only, so writes are dropped
		// unless we're in modes 0-2.
		if m.exRAMMode < 3 {
			m.exRAM[addr-0x5C00] = value
		}
		return
	}

	m.writeReg(addr, value)
}

func (m *Mapper5) prgRAMUnlocked() bool {
	return m.prgRAMW1&0x03 == 0x02 && m.prgRAMW2&0x03 == 0x01
}

func (m *Mapper5) writeReg(addr uint16, value uint8) {
	switch {
	case addr == 0x5100:
		m.prgMode = value & 0x03
	case addr == 0x5101:
		m.chrMode = value & 0x03
	case addr == 0x5102:
		m.prgRAMW1 = value
	case addr == 0x5103:
		m.prgRAMW2 = value
	case addr == 0x5104:
		m.exRAMMode = value & 0x03
	case addr == 0x5105:
		m.ntMapping = value
	case addr == 0x5106:
		m.fillTile = value
	case addr == 0x5107:
		m.fillAttrib = value & 0x03
	case addr == 0x5113:
		m.prgBanks[0] = value
	case addr >= 0x5114 && addr <= 0x5117:
		m.prgBanks[1+(addr-0x5114)] = value
	case addr >= 0x5120 && addr <= 0x5127:
		// CHR bank registers are 10 bits wide: low 8 come from the data
		// byte, top 2 come from the value latched in $5130 at the time
		// of the write. 512+ KiB CHR (Metal Slader Glory = 512 KiB)
		// uses the high bits to reach all 1-KiB slots.
		m.chrA[addr-0x5120] = uint16(value) | uint16(m.chrHigh)<<8
	case addr >= 0x5128 && addr <= 0x512B:
		m.chrB[addr-0x5128] = uint16(value) | uint16(m.chrHigh)<<8
	case addr == 0x5130:
		m.chrHigh = value & 0x03
	case addr == 0x5203:
		m.irqTarget = value
	case addr == 0x5204:
		m.irqEnable = value&0x80 != 0
	case addr == 0x5205:
		m.multA = value
	case addr == 0x5206:
		m.multB = value
	}
}

// ReadCHR routes a BG-side PPU pattern fetch ($0000-$1FFF) through the
// CHR banks. In 8×16 sprite mode the 'B' set ($5128-$512B) is used for
// BG; otherwise the 'A' set serves both halves.
func (m *Mapper5) ReadCHR(addr uint16) uint8 {
	if m.chrBankCount == 0 || len(m.cartridge.CHRROM) == 0 {
		return readCHRROMOrRAM(m.cartridge, addr)
	}
	if m.sprite8x16 {
		return m.fetchCHRFromBSet(addr)
	}
	return m.fetchCHRFromASet(addr)
}

// ReadCHRSprite routes a sprite-side pattern fetch. In 8×16 mode the
// 'A' set is used for sprites; in 8×8 mode it's the unified set (also
// 'A'). The two paths are functionally identical when 8×8 — keeping
// the method for symmetry / future expansion.
func (m *Mapper5) ReadCHRSprite(addr uint16) uint8 {
	if m.chrBankCount == 0 || len(m.cartridge.CHRROM) == 0 {
		return readCHRROMOrRAM(m.cartridge, addr)
	}
	return m.fetchCHRFromASet(addr)
}

func (m *Mapper5) fetchCHRFromASet(addr uint16) uint8 {
	var bank uint32
	var inBank uint16
	switch m.chrMode {
	case 0:
		bank = uint32(m.chrA[7]) * 8
		inBank = addr & 0x1FFF
	case 1:
		idx := 3
		if addr&0x1000 != 0 {
			idx = 7
		}
		bank = uint32(m.chrA[idx]) * 4
		inBank = addr & 0x0FFF
	case 2:
		idx := int((addr>>11)&0x03)*2 + 1
		bank = uint32(m.chrA[idx]) * 2
		inBank = addr & 0x07FF
	default:
		idx := int((addr >> 10) & 0x07)
		bank = uint32(m.chrA[idx])
		inBank = addr & 0x03FF
	}
	finalAddr := (bank*1024 + uint32(inBank)) % uint32(len(m.cartridge.CHRROM))
	return m.cartridge.CHRROM[finalAddr]
}

// fetchCHRFromBSet mirrors the 'A'-set selection logic but pulls banks
// from the 4-register 'B' set. The 'B' set's entries mirror "the
// second half" of the 'A' set, so each chrMode picks the same bank
// for both $0000-$0FFF and $1000-$1FFF — see NESdev's MMC5 chart.
func (m *Mapper5) fetchCHRFromBSet(addr uint16) uint8 {
	var bank uint32
	var inBank uint16
	switch m.chrMode {
	case 0:
		bank = uint32(m.chrB[3]) * 8
		inBank = addr & 0x1FFF
	case 1:
		bank = uint32(m.chrB[3]) * 4
		inBank = addr & 0x0FFF
	case 2:
		idx := int((addr >> 11) & 0x01) // selects $5129 or $512B
		idx = idx*2 + 1
		bank = uint32(m.chrB[idx]) * 2
		inBank = addr & 0x07FF
	default:
		idx := int((addr >> 10) & 0x03)
		bank = uint32(m.chrB[idx])
		inBank = addr & 0x03FF
	}
	finalAddr := (bank*1024 + uint32(inBank)) % uint32(len(m.cartridge.CHRROM))
	return m.cartridge.CHRROM[finalAddr]
}

// SetSpriteSize is called from the PPU whenever PPUCTRL bit 5 toggles.
// MMC5 needs this to route BG fetches through the 'B' set in 8×16
// mode.
func (m *Mapper5) SetSpriteSize(is8x16 bool) {
	m.sprite8x16 = is8x16
}

// WriteCHR routes to CHR RAM when present. MMC5 carts ship CHR ROM,
// so this is mostly a no-op; the helper keeps test ROMs that drop
// CHR-RAM mode working.
func (m *Mapper5) WriteCHR(addr uint16, value uint8) {
	writeCHRRAM(m.cartridge, addr, value)
}

// Step is unused — IRQ timing is driven via the A12 notification path.
func (m *Mapper5) Step() {}

// DecodesExpansion marks MMC5 as a mapper that claims the cartridge
// expansion address range ($4020-$5FFF). Memory routes CPU R/W in that
// window to the mapper instead of falling back to open bus.
func (m *Mapper5) DecodesExpansion() {}

// NotifyA12 — MMC5 doesn't drive its scanline counter from A12 (games
// like Metal Slader Glory share a single CHR pattern table between
// BG and sprites and never flip A12 during normal rendering). The
// per-scanline tick comes from NotifyScanline instead.
func (m *Mapper5) NotifyA12(chrAddr uint16, renderingEnabled bool) {}

// NotifyScanline drives the MMC5 scanline counter. The "in-frame"
// flag rises on scanline 0 and falls when the post-render scanline
// fires (scanline 240 isn't a render line, so the next notify after
// 239 is the next frame's scanline 0). The IRQ pending bit latches
// when the counter matches $5203.
func (m *Mapper5) NotifyScanline(scanline int, renderingEnabled bool) {
	if !renderingEnabled {
		m.inFrame = false
		return
	}
	if scanline == 0 {
		m.inFrame = true
		m.scanline = 0
	} else if m.inFrame {
		m.scanline++
	}
	if m.scanline == m.irqTarget && m.irqTarget != 0 {
		m.irqPending = true
	}
}

// IsIRQPending reports the latched IRQ status. MMC5 only asserts the
// CPU IRQ line when both the per-scanline match flag and the enable
// bit ($5204 bit 7) are set.
func (m *Mapper5) IsIRQPending() bool {
	return m.irqPending && m.irqEnable
}

// IRQCapable marks MMC5 as an IRQ-asserting mapper (scanline-match IRQ).
func (m *Mapper5) IRQCapable() {}

// ClearIRQ is a no-op for MMC5 — the IRQ status is consumed via $5204
// reads, which the CPU does explicitly to ack the interrupt.
func (m *Mapper5) ClearIRQ() {}

// GetMirroringMode maps $5105's 8-bit per-NT field ($2000/$2400/$2800/$2C00,
// 2 bits each) onto the PPU's four scheme codes. Mixed configurations using
// ExRAM or Fill as a nametable can't be expressed in those codes, so they
// fall back to vertical — proper support needs per-NT routing in the PPU.
func (m *Mapper5) GetMirroringMode() uint8 {
	nt0 := m.ntMapping & 0x03
	nt1 := (m.ntMapping >> 2) & 0x03
	nt2 := (m.ntMapping >> 4) & 0x03
	nt3 := (m.ntMapping >> 6) & 0x03
	switch {
	case nt0 == 0 && nt1 == 0 && nt2 == 1 && nt3 == 1:
		return 0 // horizontal ($2000/$2400 → NT0, $2800/$2C00 → NT1)
	case nt0 == 0 && nt1 == 1 && nt2 == 0 && nt3 == 1:
		return 1 // vertical ($2000/$2800 → NT0, $2400/$2C00 → NT1)
	case nt0 == 0 && nt1 == 0 && nt2 == 0 && nt3 == 0:
		return 2 // single-screen lower
	case nt0 == 1 && nt1 == 1 && nt2 == 1 && nt3 == 1:
		return 3 // single-screen upper
	}
	return 1 // default approximation
}

type mapper5State struct {
	PRGMode, CHRMode, PRGRAMW1, PRGRAMW2, ExRAMMode, NTMapping uint8
	FillTile, FillAttrib, CHRHigh                              uint8
	PRGBanks                                                   [5]uint8
	CHRA                                                       [8]uint16
	CHRB                                                       [4]uint16
	IRQTarget                                                  uint8
	IRQEnable, IRQPending, InFrame                             bool
	Scanline                                                   uint8
	MultA, MultB                                               uint8
	ExRAM                                                      [1024]uint8
}

func (m *Mapper5) SaveState(w io.Writer) error {
	return binary.Write(w, binary.LittleEndian, mapper5State{
		PRGMode: m.prgMode, CHRMode: m.chrMode,
		PRGRAMW1: m.prgRAMW1, PRGRAMW2: m.prgRAMW2,
		ExRAMMode: m.exRAMMode, NTMapping: m.ntMapping,
		FillTile: m.fillTile, FillAttrib: m.fillAttrib, CHRHigh: m.chrHigh,
		PRGBanks: m.prgBanks, CHRA: m.chrA, CHRB: m.chrB,
		IRQTarget: m.irqTarget, IRQEnable: m.irqEnable, IRQPending: m.irqPending,
		InFrame: m.inFrame, Scanline: m.scanline,
		MultA: m.multA, MultB: m.multB,
		ExRAM: m.exRAM,
	})
}

func (m *Mapper5) LoadState(r io.Reader) error {
	var s mapper5State
	if err := binary.Read(r, binary.LittleEndian, &s); err != nil {
		return err
	}
	m.prgMode, m.chrMode = s.PRGMode, s.CHRMode
	m.prgRAMW1, m.prgRAMW2 = s.PRGRAMW1, s.PRGRAMW2
	m.exRAMMode, m.ntMapping = s.ExRAMMode, s.NTMapping
	m.fillTile, m.fillAttrib, m.chrHigh = s.FillTile, s.FillAttrib, s.CHRHigh
	m.prgBanks, m.chrA, m.chrB = s.PRGBanks, s.CHRA, s.CHRB
	m.irqTarget = s.IRQTarget
	m.irqEnable, m.irqPending = s.IRQEnable, s.IRQPending
	m.inFrame, m.scanline = s.InFrame, s.Scanline
	m.multA, m.multB = s.MultA, s.MultB
	m.exRAM = s.ExRAM
	return nil
}
