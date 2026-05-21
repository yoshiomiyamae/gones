package ppu

import (
	"testing"

	"github.com/yoshiomiyamaegones/pkg/memory"
)

// createTestPPU creates a PPU instance for testing
func createTestPPU() *PPU {
	mem := memory.New()
	ppu := New(mem)
	ppu.Reset()
	return ppu
}

// Test PPU Reset
func TestPPUReset(t *testing.T) {
	ppu := createTestPPU()

	// Set some non-default values
	ppu.PPUCTRL = 0xFF
	ppu.PPUMASK = 0xFF
	ppu.PPUSTATUS = 0xFF
	ppu.Cycle = 100
	ppu.Scanline = 50

	// Reset should restore defaults
	ppu.Reset()

	if ppu.PPUCTRL != 0 {
		t.Errorf("Expected PPUCTRL=0, got PPUCTRL=%02X", ppu.PPUCTRL)
	}
	if ppu.PPUMASK != 0 {
		t.Errorf("Expected PPUMASK=0, got PPUMASK=%02X", ppu.PPUMASK)
	}
	if ppu.PPUSTATUS != 0 {
		t.Errorf("Expected PPUSTATUS=0, got PPUSTATUS=%02X", ppu.PPUSTATUS)
	}
	if ppu.Cycle != 0 {
		t.Errorf("Expected Cycle=0, got Cycle=%d", ppu.Cycle)
	}
	if ppu.Scanline != 0 {
		t.Errorf("Expected Scanline=0, got Scanline=%d", ppu.Scanline)
	}
}

// TestNoSpriteLimit verifies the per-scanline sprite cap: the default mode
// renders at most 8 sprites and the NoSpriteLimit mode renders all of them,
// while both latch the overflow STATUS flag once a 9th sprite is present.
func TestNoSpriteLimit(t *testing.T) {
	ppu := createTestPPU()

	// Place 10 sprites all on scanline 10 (OAM Y = screen Y - 1, so Y=9).
	const scanline = 10
	for i := 0; i < 10; i++ {
		ppu.OAM[i*4] = scanline - 1   // Y
		ppu.OAM[i*4+1] = 0            // tile
		ppu.OAM[i*4+2] = 0            // attributes
		ppu.OAM[i*4+3] = uint8(i * 8) // X (spread across the line)
	}

	// Default: capped at 8, overflow flag latched by the 9th.
	ppu.PPUSTATUS = 0
	ppu.NoSpriteLimit = false
	ppu.evaluateSprites(scanline)
	if ppu.currentSpriteCount != 8 {
		t.Errorf("limited mode: want 8 sprites, got %d", ppu.currentSpriteCount)
	}
	if ppu.PPUSTATUS&PPUSTATUSSpriteOverflow == 0 {
		t.Error("limited mode: overflow flag should be set with 10 sprites on a line")
	}

	// No-limit: all 10 render, overflow still latched (game logic unaffected).
	// The sprite count is the load-bearing assertion for the feature (the
	// overflow flag can also latch via the next-scanline lookahead, so it
	// alone wouldn't prove the past-8 collection ran).
	ppu.PPUSTATUS = 0
	ppu.NoSpriteLimit = true
	ppu.evaluateSprites(scanline)
	if ppu.currentSpriteCount != 10 {
		t.Errorf("no-limit mode: want 10 sprites, got %d", ppu.currentSpriteCount)
	}
	// The 9th/10th sprites (past the hardware cap) must be collected in OAM
	// order so priority is preserved.
	if ppu.currentSprites[8].OAMIndex != 8 || ppu.currentSprites[9].OAMIndex != 9 {
		t.Errorf("no-limit mode: sprites past the cap not collected in OAM order: [8]=%d [9]=%d",
			ppu.currentSprites[8].OAMIndex, ppu.currentSprites[9].OAMIndex)
	}
	if ppu.PPUSTATUS&PPUSTATUSSpriteOverflow == 0 {
		t.Error("no-limit mode: overflow flag must still latch per the real 8-limit")
	}
}

// Test palette operations
func TestPaletteOperations(t *testing.T) {
	ppu := createTestPPU()

	// Test palette write/read
	ppu.WriteRegister(0x2006, 0x3F) // PPUADDR high
	ppu.WriteRegister(0x2006, 0x00) // PPUADDR low (palette 0)
	ppu.WriteRegister(0x2007, 0x0F) // Write color index 0x0F

	// Read back
	ppu.WriteRegister(0x2006, 0x3F) // PPUADDR high
	ppu.WriteRegister(0x2006, 0x00) // PPUADDR low
	value := ppu.ReadRegister(0x2007)

	if value != 0x0F {
		t.Errorf("Expected palette value 0x0F, got %02X", value)
	}
}

// Test palette mirroring
func TestPaletteMirroring(t *testing.T) {
	ppu := createTestPPU()

	// Write to backdrop color at 0x3F00
	ppu.WriteRegister(0x2006, 0x3F)
	ppu.WriteRegister(0x2006, 0x00)
	ppu.WriteRegister(0x2007, 0x20)

	// Read from mirrored location 0x3F10
	ppu.WriteRegister(0x2006, 0x3F)
	ppu.WriteRegister(0x2006, 0x10)
	value := ppu.ReadRegister(0x2007)

	if value != 0x20 {
		t.Errorf("Expected mirrored palette value 0x20, got %02X", value)
	}
}

// Test PPUSTATUS register
func TestPPUSTATUS(t *testing.T) {
	ppu := createTestPPU()

	// Set VBlank flag
	ppu.PPUSTATUS |= PPUSTATUSVBlank

	// Reading PPUSTATUS should clear VBlank flag
	status := ppu.ReadRegister(0x2002)

	if status&PPUSTATUSVBlank == 0 {
		t.Error("VBlank flag should be set before read")
	}

	// Check that flag is cleared after read
	status = ppu.ReadRegister(0x2002)
	if status&PPUSTATUSVBlank != 0 {
		t.Error("VBlank flag should be cleared after read")
	}
}

// Test OAM operations
func TestOAMOperations(t *testing.T) {
	ppu := createTestPPU()

	// Set OAM address
	ppu.WriteRegister(0x2003, 0x10) // OAMADDR

	// Write OAM data
	ppu.WriteRegister(0x2004, 0x50) // Y position
	ppu.WriteRegister(0x2004, 0x01) // Tile index
	ppu.WriteRegister(0x2004, 0x02) // Attributes
	ppu.WriteRegister(0x2004, 0x60) // X position

	// Check OAM data
	if ppu.OAM[0x10] != 0x50 {
		t.Errorf("Expected OAM[0x10]=0x50, got %02X", ppu.OAM[0x10])
	}
	if ppu.OAM[0x11] != 0x01 {
		t.Errorf("Expected OAM[0x11]=0x01, got %02X", ppu.OAM[0x11])
	}
	if ppu.OAM[0x12] != 0x02 {
		t.Errorf("Expected OAM[0x12]=0x02, got %02X", ppu.OAM[0x12])
	}
	if ppu.OAM[0x13] != 0x60 {
		t.Errorf("Expected OAM[0x13]=0x60, got %02X", ppu.OAM[0x13])
	}

	// Check OAMADDR increment
	if ppu.OAMADDR != 0x14 {
		t.Errorf("Expected OAMADDR=0x14, got %02X", ppu.OAMADDR)
	}
}

// Test frame timing
func TestFrameTiming(t *testing.T) {
	ppu := createTestPPU()

	// Simulate running to VBlank
	for ppu.Scanline < 241 || (ppu.Scanline == 241 && ppu.Cycle == 0) {
		ppu.Step()
	}

	// Should be in VBlank
	if ppu.PPUSTATUS&PPUSTATUSVBlank == 0 {
		t.Error("Should be in VBlank at scanline 241")
	}

	// Continue to end of frame
	for !ppu.FrameComplete {
		ppu.Step()
	}

	// Frame should be complete and VBlank cleared
	if !ppu.FrameComplete {
		t.Error("Frame should be complete")
	}
	if ppu.PPUSTATUS&PPUSTATUSVBlank != 0 {
		t.Error("VBlank should be cleared at end of frame")
	}
}

// Test VRAM address increment
func TestVRAMAddressIncrement(t *testing.T) {
	ppu := createTestPPU()

	// Test increment by 1 (default)
	ppu.WriteRegister(0x2006, 0x20) // PPUADDR high
	ppu.WriteRegister(0x2006, 0x00) // PPUADDR low
	ppu.WriteRegister(0x2007, 0xAA) // Write data

	// Address should increment by 1
	if ppu.v != 0x2001 {
		t.Errorf("Expected VRAM address 0x2001, got %04X", ppu.v)
	}

	// Test increment by 32
	ppu.PPUCTRL |= PPUCTRLIncrement
	ppu.WriteRegister(0x2006, 0x20) // PPUADDR high
	ppu.WriteRegister(0x2006, 0x00) // PPUADDR low
	ppu.WriteRegister(0x2007, 0xBB) // Write data

	// Address should increment by 32
	if ppu.v != 0x2020 {
		t.Errorf("Expected VRAM address 0x2020, got %04X", ppu.v)
	}
}

// Test scroll register writes
func TestScrollRegister(t *testing.T) {
	ppu := createTestPPU()

	// Write X scroll
	ppu.WriteRegister(0x2005, 0x08) // PPUSCROLL X

	if ppu.x != 0 { // Fine X should be 0 (8 >> 3 = 1, 8 & 7 = 0)
		t.Errorf("Expected fine X=0, got %d", ppu.x)
	}
	if ppu.w != 1 {
		t.Errorf("Expected write toggle=1, got %d", ppu.w)
	}

	// Write Y scroll
	ppu.WriteRegister(0x2005, 0x10) // PPUSCROLL Y

	if ppu.w != 0 {
		t.Errorf("Expected write toggle=0, got %d", ppu.w)
	}
}
