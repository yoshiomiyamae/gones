package test

import (
	"os"
	"testing"
)

// TestCPUTimingTest6 runs Shay Green's cpu_timing_test6 — full CPU
// instruction-timing coverage for every official and unofficial 6502
// opcode (excluding branches and halts). The test takes ~16 s on real
// hardware and reports the result visually (no $6000 status protocol),
// so we run it for ~1500 frames and inspect the PPU nametable: tile
// indices match ASCII, so "PASSED" / "FAIL OP" lives as those bytes in
// VRAM $2000-$23FF.
func TestCPUTimingTest6(t *testing.T) {
	const romPath = `R:\nes-test-roms-master\cpu_timing_test6\cpu_timing_test.nes`
	if _, err := os.Stat(romPath); err != nil {
		t.Skipf("ROM missing: %v", err)
	}
	sys := loadNES(t, romPath)
	for i := 0; i < 1500; i++ {
		sys.StepFrame()
	}
	if nametableContains(sys, "FAIL OP") {
		t.Fatalf("test reported FAIL OP — see screenshot for the failing opcode and cycle count")
	}
	if !nametableContains(sys, "PASSED") {
		t.Fatalf("test didn't report PASSED within 1500 frames — likely hung or failed silently")
	}
}
