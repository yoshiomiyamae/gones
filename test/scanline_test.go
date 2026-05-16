package test

import (
	"os"
	"testing"
)

// TestScanlineMidWrite runs Quietust's scanline.nes — a visual test that
// renders rows of "******" placeholders on the right side of three test
// areas and expects mid-scanline $2001 / $2000 / $2005 / $2006 writes to
// obscure them. We verify by counting "lit" pixels in the stars column
// (columns 25-30 of the test-area scanlines): correct emulation hides
// most of them; broken emulation leaves the asterisks visible.
func TestScanlineMidWrite(t *testing.T) {
	const romPath = `R:\nes-test-roms-master\scanline\scanline.nes`
	if _, err := os.Stat(romPath); err != nil {
		t.Skipf("ROM missing: %v", err)
	}
	sys := loadNES(t, romPath)
	for i := 0; i < 240; i++ {
		sys.StepFrame()
	}
	fb := sys.GetDisplayFramebufferRaw()

	// Determine the "background" colour as the most common pixel — the
	// rest are foreground (text / stars / patterns).
	hist := map[uint32]int{}
	for _, p := range fb {
		hist[p]++
	}
	var bg uint32
	bestN := 0
	for v, n := range hist {
		if n > bestN {
			bg = v
			bestN = n
		}
	}

	// Each test area covers 6 nametable rows × 8 scanlines = 48 scanlines.
	// First test area starts at row 6 (scanline 48); rows 14-19 are test 2;
	// rows 23-26 are test 3 (only 4 star rows). Stars sit at columns 25-30
	// (pixels 200-247) in real-hardware terms.
	areas := []struct {
		name        string
		yStart, yEnd int
		maxLit      int
	}{
		{"test1 (D3 of $2001)", 48, 95, 100},
		{"test2 (D4 of $2000)", 120, 167, 100},
		{"test3 ($2005/$2006)", 192, 224, 100},
	}
	for _, a := range areas {
		lit := 0
		for y := a.yStart; y < a.yEnd; y++ {
			for x := 200; x < 248; x++ {
				if fb[y*256+x] != bg {
					lit++
				}
			}
		}
		t.Logf("%s lit-pixels in stars column: %d", a.name, lit)
		if lit > a.maxLit {
			t.Errorf("%s: %d lit pixels in stars column (limit %d) — mid-scanline writes not hiding the placeholder asterisks", a.name, lit, a.maxLit)
		}
	}
}
