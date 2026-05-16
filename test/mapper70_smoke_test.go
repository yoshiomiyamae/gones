package test

import (
	"os"
	"testing"
)

// TestKamenRiderClubBoots is a smoke test for mapper 70 (Bandai). It loads
// the Kamen Rider Club ROM, runs ~3 seconds of emulation, and checks that
// the framebuffer contains real picture content (multiple distinct colors
// at non-trivial counts). If mapper 70 is misimplemented the game tends to
// either hang on a black screen or render obvious garbage.
func TestKamenRiderClubBoots(t *testing.T) {
	const romPath = `R:\Kamen Rider Club (J).nes`
	if _, err := os.Stat(romPath); err != nil {
		t.Skipf("ROM missing: %v", err)
	}
	sys := loadNES(t, romPath)
	for i := 0; i < 180; i++ {
		sys.StepFrame()
	}
	fb := sys.GetDisplayFramebufferRaw()
	hist, _, dominantCount := frameHistogram(fb)
	// A title screen (or any meaningful frame) needs more than one color
	// and the dominant color should not cover essentially the whole frame.
	if len(hist) < 4 {
		t.Fatalf("framebuffer has only %d distinct colors — likely stuck/blank", len(hist))
	}
	if dominantCount > len(fb)*98/100 {
		t.Fatalf("framebuffer ≥98%% one color (%d / %d) — game not rendering", dominantCount, len(fb))
	}
	t.Logf("distinct colors=%d, dominant pixel count=%d/%d", len(hist), dominantCount, len(fb))
}
