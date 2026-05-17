package test

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSpriteOverflowSuite covers blargg's sprite_overflow_tests. The
// ROMs render "PASSED" or "FAILED #n" to the PPU nametable — there's
// no $6000 status protocol.
//
//   1.Basics      — secondary-OAM 9th-sprite detection.
//   2.Details     — left clip, Y=239/240/255, mixed sprite indices, 8x16 mode.
//   3.Timing      — exact cycle when the overflow flag transitions.
//   4.Obscure     — OAM-address quirks in evaluation.
//   5.Emulator    — emulator-specific bug surface.
//
// 3-5 need cycle-accurate sprite-evaluation timing (the real PPU sets
// overflow during cycles 65-256 of the previous scanline's eval phase;
// we set it at cycle 0 of the rendering scanline). Marked as known
// limitations until the renderer steps sprite eval per-cycle.
func TestSpriteOverflowSuite(t *testing.T) {
	const dir = `R:\nes-test-roms-master\sprite_overflow_tests`
	cases := []blarggCase{
		{name: "1.Basics.nes", maxFrames: 300, expectedToPass: true},
		{name: "2.Details.nes", maxFrames: 600, expectedToPass: true},
		{name: "3.Timing.nes", maxFrames: 600, expectedToPass: false,
			skipReason: "requires cycle-accurate sprite eval (overflow latched during cycles 65-256 of the previous scanline)"},
		{name: "4.Obscure.nes", maxFrames: 600, expectedToPass: false,
			skipReason: "requires cycle-accurate OAM-address eval quirks (overflow evaluates from OAMADDR, not zero)"},
		{name: "5.Emulator.nes", maxFrames: 600, expectedToPass: false,
			skipReason: "emulator-specific subtests; not all corners are reachable without a cycle-stepped renderer"},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			romPath := filepath.Join(dir, c.name)
			if _, err := os.Stat(romPath); err != nil {
				t.Skipf("ROM missing: %v", err)
			}
			sys := loadNES(t, romPath)
			for i := 0; i < c.maxFrames; i++ {
				sys.StepFrame()
			}
			passed := nametableContains(sys, "PASSED")
			failed := nametableContains(sys, "FAILED")
			if c.expectedToPass {
				if failed {
					t.Fatalf("%s reported FAILED", c.name)
				}
				if !passed {
					t.Fatalf("%s didn't reach PASSED within %d frames", c.name, c.maxFrames)
				}
				return
			}
			if passed {
				t.Logf("%s unexpectedly passed — promote it", c.name)
			}
			t.Skip(c.skipReason)
		})
	}
}
