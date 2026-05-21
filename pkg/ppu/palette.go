package ppu

import "github.com/yoshiomiyamaegones/pkg/logger"

// NES master palette - 64 colors total
// Each color is represented as RGB values
var masterPalette = [64][3]uint8{
	// 0x00-0x0F
	{0x80, 0x80, 0x80}, {0x00, 0x3D, 0xA6}, {0x00, 0x12, 0xB0}, {0x44, 0x00, 0x96},
	{0xA1, 0x00, 0x5E}, {0xC7, 0x00, 0x28}, {0xBA, 0x06, 0x00}, {0x8C, 0x17, 0x00},
	{0x5C, 0x2F, 0x00}, {0x10, 0x45, 0x00}, {0x05, 0x4A, 0x00}, {0x00, 0x47, 0x2E},
	{0x00, 0x41, 0x66}, {0x00, 0x00, 0x00}, {0x05, 0x05, 0x05}, {0x05, 0x05, 0x05},

	// 0x10-0x1F
	{0xC7, 0xC7, 0xC7}, {0x00, 0x77, 0xFF}, {0x21, 0x55, 0xFF}, {0x82, 0x37, 0xFA},
	{0xEB, 0x2F, 0xB5}, {0xFF, 0x29, 0x50}, {0xFF, 0x22, 0x00}, {0xD6, 0x32, 0x00},
	{0xC4, 0x62, 0x00}, {0x35, 0x80, 0x00}, {0x05, 0x8F, 0x00}, {0x00, 0x8A, 0x55},
	{0x00, 0x99, 0xCC}, {0x21, 0x21, 0x21}, {0x09, 0x09, 0x09}, {0x09, 0x09, 0x09},

	// 0x20-0x2F
	{0xFF, 0xFF, 0xFF}, {0x0F, 0xD7, 0xFF}, {0x69, 0xA2, 0xFF}, {0xD4, 0x80, 0xFF},
	{0xFF, 0x45, 0xF3}, {0xFF, 0x61, 0x8B}, {0xFF, 0x88, 0x33}, {0xFF, 0x9C, 0x12},
	{0xFA, 0xBC, 0x20}, {0x9F, 0xE3, 0x0E}, {0x2B, 0xF0, 0x35}, {0x0C, 0xF0, 0xA4},
	{0x05, 0xFB, 0xFF}, {0x5E, 0x5E, 0x5E}, {0x0D, 0x0D, 0x0D}, {0x0D, 0x0D, 0x0D},

	// 0x30-0x3F
	{0xFF, 0xFF, 0xFF}, {0xA6, 0xFC, 0xFF}, {0xB3, 0xEC, 0xFF}, {0xDA, 0xAB, 0xEB},
	{0xFF, 0xA8, 0xF9}, {0xFF, 0xAB, 0xB3}, {0xFF, 0xD2, 0xB0}, {0xFF, 0xEF, 0xA6},
	{0xFF, 0xF7, 0x9C}, {0xD7, 0xFF, 0xB3}, {0xC6, 0xFF, 0xDE}, {0xC4, 0xFF, 0xF6},
	{0xC4, 0xF0, 0xFF}, {0xCC, 0xCC, 0xCC}, {0x3C, 0x3C, 0x3C}, {0x3C, 0x3C, 0x3C},
}

// PaletteManager manages NES palette operations
type PaletteManager struct {
	// Palette RAM (32 bytes)
	// 0x00-0x0F: Background palettes (4 palettes × 4 colors)
	// 0x10-0x1F: Sprite palettes (4 palettes × 4 colors)
	// Note: 0x10, 0x14, 0x18, 0x1C mirror to 0x00, 0x04, 0x08, 0x0C
	PaletteRAM [32]uint8

	// Emphasis bits for color modification
	Emphasis uint8 // bits 5-7 of PPUMASK

	// bgColorCache / sprColorCache hold the final ARGB color for every
	// (palette<<2 | colorIndex) pair, with palette-mirroring, the
	// colorIndex-0 backdrop/transparent rule, and emphasis already folded in.
	// GetBackgroundColor / GetSpriteColor are called once per pixel, so this
	// turns that hot path into a single array index. Rebuilt by
	// rebuildColorCache whenever palette RAM or emphasis changes (rare).
	bgColorCache  [16]uint32
	sprColorCache [16]uint32
}

// NewPaletteManager creates a new palette manager
func NewPaletteManager() *PaletteManager {
	pm := &PaletteManager{}
	// Initialize palette RAM with proper power-up state
	// Universal backdrop should be a reasonable default color
	pm.PaletteRAM[0] = 0x0F // Universal backdrop (black/dark gray)

	// Initialize other palette entries to reasonable defaults
	for i := 1; i < len(pm.PaletteRAM); i++ {
		pm.PaletteRAM[i] = 0x30 // Light gray/white for better visibility during debugging
	}

	// Set some basic palettes for debugging
	// Background palette 0: black, white, light gray, dark gray
	pm.PaletteRAM[0] = 0x0F // Black backdrop
	pm.PaletteRAM[1] = 0x30 // White
	pm.PaletteRAM[2] = 0x10 // Light gray
	pm.PaletteRAM[3] = 0x00 // Dark gray

	logger.LogPPU("PaletteManager initialized with debugging colors")
	pm.rebuildColorCache()
	return pm
}

// ReadPalette reads a palette value with mirroring
func (pm *PaletteManager) ReadPalette(addr uint8) uint8 {
	addr = addr & 0x1F // Ensure within palette range

	// Handle mirroring of backdrop colors
	// $10, $14, $18, $1C mirror to $00, $04, $08, $0C respectively
	if addr == 0x10 {
		addr = 0x00
	} else if addr == 0x14 {
		addr = 0x04
	} else if addr == 0x18 {
		addr = 0x08
	} else if addr == 0x1C {
		addr = 0x0C
	}

	value := pm.PaletteRAM[addr]

	return value
}

// WritePalette writes a palette value with mirroring
func (pm *PaletteManager) WritePalette(addr uint8, value uint8) {
	addr = addr & 0x1F // Ensure within palette range

	if logger.PPUEnabled() {
		logger.LogPPU("WritePalette: original addr=$%02X, value=$%02X", addr, value)
	}

	// Handle mirroring of backdrop colors
	// $10, $14, $18, $1C mirror to $00, $04, $08, $0C respectively
	if addr == 0x10 {
		addr = 0x00
	} else if addr == 0x14 {
		addr = 0x04
	} else if addr == 0x18 {
		addr = 0x08
	} else if addr == 0x1C {
		addr = 0x0C
	}

	if logger.PPUEnabled() {
		logger.LogPPU("WritePalette: final addr=$%02X, value=$%02X", addr, value)
	}
	pm.PaletteRAM[addr] = value & 0x3F // Only 6 bits used
	pm.rebuildColorCache()
}

// rebuildColorCache recomputes bgColorCache / sprColorCache from the current
// palette RAM and emphasis. Folds in the colorIndex-0 rule (BG color 0 of
// every palette is the universal backdrop; sprite color 0 is transparent) and
// the $10/$14/$18/$1C backdrop mirroring done by ReadPalette. Cheap (32
// entries) and only called when palette RAM or emphasis changes.
func (pm *PaletteManager) rebuildColorCache() {
	for pal := uint8(0); pal < 4; pal++ {
		for ci := uint8(0); ci < 4; ci++ {
			i := pal*4 + ci
			if ci == 0 {
				// Background color 0 = universal backdrop ($3F00).
				pm.bgColorCache[i] = pm.getARGBColor(pm.ReadPalette(0))
				// Sprite color 0 = transparent.
				pm.sprColorCache[i] = 0x00000000
				continue
			}
			pm.bgColorCache[i] = pm.getARGBColor(pm.ReadPalette(i))
			pm.sprColorCache[i] = pm.getARGBColor(pm.ReadPalette(0x10 + i))
		}
	}
}

// GetBackgroundColor returns the ARGB color for a background pixel via the
// precomputed cache (palette/colorIndex each 0-3).
func (pm *PaletteManager) GetBackgroundColor(palette uint8, colorIndex uint8) uint32 {
	if palette > 3 || colorIndex > 3 {
		return 0xFF000000 // Black
	}
	return pm.bgColorCache[palette*4+colorIndex]
}

// GetSpriteColor returns the ARGB color for a sprite pixel via the precomputed
// cache; color 0 reads back as transparent (alpha 0).
func (pm *PaletteManager) GetSpriteColor(palette uint8, colorIndex uint8) uint32 {
	if palette > 3 || colorIndex > 3 {
		return 0x00000000 // Transparent
	}
	return pm.sprColorCache[palette*4+colorIndex]
}

// argbLUT[emphasis>>5][paletteIndex&0x3F] holds the precomputed ARGB
// value for every (palette index, emphasis-bit-combination) pair. The
// 3-bit emphasis index packs PPUMASK bits 5-7 (red/green/blue) into
// bits 0-2. Lookup replaces the per-pixel float32 channel dimming.
var argbLUT [8][64]uint32

func init() {
	for em := 0; em < 8; em++ {
		for idx := 0; idx < 64; idx++ {
			r := masterPalette[idx][0]
			g := masterPalette[idx][1]
			b := masterPalette[idx][2]
			if em&0x1 == 0 {
				r = uint8(float32(r) * 0.75)
			}
			if em&0x2 == 0 {
				g = uint8(float32(g) * 0.75)
			}
			if em&0x4 == 0 {
				b = uint8(float32(b) * 0.75)
			}
			argbLUT[em][idx] = 0xFF000000 | uint32(r)<<16 | uint32(g)<<8 | uint32(b)
		}
	}
}

// getARGBColor converts a 6-bit palette index to 32-bit ARGB color.
func (pm *PaletteManager) getARGBColor(paletteIndex uint8) uint32 {
	return argbLUT[pm.Emphasis>>5][paletteIndex&0x3F]
}

// SetEmphasis sets the color emphasis bits and refreshes the color cache,
// since emphasis changes the ARGB value of every palette entry.
func (pm *PaletteManager) SetEmphasis(emphasis uint8) {
	e := emphasis & 0xE0 // Only bits 5-7
	if e == pm.Emphasis {
		return
	}
	pm.Emphasis = e
	pm.rebuildColorCache()
}

// GetPaletteDebugInfo returns debug information about current palettes
func (pm *PaletteManager) GetPaletteDebugInfo() map[string]interface{} {
	debug := make(map[string]interface{})

	// Background palettes
	bgPalettes := make([][]uint32, 4)
	for palette := 0; palette < 4; palette++ {
		bgPalettes[palette] = make([]uint32, 4)
		for color := 0; color < 4; color++ {
			bgPalettes[palette][color] = pm.GetBackgroundColor(uint8(palette), uint8(color))
		}
	}
	debug["background_palettes"] = bgPalettes

	// Sprite palettes
	spritePalettes := make([][]uint32, 4)
	for palette := 0; palette < 4; palette++ {
		spritePalettes[palette] = make([]uint32, 4)
		for color := 0; color < 4; color++ {
			spritePalettes[palette][color] = pm.GetSpriteColor(uint8(palette), uint8(color))
		}
	}
	debug["sprite_palettes"] = spritePalettes

	debug["emphasis"] = pm.Emphasis
	debug["palette_ram"] = pm.PaletteRAM

	return debug
}
