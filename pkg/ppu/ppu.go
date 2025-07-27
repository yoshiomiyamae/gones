package ppu

import (
	"github.com/yoshiomiyamaegones/pkg/logger"
	"github.com/yoshiomiyamaegones/pkg/memory"
)

// PPU represents the Picture Processing Unit
type PPU struct {
	// Registers
	PPUCTRL   uint8 // $2000
	PPUMASK   uint8 // $2001
	PPUSTATUS uint8 // $2002
	OAMADDR   uint8 // $2003
	OAMDATA   uint8 // $2004
	PPUSCROLL uint8 // $2005
	PPUADDR   uint8 // $2006
	PPUDATA   uint8 // $2007

	// Internal registers
	v     uint16 // VRAM address
	t     uint16 // Temporary VRAM address
	x     uint8  // Fine X scroll
	xTemp uint8  // Temporary fine X scroll for raster effects
	w     uint8  // Write toggle

	// Scrolling
	ScrollY uint8 // Y scroll position

	// VRAM
	VRAM [0x4000]uint8

	// OAM (Object Attribute Memory)
	OAM [256]uint8

	// Frame buffer (256x240)
	FrameBuffer [256 * 240]uint32

	// Persistent frame buffer for games with intermittent rendering
	PersistentFrameBuffer [256 * 240]uint32

	// Track if any meaningful rendering occurred this frame
	renderingOccurred bool
	lastRenderFrame   uint64

	// Timing
	Cycle         int
	Scanline      int
	Frame         uint64
	FrameComplete bool

	// NMI
	NMIRequested bool

	// Rendering
	PaletteManager *PaletteManager
	currentSprites []SpriteInfo

	// PPU read buffer for $2007 reads
	readBuffer uint8

	// Memory interface
	Memory *memory.Memory

	// Cartridge interface
	Cartridge interface {
		ReadCHR(addr uint16) uint8
		WriteCHR(addr uint16, value uint8)
		Step() // Called once per scanline for mapper IRQ
		IsIRQPending() bool
		ClearIRQ()
		GetMirroring() int
		NotifyA12(chrAddr uint16, renderingEnabled bool) // For MMC3 A12 edge detection
	}
}

// PPUCTRL flags
const (
	PPUCTRLNameTable   = 0x03 // Base nametable address
	PPUCTRLIncrement   = 0x04 // VRAM address increment
	PPUCTRLSpriteTable = 0x08 // Sprite pattern table address
	PPUCTRLBGTable     = 0x10 // Background pattern table address
	PPUCTRLSpriteSize  = 0x20 // Sprite size
	PPUCTRLMasterSlave = 0x40 // PPU master/slave select
	PPUCTRLNMIEnable   = 0x80 // Generate NMI at VBlank
)

// PPUMASK flags
const (
	PPUMASKGreyscale      = 0x01 // Greyscale
	PPUMASKBGLeft         = 0x02 // Show background in leftmost 8 pixels
	PPUMASKSpriteLeft     = 0x04 // Show sprites in leftmost 8 pixels
	PPUMASKBGShow         = 0x08 // Show background
	PPUMASKSpriteShow     = 0x10 // Show sprites
	PPUMASKRedEmphasize   = 0x20 // Emphasize red
	PPUMASKGreenEmphasize = 0x40 // Emphasize green
	PPUMASKBlueEmphasize  = 0x80 // Emphasize blue
)

// PPUSTATUS flags
const (
	PPUSTATUSSprite0Hit = 0x40 // Sprite 0 hit
	PPUSTATUSVBlank     = 0x80 // VBlank flag
)

// New creates a new PPU instance
func New(mem *memory.Memory) *PPU {
	return &PPU{
		Memory:         mem,
		Cycle:          0,
		Scanline:       0,
		PaletteManager: NewPaletteManager(),
	}
}

// Reset resets the PPU to initial state
func (p *PPU) Reset() {
	p.PPUCTRL = 0
	p.PPUMASK = 0
	p.PPUSTATUS = 0
	p.OAMADDR = 0
	p.v = 0
	p.t = 0
	p.x = 0
	p.w = 0
	p.Cycle = 0
	p.Scanline = 0
	p.FrameComplete = false

	// Initialize persistent buffer with background color to indicate "no content yet"
	// Don't reset persistent buffer on Reset to preserve accumulated content
	p.renderingOccurred = false
}

// SetCartridge sets the cartridge reference
func (p *PPU) SetCartridge(cart interface {
	ReadCHR(addr uint16) uint8
	WriteCHR(addr uint16, value uint8)
	Step()
	IsIRQPending() bool
	ClearIRQ()
	GetMirroring() int
	NotifyA12(chrAddr uint16, renderingEnabled bool)
}) {
	p.Cartridge = cart
}

// Step executes one PPU cycle
func (p *PPU) Step() {
	// Update emphasis for palette manager
	p.PaletteManager.SetEmphasis(p.PPUMASK & 0xE0)

	// Render visible scanlines and handle MMC3 A12 timing
	if p.Scanline >= 0 && p.Scanline < 240 {
		p.renderPixel()
		// Trigger MMC3 A12 detection at specific cycles for accurate timing
		p.handleMMC3A12Timing()
	}

	p.Cycle++
	if p.Cycle >= 341 {
		p.Cycle = 0

		p.Scanline++

		// MMC3 IRQ timing - call mapper step for scanline-based IRQ timing
		// This works even when rendering is disabled, allowing games to set up IRQs
		if p.Cartridge != nil && p.Scanline >= 0 && p.Scanline < 240 {
			p.Cartridge.Step()
		}

		if p.Scanline == 241 {
			// VBlank start - clear sprite 0 hit flag
			p.PPUSTATUS |= PPUSTATUSVBlank
			p.PPUSTATUS &^= PPUSTATUSSprite0Hit // Clear sprite 0 hit flag at VBlank start
			// Reduced logging for performance
			if p.PPUCTRL&PPUCTRLNMIEnable != 0 {
				p.NMIRequested = true
			}
		}

		if p.Scanline >= 261 {
			p.Scanline = -1 // Pre-render scanline
			p.FrameComplete = true

			// Handle frame completion and persistent buffer management
			p.handleFrameCompletion()

			p.Frame++

			// Clear VBlank flag at end of frame (start of new frame)
			p.PPUSTATUS &^= PPUSTATUSVBlank
			// Reduced frame completion logging for performance
		}
	}

	// Handle pre-render scanline (scanline -1/261)
	if p.Scanline == -1 {
		// Copy horizontal scroll components from t to v at start of pre-render line
		if p.Cycle == 304 && (p.PPUMASK&(PPUMASKBGShow|PPUMASKSpriteShow)) != 0 {
			// Copy vertical scroll components from t to v
			p.v = (p.v & 0x841F) | (p.t & 0x7BE0)
			// logger.LogPPU("Pre-render: Copy vertical scroll t=$%04X to v=$%04X", p.t, p.v)
		}
		if p.Cycle == 257 && (p.PPUMASK&(PPUMASKBGShow|PPUMASKSpriteShow)) != 0 {
			// Copy horizontal scroll components from t to v
			p.v = (p.v & 0xFBE0) | (p.t & 0x041F)
			// logger.LogPPU("Pre-render: Copy horizontal scroll t=$%04X to v=$%04X", p.t, p.v)
		}
	}

	// Handle visible scanlines
	if p.Scanline >= 0 && p.Scanline < 240 {
		// Copy horizontal scroll components from t to v at start of next scanline
		if p.Cycle == 0 && (p.PPUMASK&(PPUMASKBGShow|PPUMASKSpriteShow)) != 0 {
			p.v = (p.v & 0xFBE0) | (p.t & 0x041F)
			p.x = p.xTemp // Apply fine X scroll from temporary register
			// logger.LogPPU("Scanline %d: Copy scroll t=$%04X to v=$%04X, x=%d", p.Scanline, p.t, p.v, p.x)
		}
	}
}

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

		if p.PPUCTRL&PPUCTRLIncrement != 0 {
			p.v += 32
		} else {
			p.v += 1
		}
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
		if p.PPUCTRL&PPUCTRLIncrement != 0 {
			p.v += 32
		} else {
			p.v += 1
		}
	}
}

// readVRAM reads from VRAM
func (p *PPU) readVRAM(addr uint16) uint8 {
	addr = addr % 0x4000

	if addr < 0x2000 {
		// Pattern table
		if p.Cartridge != nil {
			// Notify cartridge of A12 changes for MMC3 IRQ timing
			// Only during visible scanlines and rendering enabled
			renderingEnabled := (p.PPUMASK & (PPUMASKBGShow | PPUMASKSpriteShow)) != 0
			isVisibleScanline := p.Scanline >= 0 && p.Scanline < 240
			if renderingEnabled && isVisibleScanline {
				p.Cartridge.NotifyA12(addr, renderingEnabled)
			}

			value := p.Cartridge.ReadCHR(addr)
			// Debug: Log CHR reads via PPU - focus on pattern table reads with scanline info
			if addr <= 0x1FFF && (addr < 0x100 || (addr >= 0x800 && addr < 0x900)) {
				// Log first 256 bytes of each bank for key areas
				logger.LogPPU("PPU CHR Read: scanline=%d, cycle=%d, addr=$%04X, value=$%02X, table=%s",
					p.Scanline, p.Cycle, addr, value,
					func() string {
						if addr < 0x1000 {
							return "BG"
						} else {
							return "SPR"
						}
					}())
			}
			return value
		}
		logger.LogPPU("ReadCHR: no cartridge, returning 0")
		return 0
	} else if addr < 0x3F00 {
		// Nametable with mirroring
		return p.readNameTable(addr)
	} else if addr < 0x4000 {
		// Palette
		return p.PaletteManager.ReadPalette(uint8(addr & 0x1F))
	}

	return 0
}

// writeVRAM writes to VRAM
func (p *PPU) writeVRAM(addr uint16, value uint8) {
	addr = addr % 0x4000

	if addr < 0x2000 {
		// Pattern table (CHR)
		if p.Cartridge != nil {
			// Notify cartridge of A12 changes for MMC3 IRQ timing
			// Only during visible scanlines and rendering enabled
			renderingEnabled := (p.PPUMASK & (PPUMASKBGShow | PPUMASKSpriteShow)) != 0
			isVisibleScanline := p.Scanline >= 0 && p.Scanline < 240
			if renderingEnabled && isVisibleScanline {
				p.Cartridge.NotifyA12(addr, renderingEnabled)
			}

			// Debug: Log CHR writes via PPU for first bytes
			if addr <= 0x000F {
				logger.LogPPU("PPU CHR Write: addr=$%04X, value=$%02X", addr, value)
			}
			p.Cartridge.WriteCHR(addr, value)
		}
	} else if addr < 0x3F00 {
		// Nametable with mirroring
		p.writeNameTable(addr, value)
	} else if addr < 0x4000 {
		// Palette
		paletteAddr := uint8(addr & 0x1F)
		p.PaletteManager.WritePalette(paletteAddr, value)
	}
}

// GetFramebuffer returns the current framebuffer as RGBA bytes
func (p *PPU) GetFramebuffer() []uint8 {
	// Convert 32-bit framebuffer to RGBA bytes
	rgba := make([]uint8, 256*240*4)

	for i, pixel := range p.FrameBuffer {
		// Extract RGB components from 32-bit pixel (0xAARRGGBB format)
		r := uint8((pixel >> 16) & 0xFF) // Extract R correctly
		g := uint8((pixel >> 8) & 0xFF)  // Extract G correctly
		b := uint8(pixel & 0xFF)         // Extract B correctly
		a := uint8((pixel >> 24) & 0xFF) // Use alpha from pixel

		// Use RGBA order to match test pattern format
		rgba[i*4+0] = r
		rgba[i*4+1] = g
		rgba[i*4+2] = b
		rgba[i*4+3] = a

		// Debug logging for first few pixels (disabled for performance)
		// if i < 8 {
		//	logger.LogPPU("Framebuffer[%d]: pixel=%08X -> RGBA(%02X,%02X,%02X,%02X)",
		//		i, pixel, r, g, b, a)
		// }
	}

	return rgba
}

// readNameTable reads from nametable with mirroring
func (p *PPU) readNameTable(addr uint16) uint8 {
	// Mirror the address based on cartridge mirroring mode
	mirroredAddr := p.mirrorNameTableAddress(addr)
	return p.VRAM[mirroredAddr]
}

// writeNameTable writes to nametable with mirroring
func (p *PPU) writeNameTable(addr uint16, value uint8) {
	// Mirror the address based on cartridge mirroring mode
	mirroredAddr := p.mirrorNameTableAddress(addr)
	p.VRAM[mirroredAddr] = value
}

// mirrorNameTableAddress applies nametable mirroring
func (p *PPU) mirrorNameTableAddress(addr uint16) uint16 {
	// Nametable addresses are $2000-$2FFF (4KB range)
	// Remove the base offset to get 0-$FFF range
	offset := addr - 0x2000

	if p.Cartridge == nil {
		// Default to horizontal mirroring if no cartridge
		return p.applyHorizontalMirroring(offset) + 0x2000
	}

	switch p.Cartridge.GetMirroring() {
	case 0: // Horizontal mirroring
		return p.applyHorizontalMirroring(offset) + 0x2000
	case 1: // Vertical mirroring
		return p.applyVerticalMirroring(offset) + 0x2000
	default:
		// Four-screen or other modes - no mirroring
		return addr
	}
}

// applyHorizontalMirroring applies horizontal mirroring
func (p *PPU) applyHorizontalMirroring(offset uint16) uint16 {
	// Horizontal mirroring: $2000=$2400, $2800=$2C00
	if offset >= 0x800 {
		return offset - 0x400 // Map $2800-$2FFF to $2400-$27FF
	}
	return offset & 0x7FF // Map $2000-$27FF to $2000-$27FF
}

// applyVerticalMirroring applies vertical mirroring
func (p *PPU) applyVerticalMirroring(offset uint16) uint16 {
	// Vertical mirroring: $2000=$2800, $2400=$2C00
	return offset & 0x7FF // Map $2000-$2FFF to $2000-$27FF
}

// IsMapperIRQPending returns whether mapper IRQ is pending
func (p *PPU) IsMapperIRQPending() bool {
	if p.Cartridge != nil {
		return p.Cartridge.IsIRQPending()
	}
	return false
}

// ClearMapperIRQ clears mapper IRQ
func (p *PPU) ClearMapperIRQ() {
	if p.Cartridge != nil {
		p.Cartridge.ClearIRQ()
	}
}

// handleFrameCompletion manages persistent frame buffer and rendering state
func (p *PPU) handleFrameCompletion() {
	// Debug: Check first few pixels of FrameBuffer before completion handling
	nonZeroPixels := 0
	for i := 0; i < 256; i++ {
		if p.FrameBuffer[i] != 0 {
			nonZeroPixels++
		}
	}

	// Store the rendering occurred flag before resetting
	hadRendering := p.renderingOccurred

	// Reset rendering flag for next frame FIRST
	p.renderingOccurred = false

	// If rendering occurred this frame, update the last render frame
	if hadRendering {
		p.lastRenderFrame = p.Frame
		logger.LogPPU("Frame %d: Rendering occurred, updating persistent buffer", p.Frame)

		// Ensure FrameBuffer has the rendered content for display
		// (FrameBuffer should already have the content from renderPixel calls)
	} else {
		// Keep previous frame content to prevent flickering
		// Don't copy persistent buffer unnecessarily
	}
}

// GetDisplayFrameBuffer returns the frame buffer that should be displayed
// This method provides the correct buffer considering persistent rendering
func (p *PPU) GetDisplayFrameBuffer() []uint32 {
	// If recent rendering occurred, return current buffer
	frameSinceLastRender := p.Frame - p.lastRenderFrame

	// Debug logging disabled for production

	if frameSinceLastRender <= 1 || p.renderingOccurred {
		return p.FrameBuffer[:]
	}

	// Otherwise, return persistent buffer if it has content
	if frameSinceLastRender < 3600 { // Keep visible for ~1 minute (3600 frames)
		// Check if persistent buffer has meaningful content
		nonZeroCount := 0
		for i := 0; i < 100; i++ { // Sample first 100 pixels
			if p.PersistentFrameBuffer[i] != 0 {
				nonZeroCount++
			}
		}

		// Debug logging disabled for production

		return p.PersistentFrameBuffer[:]
	}

	// Fall back to current buffer
	return p.FrameBuffer[:]
}

// handleMMC3A12Timing handles cycle-accurate MMC3 A12 detection
func (p *PPU) handleMMC3A12Timing() {
	if p.Cartridge == nil {
		return
	}

	renderingEnabled := (p.PPUMASK & (PPUMASKBGShow | PPUMASKSpriteShow)) != 0
	if !renderingEnabled {
		return
	}

	// MMC3 A12 detection based on PPU tile fetching patterns
	// Background tiles: dots 0-255, 320-340 (A12 depends on BG table)
	// Sprite patterns: dots 256-319 (A12 depends on sprite table)

	var a12Addr uint16
	var shouldNotify bool = false

	// Determine which pattern table is being accessed based on cycle
	if (p.Cycle >= 0 && p.Cycle <= 255) || (p.Cycle >= 320 && p.Cycle <= 340) {
		// Background tile fetch cycles - use background pattern table
		bgTableSelect := (p.PPUCTRL & PPUCTRLBGTable) >> 4
		if bgTableSelect == 0 {
			a12Addr = 0x0000 // A12 = 0
		} else {
			a12Addr = 0x1000 // A12 = 1
		}
		shouldNotify = true
	} else if p.Cycle >= 256 && p.Cycle <= 319 {
		// Sprite pattern fetch cycles - use sprite pattern table
		spriteTableSelect := (p.PPUCTRL & PPUCTRLSpriteTable) >> 3
		if spriteTableSelect == 0 {
			a12Addr = 0x0000 // A12 = 0
		} else {
			a12Addr = 0x1000 // A12 = 1
		}
		shouldNotify = true
	}

	// Notify cartridge of A12 state for cycle-accurate timing
	// Ultra-precise notification at key tile fetch cycles
	if shouldNotify {
		// Notify at precise tile fetch boundaries for maximum accuracy
		isTileFetchCycle := (p.Cycle%8 == 0) || (p.Cycle%8 == 2) || (p.Cycle%8 == 4) || (p.Cycle%8 == 6)
		if isTileFetchCycle {
			p.Cartridge.NotifyA12(a12Addr, renderingEnabled)
		}
	}
}
