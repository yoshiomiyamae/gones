package test

import (
	"os"
	"testing"
)

// TestHeberekeBoots is a smoke test for mapper 69 (Sunsoft FME-7),
// exercising the multi-bank PRG layout, 1 KiB CHR banks, mapper-
// controlled mirroring, and the CPU-rate IRQ counter (Hebereke's
// intro animation drives all four). The game's title screen is the
// "へべれけ HEBEREKE © 1991 SUNSOFT" splash with four character
// sprites in the lower half — enough distinct colors that a blank or
// junked frame is easy to flag.
func TestHeberekeBoots(t *testing.T) {
	const romPath = `R:\Hebereke (J).nes`
	if _, err := os.Stat(romPath); err != nil {
		t.Skipf("ROM missing: %v", err)
	}
	sys := loadNES(t, romPath)
	for i := 0; i < 600; i++ {
		sys.StepFrame()
	}
	fb := sys.GetDisplayFramebufferRaw()
	hist, _, dominantCount := frameHistogram(fb)
	if len(hist) < 4 {
		t.Fatalf("framebuffer has only %d distinct colors — likely stuck/blank", len(hist))
	}
	if dominantCount > len(fb)*98/100 {
		t.Fatalf("framebuffer ≥98%% one color (%d / %d) — game not rendering", dominantCount, len(fb))
	}
	t.Logf("distinct colors=%d, dominant pixel count=%d/%d", len(hist), dominantCount, len(fb))
}
