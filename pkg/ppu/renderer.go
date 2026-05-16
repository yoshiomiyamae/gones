package ppu

import (
	"github.com/yoshiomiyamaegones/pkg/logger"
)

// TileData represents an 8x8 pixel tile
type TileData struct {
	LowByte  uint8 // Low bit plane
	HighByte uint8 // High bit plane
}

// SpriteData represents sprite attribute data
type SpriteData struct {
	Y          uint8 // Y position - 1
	TileIndex  uint8 // Tile index
	Attributes uint8 // Attributes (palette, priority, flip)
	X          uint8 // X position
}

// BackgroundTile represents a background tile with attributes
type BackgroundTile struct {
	TileIndex  uint8 // Tile index from nametable
	Attributes uint8 // Attribute data (palette selection)
	PatternLo  uint8 // Low bit plane
	PatternHi  uint8 // High bit plane
}

// SpriteInfo represents a sprite with its OAM index
type SpriteInfo struct {
	SpriteData
	OAMIndex int // Original index in OAM (for sprite 0 detection)
}

// Sprite attribute flags
const (
	SpriteFlipHorizontal = 0x40
	SpriteFlipVertical   = 0x80
	SpritePriority       = 0x20 // 0=front of background, 1=behind background
	SpritePaletteMask    = 0x03 // Palette selection (bits 0-1)
)

// fetchBackgroundTileWithScroll fetches tile data for background rendering.
// v.coarseY / v.fineY are advanced per-scanline by PPU.incrementY(), so the
// row to fetch comes from v directly. Only the X axis adds the screen tile
// column to v.coarseX (with NT wrap), since v.coarseX is restored from t at
// the start of every scanline.
func (p *PPU) fetchBackgroundTileWithScroll(tileX int) BackgroundTile {
	coarseX := int(p.v & 0x1F)
	scrolledTileY := int((p.v >> 5) & 0x1F)
	scrolledTileX := coarseX + tileX

	nameTableX := 0
	if scrolledTileX >= 32 {
		nameTableX = 1
		scrolledTileX -= 32
	}

	baseNTX := int(p.v>>10) & 1
	baseNTY := int(p.v>>11) & 1

	finalNTX := (baseNTX + nameTableX) % 2
	finalNTY := baseNTY

	nameTableIndex := finalNTY*2 + finalNTX
	nameTableBase := uint16(0x2000) + uint16(nameTableIndex)*0x400
	nameTableAddr := nameTableBase + uint16(scrolledTileY*32+scrolledTileX)

	tileIndex := p.readVRAM(nameTableAddr)

	attrAddr := nameTableBase + 0x3C0 + uint16((scrolledTileY/4)*8+(scrolledTileX/4))
	attrByte := p.readVRAM(attrAddr)

	attrShift := ((scrolledTileY & 2) * 2) + ((scrolledTileX&2)/2)*2
	attributes := (attrByte >> attrShift) & 0x03

	patternTableBase := uint16(0x0000)
	if p.PPUCTRL&PPUCTRLBGTable != 0 {
		patternTableBase = 0x1000
	}

	tileAddr := patternTableBase + uint16(tileIndex)*16

	// v.fineY is the pixel row within the current tile for this scanline.
	fineY := (p.v >> 12) & 0x07
	patternLoAddr := tileAddr + fineY
	patternHiAddr := tileAddr + fineY + 8

	// Debug removed for performance

	patternLo := p.readVRAM(patternLoAddr)
	patternHi := p.readVRAM(patternHiAddr)

	// Debug: Log pattern reads for character rendering issues
	if tileIndex >= 0x10 && tileIndex <= 0x7F && (patternLo != 0 || patternHi != 0) {
		logger.LogPPU("BG Tile: idx=$%02X, addr=$%04X, patternLo=$%02X, patternHi=$%02X, table=$%04X",
			tileIndex, tileAddr, patternLo, patternHi, patternTableBase)
	}

	return BackgroundTile{
		TileIndex:  tileIndex,
		Attributes: attributes,
		PatternLo:  patternLo,
		PatternHi:  patternHi,
	}
}

// getPixelColor extracts pixel color from tile pattern data
func getPixelColor(patternLo, patternHi uint8, pixelX int) uint8 {
	// Extract bit for this pixel (MSB = leftmost pixel)
	bitPos := 7 - pixelX

	lowBit := (patternLo >> bitPos) & 1
	highBit := (patternHi >> bitPos) & 1

	colorIndex := (highBit << 1) | lowBit

	return colorIndex
}

// isBackgroundPixelOpaque checks if background pixel is opaque (non-zero
// color index). Reads from the pre-fetched scanline tile array — sprite 0
// hit detection always runs after prefetchScanlineTiles has filled it.
func (p *PPU) isBackgroundPixelOpaque(x, y int) bool {
	_ = y
	if p.PPUMASK&PPUMASKBGShow == 0 {
		return false
	}
	if x < 8 && p.PPUMASK&PPUMASKBGLeft == 0 {
		return false
	}

	fineX := int(p.x)
	adjustedX := x + fineX
	tileX := adjustedX / 8
	pixelX := adjustedX & 7

	tile := &p.scanlineTiles[tileX]
	return getPixelColor(tile.PatternLo, tile.PatternHi, pixelX) != 0
}

// prefetchScanlineTiles fills p.scanlineTiles with the 33 background
// tiles covering the visible 256 pixels plus a margin for fineX scroll.
// Called at Cycle 0 of each visible scanline after horizontal scroll
// restore; subsequent renderBackgroundPixelCached calls become O(1).
func (p *PPU) prefetchScanlineTiles() {
	for tileX := 0; tileX < 33; tileX++ {
		p.scanlineTiles[tileX] = p.fetchBackgroundTileWithScroll(tileX)
	}
}

// renderBackgroundPixelCached renders a single background pixel using
// the pre-fetched scanline tile array. v.coarseX/fineX are fixed for
// the scanline (no per-cycle coarseX advance in this PPU model), so a
// single prefetch covers all 256 pixels.
func (p *PPU) renderBackgroundPixelCached(x, y int) uint32 {
	_ = y
	if p.PPUMASK&PPUMASKBGShow == 0 {
		return p.PaletteManager.GetBackgroundColor(0, 0)
	}
	if x < 8 && p.PPUMASK&PPUMASKBGLeft == 0 {
		return p.PaletteManager.GetBackgroundColor(0, 0)
	}

	fineX := int(p.x)
	adjustedX := x + fineX
	tileX := adjustedX / 8
	pixelX := adjustedX & 7

	tile := &p.scanlineTiles[tileX]
	colorIndex := getPixelColor(tile.PatternLo, tile.PatternHi, pixelX)
	return p.PaletteManager.GetBackgroundColor(tile.Attributes, colorIndex)
}

// renderBackgroundPixel renders a single background pixel
func (p *PPU) renderBackgroundPixel(x, y int) uint32 {
	// Check if background rendering is enabled
	if p.PPUMASK&PPUMASKBGShow == 0 {
		return p.PaletteManager.GetBackgroundColor(0, 0) // Universal backdrop
	}

	// Check if we should hide background in leftmost 8 pixels
	if x < 8 && p.PPUMASK&PPUMASKBGLeft == 0 {
		return p.PaletteManager.GetBackgroundColor(0, 0) // Universal backdrop
	}

	// Apply fine X scroll for smooth scrolling
	fineX := int(p.x)
	adjustedX := x + fineX

	_ = y // Y comes from v register (advanced per scanline by incrementY).
	tileX := adjustedX / 8
	pixelX := adjustedX % 8

	tile := p.fetchBackgroundTileWithScroll(tileX)

	// Get pixel color from pattern data
	colorIndex := getPixelColor(tile.PatternLo, tile.PatternHi, pixelX)

	// Get final color from palette
	color := p.PaletteManager.GetBackgroundColor(tile.Attributes, colorIndex)

	return color
}

// fetchSpriteData fetches data for all sprites on current scanline
func (p *PPU) fetchSpriteData(scanline int) []SpriteInfo {
	var sprites []SpriteInfo
	spriteHeight := 8

	// Check sprite size
	if p.PPUCTRL&PPUCTRLSpriteSize != 0 {
		spriteHeight = 16
	}

	// Scan through OAM for sprites on this scanline.
	// OAM Y stores (actual_screen_Y - 1) — a sprite with OAM Y = 143 first
	// appears at scanline 144 — so we shift by +1 here and in renderSpritePixel.
	for i := 0; i < 64; i++ {
		spriteY := int(p.OAM[i*4]) + 1

		if scanline >= spriteY && scanline < spriteY+spriteHeight {
			sprite := SpriteInfo{
				SpriteData: SpriteData{
					Y:          p.OAM[i*4],
					TileIndex:  p.OAM[i*4+1],
					Attributes: p.OAM[i*4+2],
					X:          p.OAM[i*4+3],
				},
				OAMIndex: i,
			}
			sprites = append(sprites, sprite)

			// NES can only render 8 sprites per scanline
			if len(sprites) >= 8 {
				// Set sprite overflow flag
				p.PPUSTATUS |= 0x20
				break
			}
		}
	}

	return sprites
}

// renderSpritePixel renders sprite pixels for a given position
func (p *PPU) renderSpritePixel(x, y int, sprites []SpriteInfo) (uint32, bool, bool) {
	// Check if sprite rendering is enabled
	if p.PPUMASK&PPUMASKSpriteShow == 0 {
		return 0x00000000, false, false
	}

	// Check if we should hide sprites in leftmost 8 pixels
	if x < 8 && p.PPUMASK&PPUMASKSpriteLeft == 0 {
		return 0x00000000, false, false
	}

	spriteHeight := 8
	if p.PPUCTRL&PPUCTRLSpriteSize != 0 {
		spriteHeight = 16
	}

	// Check sprites from highest priority (index 0) to lowest.
	// OAM Y is the displayed Y - 1 (see fetchSpriteData), so add 1 here too.
	for _, sprite := range sprites {
		spriteX := int(sprite.X)
		spriteY := int(sprite.Y) + 1

		if x >= spriteX && x < spriteX+8 && y >= spriteY && y < spriteY+spriteHeight {
			pixelX := x - spriteX
			pixelY := y - spriteY

			// Apply horizontal flip
			if sprite.Attributes&SpriteFlipHorizontal != 0 {
				pixelX = 7 - pixelX
			}

			// Apply vertical flip
			if sprite.Attributes&SpriteFlipVertical != 0 {
				pixelY = (spriteHeight - 1) - pixelY
			}

			// Fetch pattern data
			patternTableBase := uint16(0x0000)
			if p.PPUCTRL&PPUCTRLSpriteTable != 0 {
				patternTableBase = 0x1000
			}

			var tileAddr uint16
			if spriteHeight == 16 {
				// 8x16 sprites use different addressing
				tileIndex := sprite.TileIndex & 0xFE
				if pixelY >= 8 {
					tileIndex++
					pixelY -= 8
				}
				if sprite.TileIndex&1 != 0 {
					patternTableBase = 0x1000
				} else {
					patternTableBase = 0x0000
				}
				tileAddr = patternTableBase + uint16(tileIndex)*16 + uint16(pixelY)
			} else {
				// 8x8 sprites
				tileAddr = patternTableBase + uint16(sprite.TileIndex)*16 + uint16(pixelY)
			}

			patternLo := p.readVRAM(tileAddr)
			patternHi := p.readVRAM(tileAddr + 8)

			// Get pixel color
			colorIndex := getPixelColor(patternLo, patternHi, pixelX)

			// Color 0 is transparent for sprites
			if colorIndex != 0 {
				palette := sprite.Attributes & SpritePaletteMask
				color := p.PaletteManager.GetSpriteColor(palette, colorIndex)
				priority := sprite.Attributes&SpritePriority == 0
				sprite0Hit := sprite.OAMIndex == 0 // Sprite 0 hit detection

				return color, priority, sprite0Hit
			}
		}
	}

	return 0x00000000, false, false
}

// renderPixel renders a single pixel combining background and sprites
func (p *PPU) renderPixel() {
	if p.Scanline < 0 || p.Scanline >= 240 || p.Cycle < 0 || p.Cycle >= 256 {
		return
	}

	x := p.Cycle
	y := p.Scanline
	index := y*256 + x

	// Quick bounds check
	if index < 0 || index >= len(p.FrameBuffer) {
		return
	}

	if !p.renderingEnabled() {
		// Rendering disabled, just set background color
		backgroundColor := p.PaletteManager.GetBackgroundColor(0, 0)
		p.FrameBuffer[index] = backgroundColor
		return
	}

	// Render background pixel with caching
	bgColor := p.renderBackgroundPixelCached(x, y)

	// Fetch sprites for this scanline (cache for efficiency)
	if p.Cycle == 0 {
		p.currentSprites = p.fetchSpriteData(p.Scanline)
	}

	// Early exit if no sprites to check
	if len(p.currentSprites) == 0 {
		p.FrameBuffer[index] = bgColor
		if p.renderingEnabled() {
			p.PersistentFrameBuffer[index] = bgColor
			p.renderingOccurred = true
		}
		return
	}

	// Render sprite pixel
	spriteColor, spritePriority, sprite0Hit := p.renderSpritePixel(x, y, p.currentSprites)

	// Determine final pixel color
	var finalColor uint32

	if spriteColor&0xFF000000 != 0 { // Sprite pixel is not transparent
		if spritePriority || (bgColor&0x00FFFFFF) == (p.PaletteManager.GetBackgroundColor(0, 0)&0x00FFFFFF) {
			finalColor = spriteColor
		} else {
			finalColor = bgColor
		}

		// Optimized sprite 0 hit detection
		if sprite0Hit && p.PPUSTATUS&PPUSTATUSSprite0Hit == 0 {
			bgOpaque := p.isBackgroundPixelOpaque(x, y)
			spriteEnabled := p.PPUMASK&PPUMASKSpriteShow != 0
			bgEnabled := p.PPUMASK&PPUMASKBGShow != 0
			leftClipped := x < 8 && (p.PPUMASK&(PPUMASKSpriteLeft|PPUMASKBGLeft)) != (PPUMASKSpriteLeft|PPUMASKBGLeft)

			if bgOpaque && spriteEnabled && bgEnabled && !leftClipped {
				p.PPUSTATUS |= PPUSTATUSSprite0Hit
			}
		}
	} else {
		finalColor = bgColor
	}

	// Write pixel to frame buffer
	p.FrameBuffer[index] = finalColor

	// Update persistent frame buffer if rendering is enabled
	if p.renderingEnabled() {
		p.PersistentFrameBuffer[index] = finalColor
		p.renderingOccurred = true
	}
}
