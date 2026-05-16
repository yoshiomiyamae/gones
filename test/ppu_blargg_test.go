package test

import "testing"

// TestPPUBlarggROMs runs the PPU-focused blargg/bisqwit test ROMs that
// share the $6000 status / $6004 text protocol. Each ROM lives in its
// own subdirectory so we pass the full path per case.
func TestPPUBlarggROMs(t *testing.T) {
	cases := []blarggCase{
		// PPU "decay register" — writes refresh all 8 bits; reads only
		// refresh the bits the PPU drives.
		{name: `R:\nes-test-roms-master\ppu_open_bus\ppu_open_bus.nes`, maxFrames: 600, expectedToPass: true},
		// ~70 subtests covering every corner of $2007 read buffering,
		// nametable / palette mirroring, sprite-0 hit + $4014 DMA.
		{name: `R:\nes-test-roms-master\ppu_read_buffer\test_ppu_read_buffer.nes`, maxFrames: 2400, expectedToPass: true},
	}
	// These ROMs are addressed by absolute path; pass an empty base so
	// filepath.Join leaves them unchanged.
	runBlarggSuite(t, "", cases)
}
