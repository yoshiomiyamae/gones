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
		// Verifies that RMW opcodes (ASL/LSR/ROL/ROR/INC/DEC and undocumented
		// SLO/SRE/RLA/RRA/ISB/DCP) emit the dummy write before the modified
		// write; also exercises PPU open-bus and $2007 buffering corners.
		{name: `R:\nes-test-roms-master\cpu_dummy_writes\cpu_dummy_writes_ppumem.nes`, maxFrames: 1800, expectedToPass: true},
		// CPU must be able to execute code from PPU I/O ($2001) via every
		// transfer opcode (JSR, JMP, RTS, JMP+RTI, BRK). Also verifies that
		// one-byte opcodes (RTS/RTI/BRK) issue a dummy fetch of the byte
		// that follows the opcode.
		{name: `R:\nes-test-roms-master\cpu_exec_space\test_cpu_exec_space_ppuio.nes`, maxFrames: 1800, expectedToPass: true},
		// CPU must be able to execute code from APU I/O space ($4000-$401F);
		// also that write-only APU ports and unallocated $4018..$40FF return
		// open-bus.
		{name: `R:\nes-test-roms-master\cpu_exec_space\test_cpu_exec_space_apu.nes`, maxFrames: 1800, expectedToPass: true},
	}
	// These ROMs are addressed by absolute path; pass an empty base so
	// filepath.Join leaves them unchanged.
	runBlarggSuite(t, "", cases)
}
