package test

import (
	"testing"

	"github.com/yoshiomiyamaegones/pkg/nes"
)

// TestNESSystemInitialization tests that all components initialize correctly
func TestNESSystemInitialization(t *testing.T) {
	// Create NES system
	system := nes.NewNES()

	// Verify CPU is initialized
	if system.CPU == nil {
		t.Fatal("CPU should be initialized")
	}

	// Verify PPU is initialized
	if system.PPU == nil {
		t.Fatal("PPU should be initialized")
	}

	// Verify APU is initialized
	if system.APU == nil {
		t.Fatal("APU should be initialized")
	}

	// Verify memory is initialized
	if system.Memory == nil {
		t.Fatal("Memory should be initialized")
	}

	// Check initial CPU state (PC reads from reset vector which is initially 0x0000)
	if system.CPU.PC != 0x0000 {
		t.Errorf("Expected initial PC=0000, got PC=%04X", system.CPU.PC)
	}

	// Check PPU initial state
	if system.PPU.Cycle != 0 {
		t.Errorf("Expected initial PPU cycle=0, got %d", system.PPU.Cycle)
	}

	// Check APU initial state
	if system.APU.Cycles != 0 {
		t.Errorf("Expected initial APU cycle=0, got %d", system.APU.Cycles)
	}
}

// TestCPUPPUCommunication tests CPU writing to PPU registers
func TestCPUPPUCommunication(t *testing.T) {
	system := nes.NewNES()

	// Test PPUCTRL write (0x2000)
	system.Memory.Write(0x2000, 0x80) // Enable NMI

	// Test PPUMASK write (0x2001)
	system.Memory.Write(0x2001, 0x1E) // Enable background and sprites

	// Test PPUADDR writes (0x2006)
	system.Memory.Write(0x2006, 0x20) // High byte
	system.Memory.Write(0x2006, 0x00) // Low byte

	// Test PPUDATA write (0x2007)
	system.Memory.Write(0x2007, 0x42) // Write data to VRAM

	// Verify PPU received the data
	// Note: This would require exposing PPU internal state for verification
	// For now, we just verify no crashes occurred
}

// TestCPUAPUCommunication tests CPU writing to APU registers
func TestCPUAPUCommunication(t *testing.T) {
	system := nes.NewNES()

	// Test pulse channel 1 writes
	system.Memory.Write(0x4000, 0x3F) // Duty cycle and volume
	system.Memory.Write(0x4001, 0x08) // Sweep settings
	system.Memory.Write(0x4002, 0x55) // Timer low
	system.Memory.Write(0x4003, 0x02) // Timer high and length

	// Test triangle channel writes
	system.Memory.Write(0x4008, 0x81) // Linear counter
	system.Memory.Write(0x400A, 0xAA) // Timer low
	system.Memory.Write(0x400B, 0x03) // Timer high and length

	// Test APU status write
	system.Memory.Write(0x4015, 0x0F) // Enable all channels

	// Verify APU channels are enabled
	// This would require checking internal APU state
}

// TestMemoryMapping tests the complete memory mapping system
func TestMemoryMapping(t *testing.T) {
	system := nes.NewNES()

	// Test RAM mirroring (0x0000-0x1FFF)
	system.Memory.Write(0x0000, 0x42)
	if system.Memory.Read(0x0800) != 0x42 {
		t.Error("RAM mirroring failed at 0x0800")
	}
	if system.Memory.Read(0x1000) != 0x42 {
		t.Error("RAM mirroring failed at 0x1000")
	}
	if system.Memory.Read(0x1800) != 0x42 {
		t.Error("RAM mirroring failed at 0x1800")
	}

	// Test PPU register mirroring (0x2000-0x3FFF)
	// Note: PPU registers are write-only for PPUCTRL, so we skip this test
	// The mirroring works but reading PPUCTRL doesn't return the written value

	// Test cartridge ROM area (0x8000-0xFFFF)
	// Note: Without a cartridge loaded, writes to ROM area are ignored
	// This is correct behavior - ROM areas should only be writable via cartridge interface
}

// TestSystemReset tests that system reset works correctly
func TestSystemReset(t *testing.T) {
	system := nes.NewNES()

	// Modify system state
	system.CPU.A = 0xFF
	system.CPU.X = 0xFF
	system.CPU.Y = 0xFF
	system.CPU.PC = 0x1234

	// Reset system
	system.Reset()

	// Verify CPU was reset
	if system.CPU.A != 0x00 {
		t.Errorf("Expected A=00 after reset, got A=%02X", system.CPU.A)
	}
	if system.CPU.X != 0x00 {
		t.Errorf("Expected X=00 after reset, got X=%02X", system.CPU.X)
	}
	if system.CPU.Y != 0x00 {
		t.Errorf("Expected Y=00 after reset, got Y=%02X", system.CPU.Y)
	}
	if system.CPU.PC != 0x0000 {
		t.Errorf("Expected PC=0000 after reset, got PC=%04X", system.CPU.PC)
	}
}

// TestCPUExecutionIntegration tests CPU executing a simple program in RAM
func TestCPUExecutionIntegration(t *testing.T) {
	system := nes.NewNES()

	// Load a simple test program into RAM (zero page area)
	program := []uint8{
		0xA9, 0x42, // LDA #$42    - Load test value
		0x85, 0x10, // STA $10     - Store in zero page
		0xA5, 0x10, // LDA $10     - Load back from zero page
		0xC9, 0x42, // CMP #$42    - Compare with original value
		0xEA, // NOP         - End program
	}

	// Load program into RAM starting at 0x0200
	for i, byte := range program {
		system.Memory.Write(uint16(0x0200+i), byte)
	}

	// Set PC to start of program
	system.CPU.PC = 0x0200

	// Execute program step by step
	maxSteps := 10
	for i := 0; i < maxSteps; i++ {
		if system.CPU.PC == 0x0208 { // NOP instruction address
			break
		}
		system.CPU.Step()
	}

	// Verify program executed correctly
	if system.CPU.A != 0x42 {
		t.Errorf("Expected A=42 after program execution, got A=%02X", system.CPU.A)
	}

	// Verify zero page was written
	if system.Memory.Read(0x0010) != 0x42 {
		t.Errorf("Expected zero page value=42, got %02X", system.Memory.Read(0x0010))
	}

	// Verify flags are correct (Zero flag should be set after CMP)
	if !system.CPU.GetFlag(0x02) { // FlagZero
		t.Error("Zero flag should be set after successful comparison")
	}
}

// TestPPUAPUTiming tests basic timing coordination
func TestPPUAPUTiming(t *testing.T) {
	system := nes.NewNES()

	initialPPUCycle := system.PPU.Cycle
	initialAPUCycle := system.APU.Cycles

	// Step system multiple times
	for i := 0; i < 100; i++ {
		system.Step()
	}

	// Verify PPU and APU cycles advanced
	if system.PPU.Cycle <= initialPPUCycle {
		t.Error("PPU cycle should have advanced")
	}

	if system.APU.Cycles <= initialAPUCycle {
		t.Error("APU cycle should have advanced")
	}

	// PPU should run 3x faster than CPU
	// APU should run at CPU speed
	// This is a basic sanity check
}

// TestInterruptHandling tests basic NMI interrupt mechanism
func TestInterruptHandling(t *testing.T) {
	system := nes.NewNES()

	// Note: Without cartridge, interrupt vectors are 0x0000
	// This test verifies the interrupt mechanism itself

	// Set CPU to a known state
	system.CPU.PC = 0x0200
	originalSP := system.CPU.SP

	// Put NOP at interrupt vector location (0x0000)
	system.Memory.Write(0x0000, 0xEA) // NOP

	// Step CPU once to handle the NMI
	system.CPU.TriggerNMI()
	cycles := system.CPU.Step()

	// Verify NMI was handled (should take 7 cycles)
	if cycles != 7 {
		t.Errorf("Expected 7 cycles for NMI, got %d", cycles)
	}

	// Verify PC changed to NMI vector (0x0000 without cartridge)
	if system.CPU.PC != 0x0000 {
		t.Errorf("Expected PC=0000 after NMI, got PC=%04X", system.CPU.PC)
	}

	// Verify stack was used (return address and status pushed - 3 bytes total)
	if system.CPU.SP != originalSP-3 {
		t.Errorf("Expected SP=%02X after NMI, got SP=%02X", originalSP-3, system.CPU.SP)
	}

	// Verify interrupt flag was set
	if !system.CPU.GetFlag(0x04) { // FlagInterrupt
		t.Error("Interrupt flag should be set after NMI")
	}
}
