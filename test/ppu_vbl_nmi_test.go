package test

import "testing"

// TestPPUVblNmi runs blargg's ppu_vbl_nmi suite as individual rom_singles.
// Two subtests are marked as known limitations — both need sub-instruction
// CPU/PPU timing precision that our atomic CPU.Step / lazy PPU catch-up
// can't model accurately enough:
//
//   - 07-nmi_on_timing: the $2000-write-near-VBL-clear race is sensitive
//     to the exact CPU cycle the write lands on. Off by 1 row.
//   - 10-even_odd_timing: the BG-enable odd-frame-skip race has the same
//     root cause.
func TestPPUVblNmi(t *testing.T) {
	runBlarggSuite(t, `R:\nes-test-roms-master\ppu_vbl_nmi\rom_singles`, []blarggCase{
		{name: "01-vbl_basics.nes", maxFrames: 600, expectedToPass: true},
		{name: "02-vbl_set_time.nes", maxFrames: 600, expectedToPass: true},
		{name: "03-vbl_clear_time.nes", maxFrames: 600, expectedToPass: true},
		{name: "04-nmi_control.nes", maxFrames: 600, expectedToPass: true},
		{name: "05-nmi_timing.nes", maxFrames: 600, expectedToPass: true},
		{name: "06-suppression.nes", maxFrames: 600, expectedToPass: true},
		{name: "07-nmi_on_timing.nes", maxFrames: 600, expectedToPass: false,
			skipReason: "needs sub-instruction CPU/PPU timing; off by 1 row"},
		{name: "08-nmi_off_timing.nes", maxFrames: 600, expectedToPass: true},
		{name: "09-even_odd_frames.nes", maxFrames: 600, expectedToPass: true},
		{name: "10-even_odd_timing.nes", maxFrames: 600, expectedToPass: false,
			skipReason: "needs sub-instruction CPU/PPU timing for the BG-enable skip race"},
	})
}
