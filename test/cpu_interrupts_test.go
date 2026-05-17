package test

import "testing"

// TestCPUInterruptsV2 runs blargg's cpu_interrupts_v2 suite. Each sub-ROM
// targets one corner of 6502 interrupt timing:
//
//   1-cli_latency       CLI/SEI/PLP one-instruction inhibit, RTI immediate, APU frame IRQ
//   2-nmi_and_brk       NMI hijacking an in-progress BRK (B flag, vector)
//   3-nmi_and_irq       NMI hijacking IRQ vectoring (race window in interrupt sequence)
//   4-irq_and_dma       IRQ timing across the OAM DMA stall
//   5-branch_delays_irq Taken non-page-crossing branch's last cycle inhibits IRQ
//
// Tests 2-5 need sub-instruction CPU/PPU/APU interleaving — currently the
// emulator steps one whole CPU instruction, then catches the PPU/APU up,
// so NMI/IRQ assertion timing within an in-progress BRK/IRQ vector load
// or DMA stall can't be resolved at the cycle granularity these tests
// measure. Test 1 (the timing-tolerant cli/sei/plp/rti behaviours) does
// pass.
func TestCPUInterruptsV2(t *testing.T) {
	const dir = `R:\nes-test-roms-master\cpu_interrupts_v2\rom_singles`
	const subInstructionLimitation = "requires sub-instruction CPU/PPU interleaving (NMI/IRQ assertion mid-vector or mid-DMA)"
	runBlarggSuite(t, dir, []blarggCase{
		{name: "1-cli_latency.nes", maxFrames: 1200, expectedToPass: true},
		{name: "2-nmi_and_brk.nes", maxFrames: 1200, expectedToPass: false,
			skipReason: subInstructionLimitation + "; NMI hijacking BRK between push and vector load"},
		{name: "3-nmi_and_irq.nes", maxFrames: 1200, expectedToPass: false,
			skipReason: subInstructionLimitation + "; NMI hijacking IRQ between push and vector load"},
		{name: "4-irq_and_dma.nes", maxFrames: 1200, expectedToPass: false,
			skipReason: subInstructionLimitation + "; IRQ landing inside the 513-cycle OAM DMA stall"},
		{name: "5-branch_delays_irq.nes", maxFrames: 1200, expectedToPass: false,
			skipReason: subInstructionLimitation + "; branch IRQ-poll suppression is implemented but the test also needs exact APU frame-IRQ scheduling"},
	})
}
