package cpu

import (
	"testing"
)

// Test interrupt handling
func TestInterrupts(t *testing.T) {
	t.Run("BRK_Instruction", func(t *testing.T) {
		cpu := createTestCPU()
		cpu.PC = 0x0200
		
		// Set up interrupt vector
		cpu.Memory.Write(0xFFFE, 0x00) // IRQ/BRK vector low
		cpu.Memory.Write(0xFFFF, 0x05) // IRQ/BRK vector high
		
		cpu.Memory.Write(0x0200, 0x00) // BRK
		initialSP := cpu.SP
		
		cycles := cpu.Step()
		
		// BRK should jump to interrupt vector
		if cpu.PC != 0x0500 {
			t.Errorf("Expected PC=0500 after BRK, got PC=%04X", cpu.PC)
		}
		
		// BRK should push PC+2 and status to stack
		if cpu.SP != initialSP-3 {
			t.Errorf("Expected SP=%02X after BRK, got SP=%02X", initialSP-3, cpu.SP)
		}
		
		// Interrupt flag should be set
		if !cpu.getFlag(FlagInterrupt) {
			t.Error("Interrupt flag should be set after BRK")
		}
		
		if cycles != 7 {
			t.Errorf("Expected 7 cycles for BRK, got %d", cycles)
		}
	})
	
	t.Run("RTI_Instruction", func(t *testing.T) {
		cpu := createTestCPU()
		
		// Set up stack with return address and status
		cpu.SP = 0xFC
		cpu.Memory.Write(0x01FD, 0x24) // Status (with Break flag clear)
		cpu.Memory.Write(0x01FE, 0x34) // PC low
		cpu.Memory.Write(0x01FF, 0x12) // PC high
		
		cpu.PC = 0x0500
		cpu.Memory.Write(0x0500, 0x40) // RTI
		
		cycles := cpu.Step()
		
		// RTI should restore PC and status
		if cpu.PC != 0x1234 {
			t.Errorf("Expected PC=1234 after RTI, got PC=%04X", cpu.PC)
		}
		
		if cpu.SP != 0xFF {
			t.Errorf("Expected SP=FF after RTI, got SP=%02X", cpu.SP)
		}
		
		if cpu.P != 0x24 {
			t.Errorf("Expected P=24 after RTI, got P=%02X", cpu.P)
		}
		
		if cycles != 6 {
			t.Errorf("Expected 6 cycles for RTI, got %d", cycles)
		}
	})
	
	t.Run("NMI_Handling", func(t *testing.T) {
		cpu := createTestCPU()
		cpu.PC = 0x0200
		
		// Set up NMI vector
		cpu.Memory.Write(0xFFFA, 0x00) // NMI vector low
		cpu.Memory.Write(0xFFFB, 0x06) // NMI vector high
		
		// Trigger NMI
		cpu.TriggerNMI()
		
		initialSP := cpu.SP
		cycles := cpu.Step()
		
		// NMI should jump to NMI vector
		if cpu.PC != 0x0600 {
			t.Errorf("Expected PC=0600 after NMI, got PC=%04X", cpu.PC)
		}
		
		// NMI should push PC and status to stack
		if cpu.SP != initialSP-3 {
			t.Errorf("Expected SP=%02X after NMI, got SP=%02X", initialSP-3, cpu.SP)
		}
		
		// Interrupt flag should be set
		if !cpu.getFlag(FlagInterrupt) {
			t.Error("Interrupt flag should be set after NMI")
		}
		
		if cycles != 7 {
			t.Errorf("Expected 7 cycles for NMI, got %d", cycles)
		}
	})
}

// Test all addressing modes comprehensively
func TestAddressingModesComplete(t *testing.T) {
	t.Run("IndexedIndirect_X", func(t *testing.T) {
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.X = 0x04
		
		// Set up memory for (zp,X) addressing
		cpu.Memory.Write(0x0200, 0xA1) // LDA (zp,X)
		cpu.Memory.Write(0x0201, 0x20) // zero page base address
		cpu.Memory.Write(0x24, 0x74)   // Target address low (0x20 + 0x04)
		cpu.Memory.Write(0x25, 0x17)   // Target address high
		cpu.Memory.Write(0x1774, 0x42) // Target data (in RAM area)
		
		cycles := cpu.Step()
		
		if cpu.A != 0x42 {
			t.Errorf("Expected A=42, got A=%02X", cpu.A)
		}
		if cycles != 6 {
			t.Errorf("Expected 6 cycles for LDA (zp,X), got %d", cycles)
		}
	})
	
	t.Run("IndirectIndexed_Y", func(t *testing.T) {
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.Y = 0x10
		
		// Set up memory for (zp),Y addressing
		cpu.Memory.Write(0x0200, 0xB1) // LDA (zp),Y
		cpu.Memory.Write(0x0201, 0x86) // zero page address
		cpu.Memory.Write(0x86, 0x28)   // Base address low
		cpu.Memory.Write(0x87, 0x10)   // Base address high (0x1028)
		cpu.Memory.Write(0x1038, 0x55) // Target data (0x1028 + 0x10)
		
		cycles := cpu.Step()
		
		if cpu.A != 0x55 {
			t.Errorf("Expected A=55, got A=%02X", cpu.A)
		}
		if cycles != 5 { // Should be 5 cycles if no page crossing
			t.Errorf("Expected 5 cycles for LDA (zp),Y, got %d", cycles)
		}
	})
	
	t.Run("IndirectIndexed_PageCrossing", func(t *testing.T) {
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.Y = 0xFF
		
		// Set up memory for page crossing
		cpu.Memory.Write(0x0200, 0xB1) // LDA (zp),Y
		cpu.Memory.Write(0x0201, 0x86) // zero page address
		cpu.Memory.Write(0x86, 0x02)   // Base address low
		cpu.Memory.Write(0x87, 0x10)   // Base address high (0x1002)
		cpu.Memory.Write(0x1101, 0x77) // Target data (0x1002 + 0xFF = 0x1101)
		
		cycles := cpu.Step()
		
		if cpu.A != 0x77 {
			t.Errorf("Expected A=77, got A=%02X", cpu.A)
		}
		if cycles != 6 { // Should be 6 cycles with page crossing
			t.Errorf("Expected 6 cycles for LDA (zp),Y with page crossing, got %d", cycles)
		}
	})
}

// Test all stack instructions
func TestStackInstructionsComplete(t *testing.T) {
	t.Run("PHP_PLP", func(t *testing.T) {
		cpu := createTestCPU()
		cpu.PC = 0x0200
		
		// Set specific status flags
		cpu.P = FlagCarry | FlagZero | FlagNegative
		originalSP := cpu.SP
		
		// PHP - Push Processor Status
		cpu.Memory.Write(0x0200, 0x08) // PHP
		cycles := cpu.Step()
		
		if cpu.SP != originalSP-1 {
			t.Errorf("Expected SP=%02X after PHP, got SP=%02X", originalSP-1, cpu.SP)
		}
		if cycles != 3 {
			t.Errorf("Expected 3 cycles for PHP, got %d", cycles)
		}
		
		// Change flags
		cpu.P = FlagOverflow | FlagInterrupt
		
		// PLP - Pull Processor Status
		cpu.PC = 0x0201
		cpu.Memory.Write(0x0201, 0x28) // PLP
		cycles = cpu.Step()
		
		expectedFlags := uint8(FlagCarry | FlagZero | FlagNegative | FlagUnused)
		if cpu.P != expectedFlags {
			t.Errorf("Expected P=%02X after PLP, got P=%02X", expectedFlags, cpu.P)
		}
		if cpu.SP != originalSP {
			t.Errorf("Expected SP=%02X after PLP, got SP=%02X", originalSP, cpu.SP)
		}
		if cycles != 4 {
			t.Errorf("Expected 4 cycles for PLP, got %d", cycles)
		}
	})
}

// Test all transfer instructions
func TestTransferInstructionsComplete(t *testing.T) {
	t.Run("TXS_TSX", func(t *testing.T) {
		cpu := createTestCPU()
		
		// TXS - Transfer X to Stack Pointer
		cpu.PC = 0x0200
		cpu.X = 0x42
		cpu.Memory.Write(0x0200, 0x9A) // TXS
		
		cycles := cpu.Step()
		
		if cpu.SP != 0x42 {
			t.Errorf("Expected SP=42 after TXS, got SP=%02X", cpu.SP)
		}
		if cycles != 2 {
			t.Errorf("Expected 2 cycles for TXS, got %d", cycles)
		}
		// TXS does not affect flags
		
		// TSX - Transfer Stack Pointer to X
		cpu.PC = 0x0201
		cpu.SP = 0x33
		cpu.X = 0x00
		cpu.Memory.Write(0x0201, 0xBA) // TSX
		
		cycles = cpu.Step()
		
		if cpu.X != 0x33 {
			t.Errorf("Expected X=33 after TSX, got X=%02X", cpu.X)
		}
		if cycles != 2 {
			t.Errorf("Expected 2 cycles for TSX, got %d", cycles)
		}
		// TSX affects N and Z flags
	})
	
	t.Run("TAY_TYA", func(t *testing.T) {
		cpu := createTestCPU()
		
		// TAY - Transfer A to Y
		cpu.PC = 0x0200
		cpu.A = 0x80
		cpu.Memory.Write(0x0200, 0xA8) // TAY
		
		cycles := cpu.Step()
		
		if cpu.Y != 0x80 {
			t.Errorf("Expected Y=80 after TAY, got Y=%02X", cpu.Y)
		}
		if !cpu.getFlag(FlagNegative) {
			t.Error("Negative flag should be set after TAY with 0x80")
		}
		if cycles != 2 {
			t.Errorf("Expected 2 cycles for TAY, got %d", cycles)
		}
		
		// TYA - Transfer Y to A
		cpu.PC = 0x0201
		cpu.Y = 0x00
		cpu.A = 0xFF
		cpu.Memory.Write(0x0201, 0x98) // TYA
		
		cycles = cpu.Step()
		
		if cpu.A != 0x00 {
			t.Errorf("Expected A=00 after TYA, got A=%02X", cpu.A)
		}
		if !cpu.getFlag(FlagZero) {
			t.Error("Zero flag should be set after TYA with 0x00")
		}
		if cycles != 2 {
			t.Errorf("Expected 2 cycles for TYA, got %d", cycles)
		}
	})
}

// Test all flag instructions
func TestFlagInstructionsComplete(t *testing.T) {
	t.Run("CLI_SEI", func(t *testing.T) {
		cpu := createTestCPU()
		
		// CLI - Clear Interrupt Flag
		cpu.setFlag(FlagInterrupt, true)
		cpu.PC = 0x0200
		cpu.Memory.Write(0x0200, 0x58) // CLI
		
		cycles := cpu.Step()
		
		if cpu.getFlag(FlagInterrupt) {
			t.Error("Interrupt flag should be cleared after CLI")
		}
		if cycles != 2 {
			t.Errorf("Expected 2 cycles for CLI, got %d", cycles)
		}
		
		// SEI - Set Interrupt Flag
		cpu.PC = 0x0201
		cpu.Memory.Write(0x0201, 0x78) // SEI
		
		cycles = cpu.Step()
		
		if !cpu.getFlag(FlagInterrupt) {
			t.Error("Interrupt flag should be set after SEI")
		}
		if cycles != 2 {
			t.Errorf("Expected 2 cycles for SEI, got %d", cycles)
		}
	})
	
	t.Run("CLV", func(t *testing.T) {
		cpu := createTestCPU()
		
		// CLV - Clear Overflow Flag
		cpu.setFlag(FlagOverflow, true)
		cpu.PC = 0x0200
		cpu.Memory.Write(0x0200, 0xB8) // CLV
		
		cycles := cpu.Step()
		
		if cpu.getFlag(FlagOverflow) {
			t.Error("Overflow flag should be cleared after CLV")
		}
		if cycles != 2 {
			t.Errorf("Expected 2 cycles for CLV, got %d", cycles)
		}
	})
	
	t.Run("CLD_SED", func(t *testing.T) {
		cpu := createTestCPU()
		
		// CLD - Clear Decimal Flag
		cpu.setFlag(FlagDecimal, true)
		cpu.PC = 0x0200
		cpu.Memory.Write(0x0200, 0xD8) // CLD
		
		cycles := cpu.Step()
		
		if cpu.getFlag(FlagDecimal) {
			t.Error("Decimal flag should be cleared after CLD")
		}
		if cycles != 2 {
			t.Errorf("Expected 2 cycles for CLD, got %d", cycles)
		}
		
		// SED - Set Decimal Flag
		cpu.PC = 0x0201
		cpu.Memory.Write(0x0201, 0xF8) // SED
		
		cycles = cpu.Step()
		
		if !cpu.getFlag(FlagDecimal) {
			t.Error("Decimal flag should be set after SED")
		}
		if cycles != 2 {
			t.Errorf("Expected 2 cycles for SED, got %d", cycles)
		}
	})
}

// Test increment/decrement instructions
func TestIncDecComplete(t *testing.T) {
	t.Run("INC_Memory", func(t *testing.T) {
		cpu := createTestCPU()
		cpu.PC = 0x0200
		
		// INC zeropage
		cpu.Memory.Write(0x0200, 0xE6) // INC $10
		cpu.Memory.Write(0x0201, 0x10)
		cpu.Memory.Write(0x10, 0x7F)
		
		cycles := cpu.Step()
		
		if cpu.Memory.Read(0x10) != 0x80 {
			t.Errorf("Expected memory[0x10]=80, got %02X", cpu.Memory.Read(0x10))
		}
		if !cpu.getFlag(FlagNegative) {
			t.Error("Negative flag should be set")
		}
		if cycles != 5 {
			t.Errorf("Expected 5 cycles for INC zeropage, got %d", cycles)
		}
	})
	
	t.Run("DEC_Memory", func(t *testing.T) {
		cpu := createTestCPU()
		cpu.PC = 0x0200
		
		// DEC zeropage
		cpu.Memory.Write(0x0200, 0xC6) // DEC $10
		cpu.Memory.Write(0x0201, 0x10)
		cpu.Memory.Write(0x10, 0x01)
		
		cycles := cpu.Step()
		
		if cpu.Memory.Read(0x10) != 0x00 {
			t.Errorf("Expected memory[0x10]=00, got %02X", cpu.Memory.Read(0x10))
		}
		if !cpu.getFlag(FlagZero) {
			t.Error("Zero flag should be set")
		}
		if cycles != 5 {
			t.Errorf("Expected 5 cycles for DEC zeropage, got %d", cycles)
		}
	})
	
	t.Run("INX_Overflow", func(t *testing.T) {
		cpu := createTestCPU()
		cpu.X = 0xFF
		cpu.PC = 0x0200
		cpu.Memory.Write(0x0200, 0xE8) // INX
		
		cycles := cpu.Step()
		
		if cpu.X != 0x00 {
			t.Errorf("Expected X=00 after overflow, got X=%02X", cpu.X)
		}
		if !cpu.getFlag(FlagZero) {
			t.Error("Zero flag should be set after overflow")
		}
		if cycles != 2 {
			t.Errorf("Expected 2 cycles for INX, got %d", cycles)
		}
	})
	
	t.Run("DEX_Underflow", func(t *testing.T) {
		cpu := createTestCPU()
		cpu.X = 0x00
		cpu.PC = 0x0200
		cpu.Memory.Write(0x0200, 0xCA) // DEX
		
		cycles := cpu.Step()
		
		if cpu.X != 0xFF {
			t.Errorf("Expected X=FF after underflow, got X=%02X", cpu.X)
		}
		if !cpu.getFlag(FlagNegative) {
			t.Error("Negative flag should be set after underflow")
		}
		if cycles != 2 {
			t.Errorf("Expected 2 cycles for DEX, got %d", cycles)
		}
	})
}

// Test NOP instruction variations
func TestNOPInstructions(t *testing.T) {
	t.Run("Official_NOP", func(t *testing.T) {
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.Memory.Write(0x0200, 0xEA) // NOP
		
		originalA := cpu.A
		originalX := cpu.X
		originalY := cpu.Y
		originalP := cpu.P
		
		cycles := cpu.Step()
		
		// NOP should not change any registers or flags
		if cpu.A != originalA || cpu.X != originalX || cpu.Y != originalY || cpu.P != originalP {
			t.Error("NOP should not change any registers or flags")
		}
		if cpu.PC != 0x0201 {
			t.Errorf("Expected PC=0201 after NOP, got PC=%04X", cpu.PC)
		}
		if cycles != 2 {
			t.Errorf("Expected 2 cycles for NOP, got %d", cycles)
		}
	})
	
	t.Run("Illegal_NOP_Immediate", func(t *testing.T) {
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.Memory.Write(0x0200, 0x80) // Illegal NOP #imm
		cpu.Memory.Write(0x0201, 0x42) // Immediate value
		
		cycles := cpu.Step()
		
		if cpu.PC != 0x0202 {
			t.Errorf("Expected PC=0202 after illegal NOP #imm, got PC=%04X", cpu.PC)
		}
		if cycles != 2 {
			t.Errorf("Expected 2 cycles for illegal NOP #imm, got %d", cycles)
		}
	})
}

// Test arithmetic edge cases
func TestArithmeticEdgeCases(t *testing.T) {
	t.Run("ADC_Decimal_Mode", func(t *testing.T) {
		cpu := createTestCPU()
		cpu.setFlag(FlagDecimal, true)
		cpu.setFlag(FlagCarry, false)
		cpu.A = 0x09
		cpu.PC = 0x0200
		cpu.Memory.Write(0x0200, 0x69) // ADC #$01
		cpu.Memory.Write(0x0201, 0x01)
		
		cycles := cpu.Step()
		
		// NES CPU (2A03) does not support decimal mode - should work as binary
		// 0x09 + 0x01 = 0x0A in binary mode
		if cpu.A != 0x0A {
			t.Errorf("Expected A=0A in binary mode (NES has no decimal mode), got A=%02X", cpu.A)
		}
		if cycles != 2 {
			t.Errorf("Expected 2 cycles for ADC, got %d", cycles)
		}
	})
	
	t.Run("SBC_With_Borrow", func(t *testing.T) {
		cpu := createTestCPU()
		cpu.setFlag(FlagCarry, false) // Borrow needed
		cpu.A = 0x50
		cpu.PC = 0x0200
		cpu.Memory.Write(0x0200, 0xE9) // SBC #$F0
		cpu.Memory.Write(0x0201, 0xF0)
		
		cycles := cpu.Step()
		
		// 0x50 - 0xF0 - 1 (borrow) = 0x5F
		if cpu.A != 0x5F {
			t.Errorf("Expected A=5F with borrow, got A=%02X", cpu.A)
		}
		if cpu.getFlag(FlagCarry) {
			t.Error("Carry should be clear (borrow occurred)")
		}
		if cycles != 2 {
			t.Errorf("Expected 2 cycles for SBC, got %d", cycles)
		}
	})
	
	t.Run("ADC_Overflow_Positive", func(t *testing.T) {
		cpu := createTestCPU()
		cpu.A = 0x50 // Positive
		cpu.PC = 0x0200
		cpu.Memory.Write(0x0200, 0x69) // ADC #$50
		cpu.Memory.Write(0x0201, 0x50) // Positive
		
		cycles := cpu.Step()
		
		// 0x50 + 0x50 = 0xA0 (negative result from positive operands)
		if cpu.A != 0xA0 {
			t.Errorf("Expected A=A0, got A=%02X", cpu.A)
		}
		if !cpu.getFlag(FlagOverflow) {
			t.Error("Overflow flag should be set")
		}
		if !cpu.getFlag(FlagNegative) {
			t.Error("Negative flag should be set")
		}
		if cycles != 2 {
			t.Errorf("Expected 2 cycles for ADC, got %d", cycles)
		}
	})
	
	t.Run("ADC_Overflow_Negative", func(t *testing.T) {
		cpu := createTestCPU()
		cpu.A = 0x80 // Negative
		cpu.PC = 0x0200
		cpu.Memory.Write(0x0200, 0x69) // ADC #$80
		cpu.Memory.Write(0x0201, 0x80) // Negative
		
		cycles := cpu.Step()
		
		// 0x80 + 0x80 = 0x00 (positive result from negative operands)
		if cpu.A != 0x00 {
			t.Errorf("Expected A=00, got A=%02X", cpu.A)
		}
		if !cpu.getFlag(FlagOverflow) {
			t.Error("Overflow flag should be set")
		}
		if !cpu.getFlag(FlagCarry) {
			t.Error("Carry flag should be set")
		}
		if !cpu.getFlag(FlagZero) {
			t.Error("Zero flag should be set")
		}
		if cycles != 2 {
			t.Errorf("Expected 2 cycles for ADC, got %d", cycles)
		}
	})
}

// Test page boundary crossing timing
func TestPageBoundaryCrossing(t *testing.T) {
	t.Run("LDA_AbsoluteX_PageCross", func(t *testing.T) {
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.X = 0xFF
		
		cpu.Memory.Write(0x0200, 0xBD) // LDA abs,X
		cpu.Memory.Write(0x0201, 0x80) // Low byte
		cpu.Memory.Write(0x0202, 0x80) // High byte (base = 0x8080)
		cpu.Memory.Write(0x817F, 0x42) // Target (0x8080 + 0xFF = 0x817F)
		
		cycles := cpu.Step()
		
		if cpu.A != 0x42 {
			t.Errorf("Expected A=42, got A=%02X", cpu.A)
		}
		if cycles != 5 { // Extra cycle for page crossing
			t.Errorf("Expected 5 cycles for page crossing, got %d", cycles)
		}
	})
	
	t.Run("LDA_AbsoluteX_NoPageCross", func(t *testing.T) {
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.X = 0x10
		
		cpu.Memory.Write(0x0200, 0xBD) // LDA abs,X
		cpu.Memory.Write(0x0201, 0x80) // Low byte
		cpu.Memory.Write(0x0202, 0x80) // High byte (base = 0x8080)
		cpu.Memory.Write(0x8090, 0x55) // Target (0x8080 + 0x10 = 0x8090)
		
		cycles := cpu.Step()
		
		if cpu.A != 0x55 {
			t.Errorf("Expected A=55, got A=%02X", cpu.A)
		}
		if cycles != 4 { // No extra cycle
			t.Errorf("Expected 4 cycles for no page crossing, got %d", cycles)
		}
	})
}

// Test edge cases and error conditions
func TestEdgeCases(t *testing.T) {
	t.Run("Stack_Underflow", func(t *testing.T) {
		cpu := createTestCPU()
		cpu.SP = 0xFF // Stack is full
		cpu.PC = 0x0200
		cpu.Memory.Write(0x0200, 0x68) // PLA
		
		cycles := cpu.Step()
		
		// Stack should wrap around
		if cpu.SP != 0x00 {
			t.Errorf("Expected SP=00 after stack underflow, got SP=%02X", cpu.SP)
		}
		if cycles != 4 {
			t.Errorf("Expected 4 cycles for PLA, got %d", cycles)
		}
	})
	
	t.Run("Stack_Overflow", func(t *testing.T) {
		cpu := createTestCPU()
		cpu.SP = 0x00 // Stack is empty
		cpu.A = 0x42
		cpu.PC = 0x0200
		cpu.Memory.Write(0x0200, 0x48) // PHA
		
		cycles := cpu.Step()
		
		// Stack should wrap around
		if cpu.SP != 0xFF {
			t.Errorf("Expected SP=FF after stack overflow, got SP=%02X", cpu.SP)
		}
		if cpu.Memory.Read(0x0100) != 0x42 {
			t.Errorf("Expected stack[0x100]=42, got %02X", cpu.Memory.Read(0x0100))
		}
		if cycles != 3 {
			t.Errorf("Expected 3 cycles for PHA, got %d", cycles)
		}
	})
	
	t.Run("Zero_Page_Wraparound", func(t *testing.T) {
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.X = 0x10
		
		cpu.Memory.Write(0x0200, 0xB5) // LDA zp,X
		cpu.Memory.Write(0x0201, 0xF0) // Zero page address
		cpu.Memory.Write(0x00, 0x99)   // Wrapped address (0xF0 + 0x10 = 0x00)
		
		cycles := cpu.Step()
		
		if cpu.A != 0x99 {
			t.Errorf("Expected A=99 from wrapped address, got A=%02X", cpu.A)
		}
		if cycles != 4 {
			t.Errorf("Expected 4 cycles for LDA zp,X, got %d", cycles)
		}
	})
}