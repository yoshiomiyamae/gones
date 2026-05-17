package test

import "testing"

const blarggCPUResetDir = `R:\nes-test-roms-master\cpu_reset`

// TestCPUResetSuite runs blargg's cpu_reset ROMs. The runner handles
// the $81 ("press reset") status by calling NES.SoftReset on the rising
// edge.
//
//   registers       At power: A,X,Y=0, P=$34, S=$FD; after reset:
//                   A,X,Y unchanged, I forced set, S decremented by 3.
func TestCPUResetSuite(t *testing.T) {
	runBlarggSuite(t, blarggCPUResetDir, []blarggCase{
		{name: "registers.nes", maxFrames: 3600, expectedToPass: true},
	})
}
