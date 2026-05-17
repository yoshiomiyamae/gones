package test

import (
	"bytes"

	"github.com/yoshiomiyamaegones/pkg/nes"
)

// frameHistogram returns the per-color pixel counts of fb plus the
// dominant color and its count. Shared between the scanline test
// (which needs the dominant color as a "background" baseline) and
// mapper smoke tests (which use the distribution shape as a
// "is the game actually rendering?" heuristic).
func frameHistogram(fb []uint32) (hist map[uint32]int, dominant uint32, dominantCount int) {
	hist = make(map[uint32]int, 16)
	for _, p := range fb {
		hist[p]++
	}
	for v, n := range hist {
		if n > dominantCount {
			dominant = v
			dominantCount = n
		}
	}
	return
}

// nametableContains reports whether nametable 0 ($2000-$23FF) holds the
// given ASCII byte sequence. Used by ROMs that report results visually
// (cpu_timing_test6, sprite_hit, sprite_overflow, dmc_dma) — their
// console code writes character tile indices that match ASCII, so the
// pass/fail token shows up as those bytes in VRAM.
func nametableContains(sys *nes.NES, s string) bool {
	return bytes.Contains(sys.PPU.VRAM[0x2000:0x2400], []byte(s))
}
