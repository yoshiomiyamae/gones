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

// ReadRegister reads from PPU register
func (p *PPU) ReadRegister(addr uint16) uint8 {
	switch addr {
	case 0x2002: // PPUSTATUS
		value := p.PPUSTATUS
		logger.LogPPU("Read PPUSTATUS: $%02X", value)
		p.PPUSTATUS &^= PPUSTATUSVBlank // Clear VBlank flag
		p.w = 0                         // Reset write toggle
		return value
	case 0x2004: // OAMDATA
		return p.OAM[p.OAMADDR]
	case 0x2007: // PPUDATA
		var value uint8

		if p.v >= 0x3F00 {
			// Palette reads are immediate (no buffering)
			value = p.readVRAM(p.v)
			// Update buffer with underlying nametable data
			p.readBuffer = p.readVRAM(p.v - 0x1000)
		} else {
			// Non-palette reads use buffered system
			value = p.readBuffer
			p.readBuffer = p.readVRAM(p.v)
		}

		// Debug: Log $2007 reads for CHR area
		if p.v < 0x2000 && p.v <= 0x000F {
			logger.LogPPU("$2007 Read CHR: vramAddr=$%04X, value=$%02X, buffer=$%02X", p.v, value, p.readBuffer)
		}

		p.incrementVRAMAddress()
		return value
	}
	return 0
}

// WriteRegister writes to PPU register
func (p *PPU) WriteRegister(addr uint16, value uint8) {
	switch addr {
	case 0x2000: // PPUCTRL
		oldValue := p.PPUCTRL
		p.PPUCTRL = value
		p.t = (p.t & 0xF3FF) | ((uint16(value) & 0x03) << 10)
		logger.LogPPU("Write PPUCTRL: $%02X -> $%02X (NMI=%v, BG_table=$%04X, Sprite_table=$%04X)",
			oldValue, value, (value&PPUCTRLNMIEnable) != 0,
			uint16(0x1000)*uint16((value&PPUCTRLBGTable)>>4),
			uint16(0x1000)*uint16((value&PPUCTRLSpriteTable)>>3))
	case 0x2001: // PPUMASK
		oldValue := p.PPUMASK
		logger.LogPPU("Write PPUMASK: $%02X -> $%02X (BGShow=%v, SpriteShow=%v, Greyscale=%v)",
			oldValue, value, (value&PPUMASKBGShow) != 0, (value&PPUMASKSpriteShow) != 0, (value&PPUMASKGreyscale) != 0)
		p.PPUMASK = value
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
		}
	case 0x2007: // PPUDATA
		logger.LogPPU("PPU Write $2007: vramAddr=$%04X, value=$%02X", p.v, value)
		// Debug: Enhanced logging for CHR area writes
		if p.v < 0x2000 && p.v <= 0x000F {
			logger.LogPPU("$2007 Write CHR: vramAddr=$%04X, value=$%02X", p.v, value)
		}
		p.writeVRAM(p.v, value)
		p.incrementVRAMAddress()
	}
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
