package ppu

import (
	"testing"
)

// Test palette manager creation
func TestPaletteManagerCreation(t *testing.T) {
	pm := NewPaletteManager()
	
	if pm == nil {
		t.Error("PaletteManager should not be nil")
	}
	
	// Check initial state
	if pm.Emphasis != 0 {
		t.Errorf("Expected emphasis=0, got %02X", pm.Emphasis)
	}
}

// Test palette read/write operations
func TestPaletteReadWrite(t *testing.T) {
	pm := NewPaletteManager()
	
	// Write to palette
	pm.WritePalette(0x01, 0x30)
	
	// Read back
	value := pm.ReadPalette(0x01)
	if value != 0x30 {
		t.Errorf("Expected palette value 0x30, got %02X", value)
	}
	
	// Test 6-bit masking
	pm.WritePalette(0x02, 0xFF)
	value = pm.ReadPalette(0x02)
	if value != 0x3F {
		t.Errorf("Expected palette value 0x3F (masked), got %02X", value)
	}
}

// Test backdrop color mirroring
func TestBackdropMirroring(t *testing.T) {
	pm := NewPaletteManager()
	
	// Write to universal backdrop (0x00)
	pm.WritePalette(0x00, 0x0F)
	
	// Check mirrored locations (these mirror to their respective backdrop colors)
	// $10 mirrors to $00, $14 to $04, $18 to $08, $1C to $0C
	testCases := []struct {
		addr     uint8
		expected uint8
	}{
		{0x10, 0x0F}, // Should read from $00
		{0x14, 0x30}, // Should read from $04 (default initialization)
		{0x18, 0x30}, // Should read from $08 (default initialization)
		{0x1C, 0x30}, // Should read from $0C (default initialization)
	}
	
	for _, tc := range testCases {
		value := pm.ReadPalette(tc.addr)
		if value != tc.expected {
			t.Errorf("Expected mirrored value 0x%02X at address %02X, got %02X", tc.expected, tc.addr, value)
		}
	}
	
	// Write to mirrored location
	pm.WritePalette(0x10, 0x20)
	
	// Check original location
	value := pm.ReadPalette(0x00)
	if value != 0x20 {
		t.Errorf("Expected backdrop value 0x20, got %02X", value)
	}
}

// Test background color retrieval
func TestBackgroundColors(t *testing.T) {
	pm := NewPaletteManager()
	
	// Set up a background palette
	pm.WritePalette(0x00, 0x0F) // Universal backdrop
	pm.WritePalette(0x01, 0x30) // Palette 0, color 1
	pm.WritePalette(0x02, 0x27) // Palette 0, color 2
	pm.WritePalette(0x03, 0x17) // Palette 0, color 3
	
	// Test color retrieval
	color0 := pm.GetBackgroundColor(0, 0)
	color1 := pm.GetBackgroundColor(0, 1)
	color2 := pm.GetBackgroundColor(0, 2)
	color3 := pm.GetBackgroundColor(0, 3)
	
	// Colors should be different
	if color0 == color1 || color1 == color2 || color2 == color3 {
		t.Error("Background colors should be different")
	}
	
	// Test universal backdrop (any palette, color 0 should return same color)
	backdropFromPalette1 := pm.GetBackgroundColor(1, 0)
	if color0 != backdropFromPalette1 {
		t.Error("Universal backdrop should be same for all palettes")
	}
}

// Test sprite color retrieval
func TestSpriteColors(t *testing.T) {
	pm := NewPaletteManager()
	
	// Set up a sprite palette
	pm.WritePalette(0x11, 0x30) // Sprite palette 0, color 1
	pm.WritePalette(0x12, 0x27) // Sprite palette 0, color 2
	pm.WritePalette(0x13, 0x17) // Sprite palette 0, color 3
	
	// Test color retrieval
	color0 := pm.GetSpriteColor(0, 0) // Should be transparent
	color1 := pm.GetSpriteColor(0, 1)
	color2 := pm.GetSpriteColor(0, 2)
	color3 := pm.GetSpriteColor(0, 3)
	
	// Color 0 should be transparent (alpha = 0)
	if color0&0xFF000000 != 0x00000000 {
		t.Errorf("Sprite color 0 should be transparent, got %08X", color0)
	}
	
	// Other colors should be opaque
	if color1&0xFF000000 != 0xFF000000 {
		t.Errorf("Sprite color 1 should be opaque, got %08X", color1)
	}
	
	// Colors should be different
	if color1 == color2 || color2 == color3 {
		t.Error("Sprite colors should be different")
	}
}

// Test color emphasis
func TestColorEmphasis(t *testing.T) {
	pm := NewPaletteManager()
	
	// Set a test color
	pm.WritePalette(0x01, 0x30)
	
	// Get color without emphasis
	normalColor := pm.GetBackgroundColor(0, 1)
	
	// Set red emphasis
	pm.SetEmphasis(0x20)
	emphasizedColor := pm.GetBackgroundColor(0, 1)
	
	// Colors should be different with emphasis
	if normalColor == emphasizedColor {
		t.Error("Colors should be different with emphasis applied")
	}
	
	// Test multiple emphasis bits
	pm.SetEmphasis(0xE0) // All emphasis bits
	allEmphasisColor := pm.GetBackgroundColor(0, 1)
	
	if emphasizedColor == allEmphasisColor {
		t.Error("Different emphasis settings should produce different colors")
	}
}

// Test palette bounds checking
func TestPaletteBoundsChecking(t *testing.T) {
	pm := NewPaletteManager()
	
	// Test invalid palette numbers
	color := pm.GetBackgroundColor(4, 0) // Invalid palette
	if color != 0xFF000000 {
		t.Errorf("Invalid background palette should return black, got %08X", color)
	}
	
	color = pm.GetSpriteColor(4, 0) // Invalid palette
	if color != 0x00000000 {
		t.Errorf("Invalid sprite palette should return transparent, got %08X", color)
	}
	
	// Test invalid color indices
	color = pm.GetBackgroundColor(0, 4) // Invalid color
	if color != 0xFF000000 {
		t.Errorf("Invalid background color should return black, got %08X", color)
	}
	
	color = pm.GetSpriteColor(0, 4) // Invalid color
	if color != 0x00000000 {
		t.Errorf("Invalid sprite color should return transparent, got %08X", color)
	}
}

// Test master palette integrity
func TestMasterPalette(t *testing.T) {
	pm := NewPaletteManager()
	
	// Test that all 64 master palette colors are valid
	for i := 0; i < 64; i++ {
		pm.WritePalette(0x01, uint8(i))
		color := pm.GetBackgroundColor(0, 1)
		
		// Should be a valid ARGB color (alpha = 0xFF)
		if color&0xFF000000 != 0xFF000000 {
			t.Errorf("Master palette color %d should be opaque, got %08X", i, color)
		}
	}
}

// Test debug information
func TestPaletteDebugInfo(t *testing.T) {
	pm := NewPaletteManager()
	
	// Set up some palette data
	pm.WritePalette(0x01, 0x30)
	pm.WritePalette(0x11, 0x27)
	pm.SetEmphasis(0x20)
	
	// Get debug info
	debug := pm.GetPaletteDebugInfo()
	
	// Check that debug info contains expected keys
	if _, ok := debug["background_palettes"]; !ok {
		t.Error("Debug info should contain background_palettes")
	}
	if _, ok := debug["sprite_palettes"]; !ok {
		t.Error("Debug info should contain sprite_palettes")
	}
	if _, ok := debug["emphasis"]; !ok {
		t.Error("Debug info should contain emphasis")
	}
	if _, ok := debug["palette_ram"]; !ok {
		t.Error("Debug info should contain palette_ram")
	}
	
	// Check emphasis value
	if debug["emphasis"] != pm.Emphasis {
		t.Errorf("Debug emphasis should match actual emphasis")
	}
}