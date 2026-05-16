// CPU-visible PPU register interface ($2000-$2007).
//
// The $2005/$2006 first/second-write toggle (`w`), the $2007 stale-byte read
// latch (`readBuffer`), and the loopy-register dance over `v`/`t`/`x`/`xTemp`
// all live here. The bit-twiddling sequences are deliberately verbatim from
// the NESdev wiki PPU scrolling page — split-screen effects (e.g. SMB3 title)
// depend on the exact loopy semantics, so do not paraphrase.

package ppu

import (
	"github.com/yoshiomiyamaegones/pkg/logger"
)

// ReadRegister reads from a PPU register. The per-case open-bus refresh
// masks (5-7 for $2002, 0-5 for $2007 palette, full for $2004/$2007
// non-palette, none for write-only registers) match the table in blargg's
// ppu_open_bus readme; see refreshOpenBus/readOpenBus in ppu.go.
func (p *PPU) ReadRegister(addr uint16) uint8 {
	switch addr {
	case 0x2002: // PPUSTATUS
		// VBL race: a $2002 read on the cycle just before our VBL-flag set
		// transition (which lands at the (240, 340) → (241, 0) wrap in our
		// model — see PPU.vblSuppressed) suppresses the flag set and NMI
		// for this frame, matching real-hardware behaviour.
		if p.Scanline == 240 && p.Cycle == 340 {
			p.vblSuppressed = true
		}
		statusBits := p.PPUSTATUS & 0xE0
		logger.LogPPU("Read PPUSTATUS: $%02X", statusBits)
		p.refreshOpenBus(statusBits, 0xE0)
		p.PPUSTATUS &^= PPUSTATUSVBlank
		p.w = 0
		return statusBits | (p.readOpenBus() & 0x1F)
	case 0x2004: // OAMDATA
		value := p.OAM[p.OAMADDR]
		// Sprite attribute byte: bits 2-4 are unimplemented and read as 0.
		if p.OAMADDR&3 == 2 {
			value &= 0xE3
		}
		p.refreshOpenBus(value, 0xFF)
		return value
	case 0x2007: // PPUDATA
		var value uint8
		// The PPU bus is 14-bit; bits 14-15 of v are ignored when deciding
		// whether the access falls inside the palette region. Without this
		// mask, an access at v=$4000 (after an increment from $3FFF) reads
		// as palette and bypasses the buffered-read latch (blargg's
		// test 57: "Setting PPU address to 3FFF & reading $2007 thrice
		// should give the contents of $0000").
		readAddr := p.v & 0x3FFF
		palette := readAddr >= 0x3F00

		if palette {
			value = p.readVRAM(readAddr) & 0x3F
			// Buffer fills with the mirrored nametable byte at addr-$1000.
			p.readBuffer = p.readVRAM(readAddr - 0x1000)
		} else {
			value = p.readBuffer
			p.readBuffer = p.readVRAM(readAddr)
		}

		if p.v <= 0x000F {
			logger.LogPPU("$2007 Read CHR: vramAddr=$%04X, value=$%02X, buffer=$%02X", p.v, value, p.readBuffer)
		}

		p.incrementVRAMAddress()
		p.notifyCartridgeA12()

		if palette {
			// Palette only drives bits 0-5; bits 6-7 come from open bus.
			p.refreshOpenBus(value, 0x3F)
			return value | (p.readOpenBus() & 0xC0)
		}
		p.refreshOpenBus(value, 0xFF)
		return value
	}
	// $2000, $2001, $2003, $2005, $2006 are write-only — reads return the
	// current decay-register value without refreshing it.
	return p.readOpenBus()
}

// WriteRegister writes to PPU register
func (p *PPU) WriteRegister(addr uint16, value uint8) {
	// Any write to a PPU register puts the value onto the CPU-side data
	// bus, refreshing all 8 bits of the decay register (including for
	// $2002, which has no normal write effect but still drives the bus).
	p.refreshOpenBus(value, 0xFF)
	switch addr {
	case 0x2000: // PPUCTRL
		oldValue := p.PPUCTRL
		p.PPUCTRL = value
		p.t = (p.t & 0xF3FF) | ((uint16(value) & 0x03) << 10)
		logger.LogPPU("Write PPUCTRL: $%02X -> $%02X (NMI=%v, BG_table=$%04X, Sprite_table=$%04X)",
			oldValue, value, (value&PPUCTRLNMIEnable) != 0,
			uint16(0x1000)*uint16((value&PPUCTRLBGTable)>>4),
			uint16(0x1000)*uint16((value&PPUCTRLSpriteTable)>>3))
		// Immediate NMI: enabling NMI (bit 7 transition 0→1) while VBL is
		// set fires an NMI right away; pendingNMI defers one instruction
		// per nmi_control test 11 "after NEXT instruction". Re-writing $80
		// when NMI is already enabled (1→1) does NOT re-fire.
		if oldValue&PPUCTRLNMIEnable == 0 && value&PPUCTRLNMIEnable != 0 &&
			p.PPUSTATUS&PPUSTATUSVBlank != 0 {
			p.NMIRequested = true
		}
	case 0x2001: // PPUMASK
		oldValue := p.PPUMASK
		logger.LogPPU("Write PPUMASK: $%02X -> $%02X (BGShow=%v, SpriteShow=%v, Greyscale=%v)",
			oldValue, value, (value&PPUMASKBGShow) != 0, (value&PPUMASKSpriteShow) != 0, (value&PPUMASKGreyscale) != 0)
		p.PPUMASK = value
		// Render off→on transition: arm the MMC3 "first A12 rise after long
		// pause" one-shot so the next rendering scanline's cycle-5 BG fetch
		// (in BG=$1000 mode) clocks the IRQ counter once before the normal
		// per-scanline cycle-325 ticks take over. See PPU.Step for details.
		const renderShow = PPUMASKBGShow | PPUMASKSpriteShow
		if oldValue&renderShow == 0 && value&renderShow != 0 {
			p.mmc3FirstClockPending = true
		}
	case 0x2003: // OAMADDR
		p.OAMADDR = value
	case 0x2004: // OAMDATA
		p.OAM[p.OAMADDR] = value
		p.OAMADDR++
	case 0x2005: // PPUSCROLL
		logger.LogPPU("Write PPUSCROLL: value=$%02X, w=%d, scanline=%d", value, p.w, p.Scanline)
		if p.w == 0 {
			p.t = (p.t & 0xFFE0) | (uint16(value) >> 3)
			p.xTemp = value & 0x07 // Store in temporary register
			p.w = 1
			logger.LogPPU("PPUSCROLL X: value=$%02X, xTemp=%d, t=$%04X, scanline=%d", value, p.xTemp, p.t, p.Scanline)
		} else {
			p.t = (p.t & 0x8FFF) | ((uint16(value) & 0x07) << 12)
			p.t = (p.t & 0xFC1F) | ((uint16(value) & 0xF8) << 2)
			p.w = 0
			logger.LogPPU("PPUSCROLL Y: value=$%02X, t=$%04X, scanline=%d", value, p.t, p.Scanline)
		}
	case 0x2006: // PPUADDR
		logger.LogPPU("PPU Write $2006: value=$%02X, w=%d", value, p.w)
		if p.w == 0 {
			p.t = (p.t & 0x80FF) | ((uint16(value) & 0x3F) << 8)
			p.w = 1
			logger.LogPPU("Write PPUADDR (high): $%02X, t=$%04X", value, p.t)
			// Debug: Check if will point to CHR area
			if (p.t & 0xFF00) < 0x2000 {
				logger.LogPPU("PPUADDR high set for CHR area: $%04X", p.t)
			}
		} else {
			p.t = (p.t & 0xFF00) | uint16(value)
			p.v = p.t
			p.w = 0
			logger.LogPPU("Write PPUADDR (low): $%02X, v=$%04X", value, p.v)
			// Debug: Check if pointing to CHR area
			if p.v < 0x2000 {
				logger.LogPPU("PPUADDR set to CHR area: $%04X", p.v)
			}
			// The second $2006 write commits t into v, which can flip A12
			// (bit 12) — MMC3 IRQ counter clocks on A12 0→1.
			p.notifyCartridgeA12()
		}
	case 0x2007: // PPUDATA
		logger.LogPPU("PPU Write $2007: vramAddr=$%04X, value=$%02X", p.v, value)
		// Debug: Enhanced logging for CHR area writes
		if p.v <= 0x000F {
			logger.LogPPU("$2007 Write CHR: vramAddr=$%04X, value=$%02X", p.v, value)
		}
		p.writeVRAM(p.v, value)
		p.incrementVRAMAddress()
		p.notifyCartridgeA12()
	}
}

// notifyCartridgeA12 hands the current v register to the cartridge so MMC3
// (and any future A12-IRQ mapper) can detect rising edges from CPU-driven
// register accesses ($2006 second write, $2007 R/W increments).
// Rendering-side toggles are handled by Cartridge.Step in the PPU loop.
// We refresh PPU.MapperIRQ after the notification so the cached flag stays
// in sync with any IRQ asserted by the CPU-side rise.
func (p *PPU) notifyCartridgeA12() {
	if p.Cartridge == nil {
		return
	}
	p.Cartridge.NotifyA12(p.v, p.renderingEnabled())
	p.MapperIRQ = p.Cartridge.IsIRQPending()
}

// incrementVRAMAddress advances `v` after a $2007 read or write by either 1
// (across) or 32 (down) per the PPUCTRL increment bit.
func (p *PPU) incrementVRAMAddress() {
	if p.PPUCTRL&PPUCTRLIncrement != 0 {
		p.v += 32
	} else {
		p.v += 1
	}
}
