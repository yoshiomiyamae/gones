package ppu

import (
	"encoding/binary"
	"io"

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

	// Timing
	Cycle         int
	Scanline      int
	Frame         uint64
	FrameComplete bool

	// NMI
	NMIRequested bool

	// MapperIRQ mirrors the cartridge's IRQ-pending line. nes.Step reads
	// this every PPU cycle to set the CPU IRQ flag, so it MUST stay cheaper
	// than an interface dispatch through Cartridge — every code path that
	// can change mapper IRQ state (Step at the per-scanline tick,
	// notifyCartridgeA12, and a post-CPU.Step refresh in nes.Step that
	// catches $E000 acks) updates this field.
	MapperIRQ bool

	// mmc3FirstClockPending arms a one-shot extra A12-rise clock on the
	// next rendering scanline (consumed by the BG=$1000 path in Step).
	// Set on PPUMASK render-off→on; cleared after one $1000-mode tick.
	mmc3FirstClockPending bool

	// vblSuppressed records a $2002 read that landed in the race window
	// where the VBL flag is about to be set. On real hardware a read
	// straddling the set cycle suppresses both the flag set and the NMI
	// for that frame. Our PPU lags the real PPU by ~2 cycles at memory-
	// access time (the read happens before the post-instruction catch-up),
	// so the race window in our model lands at (scanline 240, cycle 340)
	// — the last cycle before our (241, 0) flag-set transition. Set by
	// ReadRegister, consumed by Step at the transition.
	vblSuppressed bool

	// nmiAssertCountdown defers the NMI-line assertion by
	// nmiAssertDelayPPUCycles cycles after the VBL flag is set. The flag
	// itself becomes visible to a CPU $2002 read immediately (so blargg
	// vbl_set_time test 5+ passes); the NMI assertion lags so nmi_timing's
	// transitions land at the right calibration row. 0 = inactive;
	// counts down per Step, asserts NMIRequested on hitting 0.
	nmiAssertCountdown uint8

	// oddFrame flips at the end of every pre-render scanline. NTSC PPU
	// "skips" one cycle (cycle 340 of pre-render) on odd frames when BG
	// rendering is enabled — blargg even_odd_frames / even_odd_timing
	// rely on this to keep their hit-counter calibration in sync.
	oddFrame bool

	// openBus models the PPU's CPU-side data-bus "decay register". Per-bit
	// refresh because $2002 reads only refresh bits 5-7 (test 7 verifies
	// bits 0-4 still decay independently) and $2007 palette reads only
	// refresh bits 0-5 (test 9 verifies bits 6-7 still decay).
	openBusValue      uint8
	openBusDecayFrame [8]uint64

	// cachedMirroring snapshots Cartridge.GetMirroring() at scanline start.
	// mirrorNameTableAddress reads this every nametable access — looking it
	// up via the cartridge interface every time was a measurable cost. MMC1
	// / MMC3 / MMC4 can change mirroring at runtime, but only via CPU writes
	// to mapper registers — refreshing once per scanline is fine-grained
	// enough for every commercial game.
	cachedMirroring int

	// currentBGTile holds the most-recently-fetched background tile for
	// the visible scanline; currentBGTileX is its index (-1 = invalid).
	// bgTileAt re-fetches when crossing an 8-pixel tile
	// boundary, so a mid-scanline $2000 D4 toggle (pattern table) or a
	// $2005/$2006 update affects subsequent tiles within the same
	// scanline — required by Quietust's scanline test. v.coarseX / fineX
	// are still constant for the scanline in this PPU model, so the
	// cache invalidates only on tileX change (≈33 fetches per scanline,
	// same volume as the old whole-scanline prefetch).
	currentBGTile  BackgroundTile
	currentBGTileX int

	// Rendering. currentSprites holds the sprites overlapping the current
	// scanline (max 8), evaluated once at cycle 0; currentSpriteCount is how
	// many are valid. Each entry carries its pre-fetched pattern row bytes so
	// per-pixel sprite rendering needs no CHR fetch. Fixed-size array to avoid
	// a per-scanline heap allocation.
	PaletteManager     *PaletteManager
	currentSprites     [8]SpriteInfo
	currentSpriteCount int

	// PPU read buffer for $2007 reads
	readBuffer uint8

	// Memory interface
	Memory *memory.Memory

	// Cartridge interface
	Cartridge interface {
		ReadCHR(addr uint16) uint8
		ReadCHRSprite(addr uint16) uint8 // sprite-side fetch — MMC5 8×16 uses a different CHR set
		WriteCHR(addr uint16, value uint8)
		Step() // Called once per scanline for mapper IRQ
		IsIRQPending() bool
		ClearIRQ()
		GetMirroring() int
		NotifyA12(chrAddr uint16, renderingEnabled bool) // For MMC3 A12 edge detection
		SetSpriteSize(is8x16 bool)                       // MMC5 tracks this for CHR routing
		NotifyScanline(scanline int, renderingEnabled bool)
		HasExpansion() bool // MMC5 — also the only mapper that remaps nametables mid-scanline
	}

	// dynamicMirroring is true when the mapper can change its nametable
	// mapping mid-scanline (MMC5's $5105). Only then does the per-NT-read
	// mirror refresh in readNameTable/writeVRAM earn its cost; every other
	// mapper changes mirroring via CPU writes the scanline-boundary refresh
	// already catches. Cached from Cartridge.HasExpansion() at SetCartridge.
	dynamicMirroring bool

	// Large arrays last so the small, per-pixel-hot scalar fields above
	// cluster into a few cache lines instead of being pushed hundreds of KB
	// apart by these buffers (which would alias the FrameBuffer write stream
	// against the control fields and cost ~2% on full-screen redraws).
	VRAM        [0x4000]uint8     // pattern/nametable/palette space
	OAM         [256]uint8        // sprite attribute memory
	FrameBuffer [256 * 240]uint32 // 256x240 ARGB output
}

// NES screen dimensions in pixels (NTSC visible area).
const (
	ScreenWidth  = 256
	ScreenHeight = 240
)

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
	PPUSTATUSSpriteOverflow = 0x20 // 9+ sprites on a scanline (eval-phase latched)
	PPUSTATUSSprite0Hit     = 0x40 // Sprite 0 hit
	PPUSTATUSVBlank         = 0x80 // VBlank flag
)

// Mirroring mode codes returned by Cartridge.GetMirroring() and mapper
// GetMirroringMode() implementations. Single-screen modes are used by MMC1.
const (
	MirroringHorizontal        = 0
	MirroringVertical          = 1
	MirroringSingleScreenLower = 2
	MirroringSingleScreenUpper = 3
)

// New creates a new PPU instance
func New(mem *memory.Memory) *PPU {
	return &PPU{
		Memory:         mem,
		Cycle:          0,
		Scanline:       0,
		PaletteManager: NewPaletteManager(),
		currentBGTileX: -1,
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
	p.currentBGTileX = -1
	// PPUMASK was just cleared; keep the palette emphasis in sync with it
	// (emphasis is only updated on $2001 writes, not derived per-cycle).
	if p.PaletteManager != nil {
		p.PaletteManager.SetEmphasis(0)
	}
}

// SetCartridge sets the cartridge reference
func (p *PPU) SetCartridge(cart interface {
	ReadCHR(addr uint16) uint8
	ReadCHRSprite(addr uint16) uint8
	WriteCHR(addr uint16, value uint8)
	Step()
	IsIRQPending() bool
	ClearIRQ()
	GetMirroring() int
	NotifyA12(chrAddr uint16, renderingEnabled bool)
	SetSpriteSize(is8x16 bool)
	NotifyScanline(scanline int, renderingEnabled bool)
	HasExpansion() bool
}) {
	p.Cartridge = cart
	p.dynamicMirroring = cart.HasExpansion()
	p.refreshMirroringCache()
}

// Step executes one PPU cycle
func (p *PPU) Step() {
	// NMI-assertion countdown — see PPU.nmiAssertCountdown. Tick down
	// here so the assertion lands N PPU cycles after the VBL set Step.
	// Re-check VBL at expiry: if a CPU $2002 read cleared the flag during
	// the countdown window, NMI is suppressed (blargg suppression test
	// rows 05-06: flag read back as set but NMI never fires because the
	// quickly-cleared flag never held the NMI line low long enough).
	if p.nmiAssertCountdown > 0 {
		p.nmiAssertCountdown--
		if p.nmiAssertCountdown == 0 && p.PPUCTRL&PPUCTRLNMIEnable != 0 && p.PPUSTATUS&PPUSTATUSVBlank != 0 {
			p.NMIRequested = true
		}
	}

	// Render visible scanlines
	if p.Scanline >= 0 && p.Scanline < 240 {
		p.renderPixel()
	}

	// MMC3 IRQ + per-scanline Y increment both need: rendering enabled, and we're
	// on a visible scanline (or pre-render). The Y increment makes mid-frame $2006
	// writes propagate for split-screen effects (e.g. SMB3 title screen) — the
	// renderer reads v.coarseY/fineY as the *current* scanline's row, not a
	// frame-start scroll.
	renderingActive := (p.Scanline >= 0 && p.Scanline < 240 || p.Scanline == -1) && p.renderingEnabled()
	if renderingActive && p.Cartridge != nil {
		// MMC3 IRQ clocking — A12 rising edges with a ~3-CPU-cycle low
		// filter. We pick the PPU cycle of the counted rise from PPUCTRL:
		//   BG=$0000, Sprites=$1000: first sprite-pattern fetch (~261);
		//     empirically cycle 273 matches blargg scanline_timing.
		//   BG=$1000, Sprites=$0000: prefetch BG-pattern fetch after the
		//     64-cycle sprite-fetch low window (~cycle 325, emu cycle 337).
		// Plus a one-shot extra clock on the first rendering scanline
		// after PPUMASK 0→on (the render-off period satisfies the filter
		// for the first BG-pattern fetch at cycle ~5 / emu cycle 17;
		// subsequent cycle-5 rises are filtered out by the short
		// inter-scanline low gap).
		bg1000 := p.PPUCTRL&PPUCTRLBGTable != 0
		tickPerScanline := 273
		if bg1000 {
			tickPerScanline = 337
		}
		clockMapper := p.Cycle == tickPerScanline
		if !clockMapper && p.mmc3FirstClockPending && bg1000 && p.Cycle == 17 {
			clockMapper = true
			p.mmc3FirstClockPending = false
		}
		if clockMapper {
			p.Cartridge.Step()
			p.MapperIRQ = p.Cartridge.IsIRQPending()
		}
	}
	if renderingActive && p.Cycle == 256 {
		p.incrementY()
	}

	p.Cycle++
	if p.Cycle >= 341 {
		p.Cycle = 0
		p.refreshMirroringCache()

		p.Scanline++
		// Odd-frame skip: NTSC PPU drops the first idle tick (cycle 0) of
		// scanline 0 on odd frames when background rendering is enabled.
		// Realised here by starting the new visible scanline at cycle 1
		// instead of 0 under those conditions.
		if p.Scanline == 0 && p.oddFrame && p.PPUMASK&PPUMASKBGShow != 0 {
			p.Cycle = 1
		}
		// Tell the mapper about the new rendering scanline. MMC5 uses
		// this for its scanline-match IRQ; other mappers ignore it.
		if p.Cartridge != nil && p.Scanline >= 0 && p.Scanline < 240 {
			p.Cartridge.NotifyScanline(int(p.Scanline), p.renderingEnabled())
		}

		if p.Scanline == 241 {
			// VBlank start: set VBlank flag immediately so a CPU $2002
			// read here observes it (vbl_set_time T+5). The NMI assertion
			// is deferred by nmiAssertDelayPPUCycles so nmi_timing's
			// calibration table lands on the right CPU instruction.
			if !p.vblSuppressed {
				p.PPUSTATUS |= PPUSTATUSVBlank
				if p.PPUCTRL&PPUCTRLNMIEnable != 0 {
					p.nmiAssertCountdown = nmiAssertDelayPPUCycles
				}
			}
			p.vblSuppressed = false
		}

		if p.Scanline >= 261 {
			p.Scanline = -1 // Pre-render scanline

			// Pre-render line: clear VBlank, sprite 0 hit, and sprite overflow
			// flags. NESdev says this is at cycle 1 of pre-render; doing it
			// here at the (260, 340) → (-1, 0) wrap places it one PPU cycle
			// earlier in absolute terms. Tests vbl_clear_time / suppression
			// pass with this earlier timing — moving the clear to (-1, 1)
			// breaks test 3's row-06 expectation.
			p.PPUSTATUS &^= PPUSTATUSVBlank
			p.PPUSTATUS &^= PPUSTATUSSprite0Hit
			p.PPUSTATUS &^= PPUSTATUSSpriteOverflow

			p.FrameComplete = true
			p.Frame++
			p.oddFrame = !p.oddFrame
		}
	}

	// Handle pre-render scanline (scanline -1/261)
	if p.Scanline == -1 {
		// Copy horizontal scroll components from t to v at start of pre-render line
		if p.Cycle == 304 && p.renderingEnabled() {
			// Copy vertical scroll components from t to v
			p.v = (p.v & 0x841F) | (p.t & 0x7BE0)
		}
		if p.Cycle == 257 && p.renderingEnabled() {
			// Copy horizontal scroll components from t to v
			p.v = (p.v & 0xFBE0) | (p.t & 0x041F)
		}
	}

	// Copy horizontal scroll from t to v at the start of each visible
	// scanline and invalidate the single-tile cache so the first fetch
	// uses the just-restored v / x.
	if p.Scanline >= 0 && p.Scanline < 240 && p.Cycle == 0 && p.renderingEnabled() {
		p.v = (p.v & 0xFBE0) | (p.t & 0x041F)
		p.x = p.xTemp
		p.currentBGTileX = -1
	}
}

// incrementY advances v's vertical position by one scanline per the NESdev
// PPU rendering spec: fine Y first, with coarse Y / NT_Y wrap on overflow.
func (p *PPU) incrementY() {
	if (p.v & 0x7000) != 0x7000 {
		p.v += 0x1000
		return
	}
	p.v &= 0x8FFF // fine Y = 0
	y := (p.v >> 5) & 0x1F
	switch y {
	case 29:
		y = 0
		p.v ^= 0x0800 // flip vertical nametable
	case 31:
		y = 0
	default:
		y++
	}
	p.v = (p.v &^ 0x03E0) | (y << 5)
}

// readVRAM reads from VRAM
func (p *PPU) readVRAM(addr uint16) uint8 {
	return p.readVRAMInternal(addr, false)
}

// readVRAMSprite is like readVRAM but tells the cartridge that the
// pattern fetch is for a sprite. MMC5 uses this to route through its
// 'A' (sprite) CHR set in 8×16 mode; other mappers ignore it.
func (p *PPU) readVRAMSprite(addr uint16) uint8 {
	return p.readVRAMInternal(addr, true)
}

func (p *PPU) readVRAMInternal(addr uint16, sprite bool) uint8 {
	addr = addr % 0x4000

	if addr < 0x2000 {
		// Pattern table
		if p.Cartridge != nil {
			var value uint8
			if sprite {
				value = p.Cartridge.ReadCHRSprite(addr)
			} else {
				value = p.Cartridge.ReadCHR(addr)
			}
			// Debug: Log CHR reads via PPU - focus on pattern table reads with scanline info
			if logger.PPUEnabled() && addr <= 0x1FFF && (addr < 0x100 || (addr >= 0x800 && addr < 0x900)) {
				// Log first 256 bytes of each bank for key areas
				table := "BG"
				if addr >= 0x1000 {
					table = "SPR"
				}
				logger.LogPPU("PPU CHR Read: scanline=%d, cycle=%d, addr=$%04X, value=$%02X, table=%s",
					p.Scanline, p.Cycle, addr, value, table)
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

	// Same MMC5 mid-rendering concern as readNameTable: $5105 can flip
	// mid-frame and we cache mirroring at scanline start. Refresh so a
	// $2007 write that lands inside the IRQ handler routes to the
	// physical NT the game intends.
	if p.dynamicMirroring && addr >= 0x2000 && addr < 0x3F00 {
		p.refreshMirroringCache()
	}

	if addr < 0x2000 {
		// Pattern table (CHR)
		if p.Cartridge != nil {
			// Debug: Log CHR writes via PPU for first bytes
			if logger.PPUEnabled() && addr <= 0x000F {
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
	// MMC5 can change its $5105 NT-mapping mid-rendering (Metal Slader
	// Glory flips it $00 ↔ $55 around scanline 162 to switch between the
	// top-half image NT and the dialog-box NT). The per-scanline mirror
	// cache misses those transitions; refresh here so the fetch sees the
	// live mapping. Gated to dynamic-mirroring mappers so the ~16k
	// per-frame interface dispatches don't burden every other game.
	if p.dynamicMirroring {
		p.refreshMirroringCache()
	}
	mirroredAddr := p.mirrorNameTableAddress(addr)
	return p.VRAM[mirroredAddr]
}

// writeNameTable writes to nametable with mirroring
func (p *PPU) writeNameTable(addr uint16, value uint8) {
	// Mirror the address based on cartridge mirroring mode
	mirroredAddr := p.mirrorNameTableAddress(addr)
	p.VRAM[mirroredAddr] = value
}

// mirrorNameTableAddress applies nametable mirroring using the cached
// mirroring mode (refreshed each scanline by refreshMirroringCache).
func (p *PPU) mirrorNameTableAddress(addr uint16) uint16 {
	offset := addr - 0x2000

	switch p.cachedMirroring {
	case MirroringHorizontal:
		return p.applyHorizontalMirroring(offset) + 0x2000
	case MirroringVertical:
		return p.applyVerticalMirroring(offset) + 0x2000
	case MirroringSingleScreenLower:
		return (offset & 0x3FF) + 0x2000
	case MirroringSingleScreenUpper:
		return (offset & 0x3FF) + 0x2400
	default:
		// Four-screen — no mirroring, use logical address as-is.
		return addr
	}
}

// refreshMirroringCache reloads cachedMirroring from the cartridge. Called
// at scanline boundaries; MMC1/MMC3/MMC4 change mirroring via CPU register
// writes whose effect doesn't need to land mid-scanline.
func (p *PPU) refreshMirroringCache() {
	if p.Cartridge != nil {
		p.cachedMirroring = p.Cartridge.GetMirroring()
	} else {
		p.cachedMirroring = MirroringHorizontal
	}
}

// applyHorizontalMirroring applies horizontal mirroring.
// $2000 and $2400 share physical NT_A; $2800 and $2C00 share physical NT_B.
// Bit 11 (0x800) selects which physical nametable; bit 10 (0x400) is ignored.
func (p *PPU) applyHorizontalMirroring(offset uint16) uint16 {
	return ((offset & 0x800) >> 1) | (offset & 0x3FF)
}

// applyVerticalMirroring applies vertical mirroring.
// $2000 and $2800 share physical NT_A; $2400 and $2C00 share physical NT_B.
// Bit 10 (0x400) selects which physical nametable; bit 11 (0x800) is ignored.
func (p *PPU) applyVerticalMirroring(offset uint16) uint16 {
	return offset & 0x7FF
}

// IsMapperIRQPending returns the cached mapper IRQ line state. Kept in
// sync with the cartridge by Step (per-scanline tick), notifyCartridgeA12
// ($2006/$2007-driven CPU rises), and a post-CPU.Step refresh in nes.Step
// (catches $E000 acks). nes.Step polls this every PPU cycle, so the field
// read must stay cheaper than an interface dispatch.
func (p *PPU) IsMapperIRQPending() bool {
	return p.MapperIRQ
}

// ClearMapperIRQ forwards to the cartridge so the mapper's internal pending
// bit drops. Retained for explicit-clear callers; the regular IRQ servicing
// path doesn't call this (it relies on the game writing $E000 to ack).
func (p *PPU) ClearMapperIRQ() {
	p.MapperIRQ = false
	if p.Cartridge != nil {
		p.Cartridge.ClearIRQ()
	}
}

// renderingEnabled reports whether either background or sprite rendering is
// turned on via PPUMASK. Read on every visible PPU cycle, so it's a single
// AND/compare — inlined by the compiler.
func (p *PPU) renderingEnabled() bool {
	return p.PPUMASK&(PPUMASKBGShow|PPUMASKSpriteShow) != 0
}

// ConsumeNMI returns and clears the PPU's pending NMI assertion flag.
// nes.Step uses this to route NMIs through its delivery pipeline without
// touching the field directly.
func (p *PPU) ConsumeNMI() bool {
	if !p.NMIRequested {
		return false
	}
	p.NMIRequested = false
	return true
}

// openBusDecayFrames is the window after which an un-refreshed bit reads
// as 0. Spec says ~600ms; 30 frames sits comfortably between blargg's
// rapid-poll tests (must NOT have decayed at <1000ms with no refresh) and
// the slow-decay test (must have decayed by 1000ms).
const openBusDecayFrames = 30

// nmiAssertDelayPPUCycles is the PPU-cycle gap between the VBL flag set
// and the NMI-line assertion that drives NMIRequested. Tuned against
// blargg nmi_timing's calibration table — anything other than 2 shifts
// the transition rows by 1 each cycle of change.
const nmiAssertDelayPPUCycles = 2

// refreshOpenBus updates bits in `mask` to take their values from `value`
// and resets their decay timers to the current frame. Hot path — the
// mask=0xFF case (every WriteRegister, $2004 read, $2007 non-palette read)
// gets a vectorisable array assignment instead of an 8-iter masked loop.
func (p *PPU) refreshOpenBus(value, mask uint8) {
	p.openBusValue = (p.openBusValue &^ mask) | (value & mask)
	f := p.Frame
	if mask == 0xFF {
		p.openBusDecayFrame = [8]uint64{f, f, f, f, f, f, f, f}
		return
	}
	for i := uint8(0); i < 8; i++ {
		if mask&(1<<i) != 0 {
			p.openBusDecayFrame[i] = f
		}
	}
}

// readOpenBus returns the decay-register value with each bit zeroed if it
// hasn't been refreshed within the decay window.
func (p *PPU) readOpenBus() uint8 {
	var result uint8
	f := p.Frame
	for i := uint8(0); i < 8; i++ {
		if f-p.openBusDecayFrame[i] < openBusDecayFrames {
			result |= p.openBusValue & (1 << i)
		}
	}
	return result
}

// ppuState is the on-disk layout for PPU state. Frame buffers are excluded;
// they get redrawn from VRAM/OAM/palette by the next scanline anyway.
type ppuState struct {
	PPUCTRL, PPUMASK, PPUSTATUS, OAMADDR, OAMDATA uint8
	PPUSCROLL, PPUADDR, PPUDATA                   uint8
	V, T                                          uint16
	X, XTemp, W                                   uint8
	ScrollY                                       uint8
	ReadBuffer                                    uint8
	Cycle                                         int32
	Scanline                                      int32
	Frame                                         uint64
	NMIRequested                                  bool
	VblSuppressed                                 bool
	NmiAssertCountdown                            uint8
	OddFrame                                      bool
	VRAM                                          [0x4000]uint8
	OAM                                           [256]uint8
	PaletteRAM                                    [32]uint8
	PaletteEmphasis                               uint8
}

// SaveState writes PPU + palette state to w.
func (p *PPU) SaveState(w io.Writer) error {
	s := ppuState{
		PPUCTRL: p.PPUCTRL, PPUMASK: p.PPUMASK, PPUSTATUS: p.PPUSTATUS,
		OAMADDR: p.OAMADDR, OAMDATA: p.OAMDATA,
		PPUSCROLL: p.PPUSCROLL, PPUADDR: p.PPUADDR, PPUDATA: p.PPUDATA,
		V: p.v, T: p.t, X: p.x, XTemp: p.xTemp, W: p.w,
		ScrollY:            p.ScrollY,
		ReadBuffer:         p.readBuffer,
		Cycle:              int32(p.Cycle),
		Scanline:           int32(p.Scanline),
		Frame:              p.Frame,
		NMIRequested:       p.NMIRequested,
		VblSuppressed:      p.vblSuppressed,
		NmiAssertCountdown: p.nmiAssertCountdown,
		OddFrame:           p.oddFrame,
		VRAM:               p.VRAM,
		OAM:                p.OAM,
	}
	if p.PaletteManager != nil {
		s.PaletteRAM = p.PaletteManager.PaletteRAM
		s.PaletteEmphasis = p.PaletteManager.Emphasis
	}
	return binary.Write(w, binary.LittleEndian, &s)
}

// LoadState restores PPU state written by SaveState.
func (p *PPU) LoadState(r io.Reader) error {
	var s ppuState
	if err := binary.Read(r, binary.LittleEndian, &s); err != nil {
		return err
	}
	p.PPUCTRL, p.PPUMASK, p.PPUSTATUS = s.PPUCTRL, s.PPUMASK, s.PPUSTATUS
	p.OAMADDR, p.OAMDATA = s.OAMADDR, s.OAMDATA
	p.PPUSCROLL, p.PPUADDR, p.PPUDATA = s.PPUSCROLL, s.PPUADDR, s.PPUDATA
	p.v, p.t, p.x, p.xTemp, p.w = s.V, s.T, s.X, s.XTemp, s.W
	p.ScrollY = s.ScrollY
	p.readBuffer = s.ReadBuffer
	p.Cycle, p.Scanline = int(s.Cycle), int(s.Scanline)
	p.Frame = s.Frame
	p.NMIRequested = s.NMIRequested
	p.vblSuppressed = s.VblSuppressed
	p.nmiAssertCountdown = s.NmiAssertCountdown
	p.oddFrame = s.OddFrame
	p.VRAM = s.VRAM
	p.OAM = s.OAM
	if p.PaletteManager != nil {
		p.PaletteManager.PaletteRAM = s.PaletteRAM
		p.PaletteManager.Emphasis = s.PaletteEmphasis
		// PaletteRAM/Emphasis were set by direct field assignment (bypassing
		// WritePalette/SetEmphasis), so refresh the derived color cache.
		p.PaletteManager.rebuildColorCache()
	}
	p.invalidateRenderCache()
	return nil
}

// invalidateRenderCache drops any tile/sprite data cached by the renderer
// for the current scanline. Call after restoring VRAM/OAM so the next
// pixel fetch re-reads from the freshly loaded state.
func (p *PPU) invalidateRenderCache() {
	p.currentBGTileX = -1
	p.currentSpriteCount = 0
}

