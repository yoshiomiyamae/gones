package ppu

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

// SpriteInfo represents a sprite with its OAM index plus the pattern-row
// bytes for the scanline it was evaluated on. PatternLo/PatternHi are fetched
// once per scanline in evaluateSprites (vertical flip already applied), so
// per-pixel rendering only shifts out bits — no CHR fetch per pixel.
type SpriteInfo struct {
	SpriteData
	OAMIndex  uint8 // Original index in OAM (for sprite 0 detection)
	PatternLo uint8 // Low bit plane for this scanline's row
	PatternHi uint8 // High bit plane for this scanline's row
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
	return BackgroundTile{
		TileIndex:  tileIndex,
		Attributes: attributes,
		PatternLo:  p.readVRAM(tileAddr + fineY),
		PatternHi:  p.readVRAM(tileAddr + fineY + 8),
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

// evaluateSprites fills currentSprites/currentSpriteCount with the (up to 8)
// sprites overlapping scanline, pre-fetching each one's pattern-row bytes, and
// runs the next-scanline sprite-overflow evaluation that the PPU does during
// cycles 65-256 of scanline N (i.e. for scanline N+1). The next-scanline pass
// is what lets the overflow flag fire when 9+ sprites have Y=239 — those
// render at scanline 240 (post-render), so the current-scanline count never
// sees them.
func (p *PPU) evaluateSprites(scanline int) {
	spriteHeight := 8
	if p.PPUCTRL&PPUCTRLSpriteSize != 0 {
		spriteHeight = 16
	}

	// Scan through OAM for sprites on this scanline.
	// OAM Y stores (actual_screen_Y - 1) — a sprite with OAM Y = 143 first
	// appears at scanline 144 — so we shift by +1 here and in spritePixelAt.
	// Secondary OAM holds 8 sprites; overflow fires when a 9th matches.
	count := 0
	for i := 0; i < totalOAMSprites; i++ {
		spriteY := int(p.OAM[i*4]) + 1

		if scanline >= spriteY && scanline < spriteY+spriteHeight {
			// The 9th+ sprite latches the overflow flag exactly as on
			// hardware. NoSpriteLimit keeps it for rendering anyway.
			if count >= maxSpritesPerScanline {
				p.PPUSTATUS |= PPUSTATUSSpriteOverflow
				if !p.NoSpriteLimit {
					break
				}
			}
			s := SpriteInfo{
				SpriteData: SpriteData{
					Y:          p.OAM[i*4],
					TileIndex:  p.OAM[i*4+1],
					Attributes: p.OAM[i*4+2],
					X:          p.OAM[i*4+3],
				},
				OAMIndex: uint8(i),
			}
			p.fetchSpritePattern(&s, scanline, spriteHeight)
			p.currentSprites[count] = s
			count++
		}
	}
	p.currentSpriteCount = count

	// Next-scanline overflow lookahead. Independent count — the real PPU's
	// eval phase doesn't know about the render list.
	next := scanline + 1
	lookahead := 0
	for i := 0; i < totalOAMSprites; i++ {
		spriteY := int(p.OAM[i*4]) + 1
		if next >= spriteY && next < spriteY+spriteHeight {
			lookahead++
			if lookahead > maxSpritesPerScanline {
				p.PPUSTATUS |= PPUSTATUSSpriteOverflow
				break
			}
		}
	}
}

// fetchSpritePattern fetches the two pattern-plane bytes for the row of sprite
// that lands on scanline, applying vertical flip and 8×16 tile selection.
// Sprite fetches route through readVRAMSprite so MMC5 8×16 picks its sprite
// CHR set.
func (p *PPU) fetchSpritePattern(sprite *SpriteInfo, scanline, spriteHeight int) {
	pixelY := scanline - (int(sprite.Y) + 1)
	if sprite.Attributes&SpriteFlipVertical != 0 {
		pixelY = (spriteHeight - 1) - pixelY
	}

	var tileAddr uint16
	if spriteHeight == 16 {
		// 8×16 sprites: bit 0 of the tile index selects the pattern table,
		// the rest is the (even) top tile; the bottom tile is the next one.
		tileIndex := sprite.TileIndex & 0xFE
		if pixelY >= 8 {
			tileIndex++
			pixelY -= 8
		}
		patternTableBase := uint16(0x0000)
		if sprite.TileIndex&1 != 0 {
			patternTableBase = 0x1000
		}
		tileAddr = patternTableBase + uint16(tileIndex)*16 + uint16(pixelY)
	} else {
		patternTableBase := uint16(0x0000)
		if p.PPUCTRL&PPUCTRLSpriteTable != 0 {
			patternTableBase = 0x1000
		}
		tileAddr = patternTableBase + uint16(sprite.TileIndex)*16 + uint16(pixelY)
	}

	sprite.PatternLo = p.readVRAMSprite(tileAddr)
	sprite.PatternHi = p.readVRAMSprite(tileAddr + 8)
}

// spritePixelAt returns the front-most opaque sprite pixel covering screen x
// on the current scanline (sprites already evaluated for this scanline). It
// only shifts bits out of the pre-fetched pattern bytes — no CHR fetch. The y
// range was already checked in evaluateSprites, so only the x range is tested
// here. Returns (color, priorityFront, isSprite0).
func (p *PPU) spritePixelAt(x int) (uint32, bool, bool) {
	if p.PPUMASK&PPUMASKSpriteShow == 0 {
		return 0x00000000, false, false
	}
	if x < 8 && p.PPUMASK&PPUMASKSpriteLeft == 0 {
		return 0x00000000, false, false
	}

	// Highest priority (lowest OAM index) first.
	for i := 0; i < p.currentSpriteCount; i++ {
		sprite := &p.currentSprites[i]
		spriteX := int(sprite.X)
		if x < spriteX || x >= spriteX+8 {
			continue
		}

		pixelX := x - spriteX
		if sprite.Attributes&SpriteFlipHorizontal != 0 {
			pixelX = 7 - pixelX
		}

		colorIndex := getPixelColor(sprite.PatternLo, sprite.PatternHi, pixelX)
		// Color 0 is transparent for sprites — keep scanning lower-priority
		// sprites at this x.
		if colorIndex != 0 {
			palette := sprite.Attributes & SpritePaletteMask
			return p.PaletteManager.GetSpriteColor(palette, colorIndex),
				sprite.Attributes&SpritePriority == 0,
				sprite.OAMIndex == 0
		}
	}

	return 0x00000000, false, false
}

// renderPixel renders a single pixel combining background and sprites. The
// caller (StepN) passes the current cycle/scanline as x/y and guarantees
// y ∈ [0,240) and x ∈ [0,256), so no bounds check is needed here.
func (p *PPU) renderPixel(x, y int) {
	index := y*256 + x

	if !p.renderEnabled {
		// Rendering disabled, just set background color
		p.FrameBuffer[index] = p.PaletteManager.GetBackgroundColor(0, 0)
		return
	}

	// Background pixel. The tile is fetched lazily and cached, refetching only
	// when crossing an 8-pixel boundary so a mid-scanline $2000/$2005/$2006
	// write propagates to later tiles. bgColorIndex (0 = transparent) drives
	// sprite priority / sprite-0 hit and is reused below so the tile is decoded
	// only once. BG off or left-clip shows the backdrop as transparent index 0.
	var bgColorIndex uint8
	var bgColor uint32
	if p.PPUMASK&PPUMASKBGShow == 0 || (x < 8 && p.PPUMASK&PPUMASKBGLeft == 0) {
		bgColor = p.PaletteManager.GetBackgroundColor(0, 0)
	} else {
		adjustedX := x + int(p.x)
		if tileX := adjustedX >> 3; tileX != p.currentBGTileX {
			p.currentBGTile = p.fetchBackgroundTileWithScroll(tileX)
			p.currentBGTileX = tileX
		}
		t := &p.currentBGTile
		bgColorIndex = getPixelColor(t.PatternLo, t.PatternHi, adjustedX&7)
		bgColor = p.PaletteManager.GetBackgroundColor(t.Attributes, bgColorIndex)
	}

	// Evaluate sprites once per scanline (fetches each sprite's pattern row).
	if x == 0 {
		p.evaluateSprites(y)
	}

	finalColor := bgColor
	if p.currentSpriteCount > 0 {
		spriteColor, spritePriority, sprite0Hit := p.spritePixelAt(x)

		if spriteColor&0xFF000000 != 0 { // Sprite pixel is not transparent
			bgOpaque := bgColorIndex != 0

			// Sprite priority: 0 = in front of BG, 1 = behind BG. Hardware
			// gates on the BG *pattern* value (color 0 = transparent), NOT
			// on the rendered RGB — palette tricks like Metal Slader Glory's
			// title screen where palette 3 maps both color 0 and color 1 to
			// $0F still need BG color 1 to occlude the sprite.
			if spritePriority || !bgOpaque {
				finalColor = spriteColor
			}

			// Sprite 0 hit detection.
			//
			// NESdev: sprite 0 hit never fires at x=255 (the PPU's output
			// pipeline rules out that final dot for an "obscure" pixel-
			// output-circuitry reason). blargg's right_edge test exercises
			// it directly.
			if sprite0Hit && p.PPUSTATUS&PPUSTATUSSprite0Hit == 0 && x != 255 {
				spriteEnabled := p.PPUMASK&PPUMASKSpriteShow != 0
				bgEnabled := p.PPUMASK&PPUMASKBGShow != 0
				leftClipped := x < 8 && (p.PPUMASK&(PPUMASKSpriteLeft|PPUMASKBGLeft)) != (PPUMASKSpriteLeft|PPUMASKBGLeft)

				if bgOpaque && spriteEnabled && bgEnabled && !leftClipped {
					p.PPUSTATUS |= PPUSTATUSSprite0Hit
				}
			}
		}
	}

	p.FrameBuffer[index] = finalColor
}
