package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPPUBlarggROMs runs the PPU-focused blargg/bisqwit test ROMs. Both
// follow the same $6000 status / $6004 text protocol as the MMC3 suite,
// so they share runBlarggTest. Each ROM passes when the status byte
// reaches 0 and the printed text contains "passed".
func TestPPUBlarggROMs(t *testing.T) {
	cases := []struct {
		name      string
		path      string
		maxFrames int
	}{
		// PPU "decay register" — writes refresh all 8 bits; reads only
		// refresh the bits the PPU drives.
		{"ppu_open_bus.nes", `R:\nes-test-roms-master\ppu_open_bus\ppu_open_bus.nes`, 600},
		// ~70 subtests covering every corner of $2007 read buffering,
		// nametable / palette mirroring, sprite-0 hit + $4014 DMA.
		{"test_ppu_read_buffer.nes", `R:\nes-test-roms-master\ppu_read_buffer\test_ppu_read_buffer.nes`, 2400},
	}
	for _, c := range cases {
		c := c
		t.Run(filepath.Base(c.path), func(t *testing.T) {
			if _, err := os.Stat(c.path); err != nil {
				t.Skipf("ROM missing: %v", err)
			}
			status, text, frames := runBlarggTest(t, c.path, c.maxFrames)
			t.Logf("frames=%d status=$%02X output=%q", frames, status, text)
			if status != 0x00 {
				t.Fatalf("test did not pass: status=$%02X output=%q", status, text)
			}
			if !strings.Contains(strings.ToLower(text), "passed") {
				t.Fatalf("missing 'passed' in output: %q", text)
			}
		})
	}
}
