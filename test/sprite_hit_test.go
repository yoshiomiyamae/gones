package test

import (
	"os"
	"testing"
)

// TestSpriteHitRightEdge runs blargg's sprite_hit_tests_2005.10.05
// 06.right_edge.nes. It checks that sprite 0 hit is suppressed when
// the visible pixel lands on column 255 (NESdev: "obscure
// pixel-output-circuitry reason") but fires normally at column 254.
// Result is rendered to the PPU nametable as "PASSED" / "FAILED #N".
func TestSpriteHitRightEdge(t *testing.T) {
	const romPath = `R:\nes-test-roms-master\sprite_hit_tests_2005.10.05\06.right_edge.nes`
	if _, err := os.Stat(romPath); err != nil {
		t.Skipf("ROM missing: %v", err)
	}
	sys := loadNES(t, romPath)
	for i := 0; i < 300; i++ {
		sys.StepFrame()
	}
	if nametableContains(sys, "FAILED") {
		t.Fatalf("06.right_edge reported FAILED — see screenshot for the failing subtest")
	}
	if !nametableContains(sys, "PASSED") {
		t.Fatalf("06.right_edge didn't reach PASSED within 300 frames — likely hung")
	}
}
