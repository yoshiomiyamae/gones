package cpu

import (
	"fmt"
	"testing"
)

// Test illegal/undocumented 6502 instructions
func TestIllegalInstructions(t *testing.T) {
	t.Run("LAX_LoadAAndX", func(t *testing.T) {
		// Test LAX absolute
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.Memory.Write(0x0200, 0xAF) // LAX abs
		cpu.Memory.Write(0x0201, 0x00)
		cpu.Memory.Write(0x0202, 0x18)
		cpu.Memory.Write(0x1800, 0x42)
		
		cycles := cpu.Step()
		
		if cpu.A != 0x42 {
			t.Errorf("Expected A=42, got A=%02X", cpu.A)
		}
		if cpu.X != 0x42 {
			t.Errorf("Expected X=42, got X=%02X", cpu.X)
		}
		if cycles != 4 {
			t.Errorf("Expected 4 cycles for LAX abs, got %d", cycles)
		}
		
		// Test LAX zeropage,Y
		cpu = createTestCPU()
		cpu.PC = 0x0200
		cpu.Y = 0x02
		cpu.Memory.Write(0x0200, 0xB7) // LAX zp,Y
		cpu.Memory.Write(0x0201, 0x10)
		cpu.Memory.Write(0x12, 0x80)
		
		cycles = cpu.Step()
		
		if cpu.A != 0x80 {
			t.Errorf("Expected A=80, got A=%02X", cpu.A)
		}
		if cpu.X != 0x80 {
			t.Errorf("Expected X=80, got X=%02X", cpu.X)
		}
		if !cpu.getFlag(FlagNegative) {
			t.Error("Negative flag should be set")
		}
		if cycles != 4 {
			t.Errorf("Expected 4 cycles for LAX zp,Y, got %d", cycles)
		}
		
		// Test LAX (zp,X)
		cpu = createTestCPU()
		cpu.PC = 0x0200
		cpu.X = 0x03
		cpu.Memory.Write(0x0200, 0xA3) // LAX (zp,X)
		cpu.Memory.Write(0x0201, 0x20)
		cpu.Memory.Write(0x23, 0x00) // Target address low
		cpu.Memory.Write(0x24, 0x19) // Target address high
		cpu.Memory.Write(0x1900, 0x00)
		
		cycles = cpu.Step()
		
		if cpu.A != 0x00 {
			t.Errorf("Expected A=00, got A=%02X", cpu.A)
		}
		if cpu.X != 0x00 {
			t.Errorf("Expected X=00, got X=%02X", cpu.X)
		}
		if !cpu.getFlag(FlagZero) {
			t.Error("Zero flag should be set")
		}
		if cycles != 6 {
			t.Errorf("Expected 6 cycles for LAX (zp,X), got %d", cycles)
		}
		
		// Test LAX (zp),Y
		cpu = createTestCPU()
		cpu.PC = 0x0200
		cpu.Y = 0x01
		cpu.Memory.Write(0x0200, 0xB3) // LAX (zp),Y
		cpu.Memory.Write(0x0201, 0x30)
		cpu.Memory.Write(0x30, 0xFF) // Base address low
		cpu.Memory.Write(0x31, 0x0F) // Base address high (0x0FFF)
		cpu.Memory.Write(0x1000, 0x33) // 0x0FFF + 1 = 0x1000 (page crossing)
		
		cycles = cpu.Step()
		
		if cpu.A != 0x33 {
			t.Errorf("Expected A=33, got A=%02X", cpu.A)
		}
		if cpu.X != 0x33 {
			t.Errorf("Expected X=33, got X=%02X", cpu.X)
		}
		if cycles != 6 { // Page crossing adds cycle
			t.Errorf("Expected 6 cycles for LAX (zp),Y with page crossing, got %d", cycles)
		}
	})
	
	t.Run("SAX_StoreAAndX", func(t *testing.T) {
		// Test SAX zeropage
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.A = 0xFF
		cpu.X = 0x0F
		cpu.Memory.Write(0x0200, 0x87) // SAX zp
		cpu.Memory.Write(0x0201, 0x10)
		
		cycles := cpu.Step()
		
		expectedValue := uint8(0xFF & 0x0F) // A AND X
		if cpu.Memory.Read(0x10) != expectedValue {
			t.Errorf("Expected memory[0x10]=%02X, got %02X", expectedValue, cpu.Memory.Read(0x10))
		}
		if cycles != 3 {
			t.Errorf("Expected 3 cycles for SAX zp, got %d", cycles)
		}
		
		// Test SAX zeropage,Y
		cpu = createTestCPU()
		cpu.PC = 0x0200
		cpu.A = 0xAA
		cpu.X = 0x55
		cpu.Y = 0x02
		cpu.Memory.Write(0x0200, 0x97) // SAX zp,Y
		cpu.Memory.Write(0x0201, 0x20)
		
		cycles = cpu.Step()
		
		expectedValue = uint8(0xAA & 0x55) // A AND X = 0x00
		if cpu.Memory.Read(0x22) != expectedValue {
			t.Errorf("Expected memory[0x22]=%02X, got %02X", expectedValue, cpu.Memory.Read(0x22))
		}
		if cycles != 4 {
			t.Errorf("Expected 4 cycles for SAX zp,Y, got %d", cycles)
		}
		
		// Test SAX absolute
		cpu = createTestCPU()
		cpu.PC = 0x0200
		cpu.A = 0xF0
		cpu.X = 0x0F
		cpu.Memory.Write(0x0200, 0x8F) // SAX abs
		cpu.Memory.Write(0x0201, 0x00)
		cpu.Memory.Write(0x0202, 0x18)
		
		cycles = cpu.Step()
		
		expectedValue = uint8(0xF0 & 0x0F) // A AND X = 0x00
		if cpu.Memory.Read(0x1800) != expectedValue {
			t.Errorf("Expected memory[0x1800]=%02X, got %02X", expectedValue, cpu.Memory.Read(0x1800))
		}
		if cycles != 4 {
			t.Errorf("Expected 4 cycles for SAX abs, got %d", cycles)
		}
		
		// Test SAX (zp,X)
		cpu = createTestCPU()
		cpu.PC = 0x0200
		cpu.A = 0xCC
		cpu.X = 0x33
		cpu.Memory.Write(0x0200, 0x83) // SAX (zp,X)
		cpu.Memory.Write(0x0201, 0x10)
		cpu.Memory.Write(0x43, 0x00) // Target address low (0x10 + 0x33 = 0x43)
		cpu.Memory.Write(0x44, 0x19) // Target address high
		
		cycles = cpu.Step()
		
		expectedValue = uint8(0xCC & 0x33) // A AND X = 0x00
		if cpu.Memory.Read(0x1900) != expectedValue {
			t.Errorf("Expected memory[0x1900]=%02X, got %02X", expectedValue, cpu.Memory.Read(0x1900))
		}
		if cycles != 6 {
			t.Errorf("Expected 6 cycles for SAX (zp,X), got %d", cycles)
		}
	})
}

// Test illegal NOP instructions
func TestIllegalNOPs(t *testing.T) {
	t.Run("Illegal_NOP_Variants", func(t *testing.T) {
		testCases := []struct {
			name     string
			opcode   uint8
			cycles   int
			pcAdvance int
		}{
			{"NOP_1A", 0x1A, 2, 1}, // Implied
			{"NOP_3A", 0x3A, 2, 1}, // Implied
			{"NOP_5A", 0x5A, 2, 1}, // Implied
			{"NOP_7A", 0x7A, 2, 1}, // Implied
			{"NOP_DA", 0xDA, 2, 1}, // Implied
			{"NOP_FA", 0xFA, 2, 1}, // Implied
			{"NOP_80", 0x80, 2, 2}, // Immediate
			{"NOP_82", 0x82, 2, 2}, // Immediate
			{"NOP_89", 0x89, 2, 2}, // Immediate
			{"NOP_C2", 0xC2, 2, 2}, // Immediate
			{"NOP_E2", 0xE2, 2, 2}, // Immediate
			{"NOP_04", 0x04, 3, 2}, // Zero page
			{"NOP_44", 0x44, 3, 2}, // Zero page
			{"NOP_64", 0x64, 3, 2}, // Zero page
			{"NOP_14", 0x14, 4, 2}, // Zero page,X
			{"NOP_34", 0x34, 4, 2}, // Zero page,X
			{"NOP_54", 0x54, 4, 2}, // Zero page,X
			{"NOP_74", 0x74, 4, 2}, // Zero page,X
			{"NOP_D4", 0xD4, 4, 2}, // Zero page,X
			{"NOP_F4", 0xF4, 4, 2}, // Zero page,X
			{"NOP_0C", 0x0C, 4, 3}, // Absolute
			{"NOP_1C", 0x1C, 4, 3}, // Absolute,X (simplified, no page crossing)
			{"NOP_3C", 0x3C, 4, 3}, // Absolute,X
			{"NOP_5C", 0x5C, 4, 3}, // Absolute,X
			{"NOP_7C", 0x7C, 4, 3}, // Absolute,X
			{"NOP_DC", 0xDC, 4, 3}, // Absolute,X
			{"NOP_FC", 0xFC, 4, 3}, // Absolute,X
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cpu := createTestCPU()
				cpu.PC = 0x0200
				cpu.Memory.Write(0x0200, tc.opcode)
				cpu.Memory.Write(0x0201, 0x42) // Operand for immediate/zp
				cpu.Memory.Write(0x0202, 0x30) // High byte for absolute
				
				originalA := cpu.A
				originalX := cpu.X
				originalY := cpu.Y
				originalP := cpu.P
				originalSP := cpu.SP
				
				cycles := cpu.Step()
				
				// Illegal NOPs should not change any registers or flags
				if cpu.A != originalA || cpu.X != originalX || cpu.Y != originalY {
					t.Errorf("Illegal NOP changed registers: A=%02X->%02X, X=%02X->%02X, Y=%02X->%02X",
						originalA, cpu.A, originalX, cpu.X, originalY, cpu.Y)
				}
				if cpu.P != originalP {
					t.Errorf("Illegal NOP changed flags: P=%02X->%02X", originalP, cpu.P)
				}
				if cpu.SP != originalSP {
					t.Errorf("Illegal NOP changed stack pointer: SP=%02X->%02X", originalSP, cpu.SP)
				}
				
				expectedPC := uint16(0x0200 + tc.pcAdvance)
				if cpu.PC != expectedPC {
					t.Errorf("Expected PC=%04X, got PC=%04X", expectedPC, cpu.PC)
				}
				if cycles != tc.cycles {
					t.Errorf("Expected %d cycles, got %d", tc.cycles, cycles)
				}
			})
		}
	})
}

// Test behavior of completely undefined opcodes
func TestUndefinedOpcodes(t *testing.T) {
	t.Run("Undefined_Opcodes_Behavior", func(t *testing.T) {
		// Test some undefined opcodes that might cause different behavior
		undefinedOpcodes := []uint8{
			0x02, 0x12, 0x22, 0x32, 0x42, 0x52, 0x62, 0x72,
			0x92, 0xB2, 0xD2, 0xF2,
		}
		
		for _, opcode := range undefinedOpcodes {
			t.Run(fmt.Sprintf("Opcode_0x%02X", opcode), func(t *testing.T) {
				cpu := createTestCPU()
				cpu.PC = 0x0200
				cpu.Memory.Write(0x0200, opcode)
				
				// Store original state
				originalA := cpu.A
				originalX := cpu.X
				originalY := cpu.Y
				originalP := cpu.P
				originalSP := cpu.SP
				originalPC := cpu.PC
				
				// Execute the undefined opcode
				cycles := cpu.Step()
				
				// Document the behavior for regression testing
				t.Logf("Opcode 0x%02X: PC=%04X->%04X, A=%02X->%02X, X=%02X->%02X, Y=%02X->%02X, P=%02X->%02X, SP=%02X->%02X, cycles=%d",
					opcode,
					originalPC, cpu.PC,
					originalA, cpu.A,
					originalX, cpu.X,
					originalY, cpu.Y,
					originalP, cpu.P,
					originalSP, cpu.SP,
					cycles)
				
				// At minimum, PC should advance
				if cpu.PC == originalPC {
					t.Errorf("PC did not advance for undefined opcode 0x%02X", opcode)
				}
			})
		}
	})
}

// Test some additional illegal instructions that have specific behaviors
func TestAdditionalIllegalInstructions(t *testing.T) {
	t.Run("DCP_DecrementAndCompare", func(t *testing.T) {
		// DCP (also known as DCM) decrements memory then compares with A
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.A = 0x10
		cpu.Memory.Write(0x0200, 0xC7) // DCP zp (if implemented)
		cpu.Memory.Write(0x0201, 0x10)
		cpu.Memory.Write(0x10, 0x11)
		
		// This test documents expected behavior if DCP is implemented
		// If not implemented, it should be treated as undefined
		cycles := cpu.Step()
		
		// Expected: memory[0x10] = 0x10 (decremented), then compare with A
		// A (0x10) == memory (0x10), so Z=1, C=1
		t.Logf("DCP test executed with %d cycles", cycles)
	})
	
	t.Run("ISC_IncrementAndSubtract", func(t *testing.T) {
		// ISC (also known as ISB) increments memory then subtracts from A
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.A = 0x20
		cpu.setFlag(FlagCarry, true) // No borrow
		cpu.Memory.Write(0x0200, 0xE7) // ISC zp (if implemented)
		cpu.Memory.Write(0x0201, 0x10)
		cpu.Memory.Write(0x10, 0x0F)
		
		cycles := cpu.Step()
		
		// Expected: memory[0x10] = 0x10 (incremented), then A = A - memory
		// A = 0x20 - 0x10 = 0x10
		t.Logf("ISC test executed with %d cycles", cycles)
	})
	
	t.Run("SLO_ShiftLeftAndOr", func(t *testing.T) {
		// SLO (also known as ASO) shifts memory left then ORs with A
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.A = 0x0F
		cpu.Memory.Write(0x0200, 0x07) // SLO zp (if implemented)
		cpu.Memory.Write(0x0201, 0x10)
		cpu.Memory.Write(0x10, 0x40)
		
		cycles := cpu.Step()
		
		// Expected: memory[0x10] = 0x80 (shifted left), then A = A | memory
		// A = 0x0F | 0x80 = 0x8F
		t.Logf("SLO test executed with %d cycles", cycles)
	})
	
	t.Run("RLA_RotateLeftAndAnd", func(t *testing.T) {
		// RLA rotates memory left then ANDs with A
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.A = 0xFF
		cpu.setFlag(FlagCarry, false)
		cpu.Memory.Write(0x0200, 0x27) // RLA zp (if implemented)
		cpu.Memory.Write(0x0201, 0x10)
		cpu.Memory.Write(0x10, 0x81)
		
		cycles := cpu.Step()
		
		// Expected: memory[0x10] = 0x02 (rotated left), then A = A & memory
		// A = 0xFF & 0x02 = 0x02
		t.Logf("RLA test executed with %d cycles", cycles)
	})
	
	t.Run("SRE_ShiftRightAndEor", func(t *testing.T) {
		// SRE (also known as LSE) shifts memory right then EORs with A
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.A = 0xFF
		cpu.Memory.Write(0x0200, 0x47) // SRE zp (if implemented)
		cpu.Memory.Write(0x0201, 0x10)
		cpu.Memory.Write(0x10, 0x81)
		
		cycles := cpu.Step()
		
		// Expected: memory[0x10] = 0x40 (shifted right), then A = A ^ memory
		// A = 0xFF ^ 0x40 = 0xBF
		t.Logf("SRE test executed with %d cycles", cycles)
	})
	
	t.Run("RRA_RotateRightAndAdd", func(t *testing.T) {
		// RRA rotates memory right then adds to A
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.A = 0x10
		cpu.setFlag(FlagCarry, true)
		cpu.Memory.Write(0x0200, 0x67) // RRA zp (if implemented)
		cpu.Memory.Write(0x0201, 0x10)
		cpu.Memory.Write(0x10, 0x02)
		
		cycles := cpu.Step()
		
		// Expected: memory[0x10] = 0x81 (rotated right with carry), then A = A + memory
		// A = 0x10 + 0x81 = 0x91
		t.Logf("RRA test executed with %d cycles", cycles)
	})
}