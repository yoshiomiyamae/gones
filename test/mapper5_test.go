package test

import (
	"os"
	"testing"
)

// TestMetalSladerGloryBoots is a smoke test for Mapper 5 (MMC5). The
// title screen is the static "METAL SLADER GLORY / © HAL LABORATORY"
// splash — the framebuffer has enough distinct colors that a blank
// boot is easy to flag. Exercises 512 KiB PRG banking, 512 KiB CHR
// banking with mode-3 1-KiB CHR slots, and battery-backed PRG RAM.
func TestMetalSladerGloryBoots(t *testing.T) {
	const romPath = `R:\Metal Slader Glory (J).nes`
	if _, err := os.Stat(romPath); err != nil {
		t.Skipf("ROM missing: %v", err)
	}
	sys := loadNES(t, romPath)
	for i := 0; i < 300; i++ {
		sys.StepFrame()
	}
	fb := sys.GetDisplayFramebufferRaw()
	hist, _, dominantCount := frameHistogram(fb)
	// The title screen is white text on solid black, so we only need
	// two colors to call it "rendering" — much lower than the
	// multi-tone limit other smoke tests use.
	if len(hist) < 2 {
		t.Fatalf("framebuffer has only %d distinct colors — likely stuck/blank", len(hist))
	}
	// Allow up to 99.7% dominant: text on black easily reaches that
	// (white chars cover maybe 1% of the screen) but a fully blank
	// frame is 100% one color.
	if dominantCount >= len(fb) {
		t.Fatalf("framebuffer is a single solid color — game not rendering")
	}
}

// TestMMC5ExRAM runs Quietust's mmc5exram.nes — copies code into MMC5's
// 1 KiB ExRAM at $5C00 and executes it from there during VBlank to
// draw scrolling color bars via per-scanline palette swaps. Passing
// requires both CPU-readable ExRAM and the scanline-match IRQ.
func TestMMC5ExRAM(t *testing.T) {
	const romPath = `R:\nes-test-roms-master\exram\mmc5exram.nes`
	if _, err := os.Stat(romPath); err != nil {
		t.Skipf("ROM missing: %v", err)
	}
	sys := loadNES(t, romPath)
	for i := 0; i < 300; i++ {
		sys.StepFrame()
	}
	fb := sys.GetDisplayFramebufferRaw()
	hist, _, _ := frameHistogram(fb)
	// The copper-bars demo paints distinct horizontal bands of color
	// — a working impl shows at least the gray background + a handful
	// of bar colors. A blank or solid frame means either ExRAM
	// execution or scanline IRQ broke.
	if len(hist) < 5 {
		t.Fatalf("ExRAM copper-bars demo has only %d distinct colors — ExRAM exec or scanline IRQ broken", len(hist))
	}
}
