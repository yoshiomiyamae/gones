package cpu

import (
	"testing"
)

// Test missing logical instructions comprehensively
func TestLogicalInstructionsComplete(t *testing.T) {
	t.Run("AND_AllAddressingModes", func(t *testing.T) {
		testCases := []struct {
			name     string
			opcode   uint8
			setup    func(*CPU)
			expected uint8
			cycles   int
		}{
			{"AND_ZeroPage", 0x25, func(cpu *CPU) {
				cpu.Memory.Write(0x0201, 0x10)
				cpu.Memory.Write(0x10, 0x0F)
				cpu.A = 0xFF
			}, 0x0F, 3},
			{"AND_ZeroPageX", 0x35, func(cpu *CPU) {
				cpu.Memory.Write(0x0201, 0x10)
				cpu.Memory.Write(0x11, 0x33)
				cpu.A = 0xFF
				cpu.X = 0x01
			}, 0x33, 4},
			{"AND_Absolute", 0x2D, func(cpu *CPU) {
				cpu.Memory.Write(0x0201, 0x00)
				cpu.Memory.Write(0x0202, 0x80)
				cpu.Memory.Write(0x8000, 0xAA)
				cpu.A = 0xFF
			}, 0xAA, 4},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cpu := createTestCPU()
				cpu.PC = 0x0200
				cpu.Memory.Write(0x0200, tc.opcode)
				tc.setup(cpu)
				
				cycles := cpu.Step()
				
				if cpu.A != tc.expected {
					t.Errorf("Expected A=%02X, got A=%02X", tc.expected, cpu.A)
				}
				if cycles != tc.cycles {
					t.Errorf("Expected %d cycles, got %d", tc.cycles, cycles)
				}
			})
		}
	})
	
	t.Run("ORA_AllAddressingModes", func(t *testing.T) {
		testCases := []struct {
			name     string
			opcode   uint8
			setup    func(*CPU)
			expected uint8
			cycles   int
		}{
			{"ORA_ZeroPage", 0x05, func(cpu *CPU) {
				cpu.Memory.Write(0x0201, 0x10)
				cpu.Memory.Write(0x10, 0x0F)
				cpu.A = 0xF0
			}, 0xFF, 3},
			{"ORA_AbsoluteX", 0x1D, func(cpu *CPU) {
				cpu.Memory.Write(0x0201, 0x00)
				cpu.Memory.Write(0x0202, 0x80)
				cpu.Memory.Write(0x8001, 0x55)
				cpu.A = 0xAA
				cpu.X = 0x01
			}, 0xFF, 4},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cpu := createTestCPU()
				cpu.PC = 0x0200
				cpu.Memory.Write(0x0200, tc.opcode)
				tc.setup(cpu)
				
				cycles := cpu.Step()
				
				if cpu.A != tc.expected {
					t.Errorf("Expected A=%02X, got A=%02X", tc.expected, cpu.A)
				}
				if cycles != tc.cycles {
					t.Errorf("Expected %d cycles, got %d", tc.cycles, cycles)
				}
			})
		}
	})
	
	t.Run("EOR_AllAddressingModes", func(t *testing.T) {
		testCases := []struct {
			name     string
			opcode   uint8
			setup    func(*CPU)
			expected uint8
		}{
			{"EOR_ZeroPage", 0x45, func(cpu *CPU) {
				cpu.Memory.Write(0x0201, 0x10)
				cpu.Memory.Write(0x10, 0xFF)
				cpu.A = 0xAA
			}, 0x55},
			{"EOR_IndexedIndirect", 0x41, func(cpu *CPU) {
				cpu.Memory.Write(0x0201, 0x20)
				cpu.Memory.Write(0x22, 0x00)
				cpu.Memory.Write(0x23, 0x80)
				cpu.Memory.Write(0x8000, 0x33)
				cpu.A = 0x33
				cpu.X = 0x02
			}, 0x00},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cpu := createTestCPU()
				cpu.PC = 0x0200
				cpu.Memory.Write(0x0200, tc.opcode)
				tc.setup(cpu)
				
				cpu.Step()
				
				if cpu.A != tc.expected {
					t.Errorf("Expected A=%02X, got A=%02X", tc.expected, cpu.A)
				}
			})
		}
	})
}

// Test all shift and rotate instructions with all addressing modes
func TestShiftRotateComplete(t *testing.T) {
	t.Run("ASL_AllModes", func(t *testing.T) {
		// Test ASL zeropage,X
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.X = 0x01
		cpu.Memory.Write(0x0200, 0x16) // ASL zp,X
		cpu.Memory.Write(0x0201, 0x10)
		cpu.Memory.Write(0x11, 0x40)
		
		cycles := cpu.Step()
		
		if cpu.Memory.Read(0x11) != 0x80 {
			t.Errorf("Expected memory[0x11]=80, got %02X", cpu.Memory.Read(0x11))
		}
		if !cpu.getFlag(FlagNegative) {
			t.Error("Negative flag should be set")
		}
		if cycles != 6 {
			t.Errorf("Expected 6 cycles for ASL zp,X, got %d", cycles)
		}
		
		// Test ASL absolute,X
		cpu = createTestCPU()
		cpu.PC = 0x0200
		cpu.X = 0x02
		cpu.Memory.Write(0x0200, 0x1E) // ASL abs,X
		cpu.Memory.Write(0x0201, 0x00)
		cpu.Memory.Write(0x0202, 0x80)
		cpu.Memory.Write(0x8002, 0x81)
		
		cycles = cpu.Step()
		
		if cpu.Memory.Read(0x8002) != 0x02 {
			t.Errorf("Expected memory[0x8002]=02, got %02X", cpu.Memory.Read(0x8002))
		}
		if !cpu.getFlag(FlagCarry) {
			t.Error("Carry flag should be set")
		}
		if cycles != 7 {
			t.Errorf("Expected 7 cycles for ASL abs,X, got %d", cycles)
		}
	})
	
	t.Run("LSR_AllModes", func(t *testing.T) {
		// Test LSR zeropage
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.Memory.Write(0x0200, 0x46) // LSR zp
		cpu.Memory.Write(0x0201, 0x10)
		cpu.Memory.Write(0x10, 0x81)
		
		cycles := cpu.Step()
		
		if cpu.Memory.Read(0x10) != 0x40 {
			t.Errorf("Expected memory[0x10]=40, got %02X", cpu.Memory.Read(0x10))
		}
		if !cpu.getFlag(FlagCarry) {
			t.Error("Carry flag should be set")
		}
		if cycles != 5 {
			t.Errorf("Expected 5 cycles for LSR zp, got %d", cycles)
		}
	})
	
	t.Run("ROL_AllModes", func(t *testing.T) {
		// Test ROL zeropage with carry
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.setFlag(FlagCarry, true)
		cpu.Memory.Write(0x0200, 0x26) // ROL zp
		cpu.Memory.Write(0x0201, 0x10)
		cpu.Memory.Write(0x10, 0x80)
		
		cycles := cpu.Step()
		
		if cpu.Memory.Read(0x10) != 0x01 {
			t.Errorf("Expected memory[0x10]=01, got %02X", cpu.Memory.Read(0x10))
		}
		if !cpu.getFlag(FlagCarry) {
			t.Error("Carry flag should be set from bit 7")
		}
		if cycles != 5 {
			t.Errorf("Expected 5 cycles for ROL zp, got %d", cycles)
		}
	})
	
	t.Run("ROR_AllModes", func(t *testing.T) {
		// Test ROR absolute
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.setFlag(FlagCarry, true)
		cpu.Memory.Write(0x0200, 0x6E) // ROR abs
		cpu.Memory.Write(0x0201, 0x00)
		cpu.Memory.Write(0x0202, 0x80)
		cpu.Memory.Write(0x8000, 0x01)
		
		cycles := cpu.Step()
		
		if cpu.Memory.Read(0x8000) != 0x80 {
			t.Errorf("Expected memory[0x8000]=80, got %02X", cpu.Memory.Read(0x8000))
		}
		if !cpu.getFlag(FlagCarry) {
			t.Error("Carry flag should be set from bit 0")
		}
		if !cpu.getFlag(FlagNegative) {
			t.Error("Negative flag should be set")
		}
		if cycles != 6 {
			t.Errorf("Expected 6 cycles for ROR abs, got %d", cycles)
		}
	})
}

// Test compare instructions with all addressing modes
func TestCompareInstructionsComplete(t *testing.T) {
	t.Run("CPX_AllModes", func(t *testing.T) {
		testCases := []struct {
			name     string
			opcode   uint8
			xValue   uint8
			memValue uint8
			expCarry bool
			expZero  bool
			expNeg   bool
		}{
			{"CPX_Equal", 0xE0, 0x42, 0x42, true, true, false},
			{"CPX_Greater", 0xE0, 0x50, 0x40, true, false, false},
			{"CPX_Less", 0xE0, 0x30, 0x40, false, false, true},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cpu := createTestCPU()
				cpu.PC = 0x0200
				cpu.X = tc.xValue
				cpu.Memory.Write(0x0200, tc.opcode) // CPX #imm
				cpu.Memory.Write(0x0201, tc.memValue)
				
				cycles := cpu.Step()
				
				if cpu.getFlag(FlagCarry) != tc.expCarry {
					t.Errorf("Expected Carry=%v, got %v", tc.expCarry, cpu.getFlag(FlagCarry))
				}
				if cpu.getFlag(FlagZero) != tc.expZero {
					t.Errorf("Expected Zero=%v, got %v", tc.expZero, cpu.getFlag(FlagZero))
				}
				if cpu.getFlag(FlagNegative) != tc.expNeg {
					t.Errorf("Expected Negative=%v, got %v", tc.expNeg, cpu.getFlag(FlagNegative))
				}
				if cycles != 2 {
					t.Errorf("Expected 2 cycles, got %d", cycles)
				}
			})
		}
		
		// Test CPX zeropage
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.X = 0x80
		cpu.Memory.Write(0x0200, 0xE4) // CPX zp
		cpu.Memory.Write(0x0201, 0x10)
		cpu.Memory.Write(0x10, 0x80)
		
		cycles := cpu.Step()
		
		if !cpu.getFlag(FlagZero) {
			t.Error("Zero flag should be set when X == memory")
		}
		if cycles != 3 {
			t.Errorf("Expected 3 cycles for CPX zp, got %d", cycles)
		}
	})
	
	t.Run("CPY_AllModes", func(t *testing.T) {
		// Test CPY absolute
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.Y = 0x10
		cpu.Memory.Write(0x0200, 0xCC) // CPY abs
		cpu.Memory.Write(0x0201, 0x00)
		cpu.Memory.Write(0x0202, 0x80)
		cpu.Memory.Write(0x8000, 0x20)
		
		cycles := cpu.Step()
		
		if cpu.getFlag(FlagCarry) {
			t.Error("Carry should be clear when Y < memory")
		}
		if !cpu.getFlag(FlagNegative) {
			t.Error("Negative flag should be set")
		}
		if cycles != 4 {
			t.Errorf("Expected 4 cycles for CPY abs, got %d", cycles)
		}
	})
}

// Test BIT instruction comprehensively
func TestBITInstructionComplete(t *testing.T) {
	t.Run("BIT_ZeroPage", func(t *testing.T) {
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.A = 0x40
		cpu.Memory.Write(0x0200, 0x24) // BIT zp
		cpu.Memory.Write(0x0201, 0x10)
		cpu.Memory.Write(0x10, 0x40) // Same bit set as A
		
		cycles := cpu.Step()
		
		if cpu.getFlag(FlagZero) {
			t.Error("Zero flag should not be set (A & memory != 0)")
		}
		if cpu.getFlag(FlagNegative) {
			t.Error("Negative flag should not be set (bit 7 of memory)")
		}
		if !cpu.getFlag(FlagOverflow) {
			t.Error("Overflow flag should be set (bit 6 of memory)")
		}
		if cycles != 3 {
			t.Errorf("Expected 3 cycles for BIT zp, got %d", cycles)
		}
	})
	
	t.Run("BIT_Absolute", func(t *testing.T) {
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.A = 0x0F
		cpu.Memory.Write(0x0200, 0x2C) // BIT abs
		cpu.Memory.Write(0x0201, 0x00)
		cpu.Memory.Write(0x0202, 0x80)
		cpu.Memory.Write(0x8000, 0xF0) // No common bits with A
		
		cycles := cpu.Step()
		
		if !cpu.getFlag(FlagZero) {
			t.Error("Zero flag should be set (A & memory == 0)")
		}
		if !cpu.getFlag(FlagNegative) {
			t.Error("Negative flag should be set (bit 7 of memory)")
		}
		if !cpu.getFlag(FlagOverflow) {
			t.Error("Overflow flag should be set (bit 6 of memory)")
		}
		if cycles != 4 {
			t.Errorf("Expected 4 cycles for BIT abs, got %d", cycles)
		}
	})
}

// Test store instructions with all addressing modes
func TestStoreInstructionsComplete(t *testing.T) {
	t.Run("STX_AllModes", func(t *testing.T) {
		// Test STX zeropage,Y
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.X = 0x42
		cpu.Y = 0x05
		cpu.Memory.Write(0x0200, 0x96) // STX zp,Y
		cpu.Memory.Write(0x0201, 0x10)
		
		cycles := cpu.Step()
		
		if cpu.Memory.Read(0x15) != 0x42 {
			t.Errorf("Expected memory[0x15]=42, got %02X", cpu.Memory.Read(0x15))
		}
		if cycles != 4 {
			t.Errorf("Expected 4 cycles for STX zp,Y, got %d", cycles)
		}
		
		// Test STX absolute
		cpu = createTestCPU()
		cpu.PC = 0x0200
		cpu.X = 0x33
		cpu.Memory.Write(0x0200, 0x8E) // STX abs
		cpu.Memory.Write(0x0201, 0x00)
		cpu.Memory.Write(0x0202, 0x80)
		
		cycles = cpu.Step()
		
		if cpu.Memory.Read(0x8000) != 0x33 {
			t.Errorf("Expected memory[0x8000]=33, got %02X", cpu.Memory.Read(0x8000))
		}
		if cycles != 4 {
			t.Errorf("Expected 4 cycles for STX abs, got %d", cycles)
		}
	})
	
	t.Run("STY_AllModes", func(t *testing.T) {
		// Test STY zeropage,X
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.Y = 0x55
		cpu.X = 0x03
		cpu.Memory.Write(0x0200, 0x94) // STY zp,X
		cpu.Memory.Write(0x0201, 0x20)
		
		cycles := cpu.Step()
		
		if cpu.Memory.Read(0x23) != 0x55 {
			t.Errorf("Expected memory[0x23]=55, got %02X", cpu.Memory.Read(0x23))
		}
		if cycles != 4 {
			t.Errorf("Expected 4 cycles for STY zp,X, got %d", cycles)
		}
	})
	
	t.Run("STA_IndirectModes", func(t *testing.T) {
		// Test STA (zp,X)
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.A = 0x77
		cpu.X = 0x02
		cpu.Memory.Write(0x0200, 0x81) // STA (zp,X)
		cpu.Memory.Write(0x0201, 0x10)
		cpu.Memory.Write(0x12, 0x00) // Target address low
		cpu.Memory.Write(0x13, 0x80) // Target address high
		
		cycles := cpu.Step()
		
		if cpu.Memory.Read(0x8000) != 0x77 {
			t.Errorf("Expected memory[0x8000]=77, got %02X", cpu.Memory.Read(0x8000))
		}
		if cycles != 6 {
			t.Errorf("Expected 6 cycles for STA (zp,X), got %d", cycles)
		}
		
		// Test STA (zp),Y
		cpu = createTestCPU()
		cpu.PC = 0x0200
		cpu.A = 0x88
		cpu.Y = 0x05
		cpu.Memory.Write(0x0200, 0x91) // STA (zp),Y
		cpu.Memory.Write(0x0201, 0x20)
		cpu.Memory.Write(0x20, 0x00) // Base address low
		cpu.Memory.Write(0x21, 0x80) // Base address high
		
		cycles = cpu.Step()
		
		if cpu.Memory.Read(0x8005) != 0x88 {
			t.Errorf("Expected memory[0x8005]=88, got %02X", cpu.Memory.Read(0x8005))
		}
		if cycles != 6 {
			t.Errorf("Expected 6 cycles for STA (zp),Y, got %d", cycles)
		}
	})
}

// Test load instructions with all addressing modes
func TestLoadInstructionsComplete(t *testing.T) {
	t.Run("LDX_AllModes", func(t *testing.T) {
		// Test LDX zeropage,Y
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.Y = 0x03
		cpu.Memory.Write(0x0200, 0xB6) // LDX zp,Y
		cpu.Memory.Write(0x0201, 0x10)
		cpu.Memory.Write(0x13, 0x99)
		
		cycles := cpu.Step()
		
		if cpu.X != 0x99 {
			t.Errorf("Expected X=99, got X=%02X", cpu.X)
		}
		if !cpu.getFlag(FlagNegative) {
			t.Error("Negative flag should be set")
		}
		if cycles != 4 {
			t.Errorf("Expected 4 cycles for LDX zp,Y, got %d", cycles)
		}
		
		// Test LDX absolute,Y
		cpu = createTestCPU()
		cpu.PC = 0x0200
		cpu.Y = 0x01
		cpu.Memory.Write(0x0200, 0xBE) // LDX abs,Y
		cpu.Memory.Write(0x0201, 0xFF)
		cpu.Memory.Write(0x0202, 0x7F)
		cpu.Memory.Write(0x8000, 0x00) // Page crossing: 0x7FFF + 1 = 0x8000
		
		cycles = cpu.Step()
		
		if cpu.X != 0x00 {
			t.Errorf("Expected X=00, got X=%02X", cpu.X)
		}
		if !cpu.getFlag(FlagZero) {
			t.Error("Zero flag should be set")
		}
		if cycles != 5 { // Page crossing adds cycle
			t.Errorf("Expected 5 cycles for LDX abs,Y with page crossing, got %d", cycles)
		}
	})
	
	t.Run("LDY_AllModes", func(t *testing.T) {
		// Test LDY absolute,X
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.X = 0x02
		cpu.Memory.Write(0x0200, 0xBC) // LDY abs,X
		cpu.Memory.Write(0x0201, 0x00)
		cpu.Memory.Write(0x0202, 0x80)
		cpu.Memory.Write(0x8002, 0x44)
		
		cycles := cpu.Step()
		
		if cpu.Y != 0x44 {
			t.Errorf("Expected Y=44, got Y=%02X", cpu.Y)
		}
		if cycles != 4 { // No page crossing
			t.Errorf("Expected 4 cycles for LDY abs,X, got %d", cycles)
		}
	})
}

// Test arithmetic instructions with all addressing modes and edge cases
func TestArithmeticComplete(t *testing.T) {
	t.Run("ADC_AllModes", func(t *testing.T) {
		// Test ADC (zp,X)
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.A = 0x10
		cpu.X = 0x04
		cpu.Memory.Write(0x0200, 0x61) // ADC (zp,X)
		cpu.Memory.Write(0x0201, 0x20)
		cpu.Memory.Write(0x24, 0x00) // Target address low
		cpu.Memory.Write(0x25, 0x18) // Target address high
		cpu.Memory.Write(0x1800, 0x20)
		
		cycles := cpu.Step()
		
		if cpu.A != 0x30 {
			t.Errorf("Expected A=30, got A=%02X", cpu.A)
		}
		if cycles != 6 {
			t.Errorf("Expected 6 cycles for ADC (zp,X), got %d", cycles)
		}
		
		// Test ADC (zp),Y
		cpu = createTestCPU()
		cpu.PC = 0x0200
		cpu.A = 0x50
		cpu.Y = 0x02
		cpu.setFlag(FlagCarry, true)
		cpu.Memory.Write(0x0200, 0x71) // ADC (zp),Y
		cpu.Memory.Write(0x0201, 0x30)
		cpu.Memory.Write(0x30, 0x00) // Base address low
		cpu.Memory.Write(0x31, 0x19) // Base address high
		cpu.Memory.Write(0x1902, 0x2F)
		
		cycles = cpu.Step()
		
		if cpu.A != 0x80 { // 0x50 + 0x2F + 1 (carry)
			t.Errorf("Expected A=80, got A=%02X", cpu.A)
		}
		if !cpu.getFlag(FlagNegative) {
			t.Error("Negative flag should be set")
		}
		if cycles != 5 {
			t.Errorf("Expected 5 cycles for ADC (zp),Y, got %d", cycles)
		}
	})
	
	t.Run("SBC_AllModes", func(t *testing.T) {
		// Test SBC zeropage,X
		cpu := createTestCPU()
		cpu.PC = 0x0200
		cpu.A = 0x50
		cpu.X = 0x01
		cpu.setFlag(FlagCarry, true) // No borrow
		cpu.Memory.Write(0x0200, 0xF5) // SBC zp,X
		cpu.Memory.Write(0x0201, 0x10)
		cpu.Memory.Write(0x11, 0x30)
		
		cycles := cpu.Step()
		
		if cpu.A != 0x20 {
			t.Errorf("Expected A=20, got A=%02X", cpu.A)
		}
		if !cpu.getFlag(FlagCarry) {
			t.Error("Carry should be set (no borrow)")
		}
		if cycles != 4 {
			t.Errorf("Expected 4 cycles for SBC zp,X, got %d", cycles)
		}
		
		// Test SBC absolute,Y with page crossing
		cpu = createTestCPU()
		cpu.PC = 0x0200
		cpu.A = 0x80
		cpu.Y = 0xFF
		cpu.setFlag(FlagCarry, false) // Borrow needed
		cpu.Memory.Write(0x0200, 0xF9) // SBC abs,Y
		cpu.Memory.Write(0x0201, 0x01)
		cpu.Memory.Write(0x0202, 0x10)
		cpu.Memory.Write(0x1100, 0x01) // 0x1001 + 0xFF = 0x1100 (within RAM)
		
		cycles = cpu.Step()
		
		if cpu.A != 0x7E { // 0x80 - 0x01 - 1 (borrow)
			t.Errorf("Expected A=7E, got A=%02X", cpu.A)
		}
		if !cpu.getFlag(FlagCarry) {
			t.Error("Carry should be set (no borrow occurred)")
		}
		if cycles != 5 { // Page crossing adds cycle
			t.Errorf("Expected 5 cycles for SBC abs,Y with page crossing, got %d", cycles)
		}
	})
}