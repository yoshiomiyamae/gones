package cpu

import (
	"testing"

	"github.com/yoshiomiyamaegones/pkg/memory"
)

// createTestCPU creates a CPU instance for testing
func createTestCPU() *CPU {
	mem := memory.New()
	cpu := New(mem)

	// Set reset vector to 0x0200 for testing
	mem.Write(0xFFFC, 0x00)
	mem.Write(0xFFFD, 0x02)

	cpu.Reset()
	return cpu
}

// Test CPU Reset
func TestCPUReset(t *testing.T) {
	cpu := createTestCPU()

	// Set some non-default values
	cpu.A = 0xFF
	cpu.X = 0xFF
	cpu.Y = 0xFF
	cpu.SP = 0x00
	cpu.P = 0xFF

	// Reset should restore defaults
	cpu.Reset()

	if cpu.A != 0 {
		t.Errorf("Expected A=0, got A=%02X", cpu.A)
	}
	if cpu.X != 0 {
		t.Errorf("Expected X=0, got X=%02X", cpu.X)
	}
	if cpu.Y != 0 {
		t.Errorf("Expected Y=0, got Y=%02X", cpu.Y)
	}
	if cpu.SP != 0xFD {
		t.Errorf("Expected SP=0xFD, got SP=%02X", cpu.SP)
	}
	if cpu.P != (FlagUnused | FlagInterrupt) {
		t.Errorf("Expected P=%02X, got P=%02X", FlagUnused|FlagInterrupt, cpu.P)
	}
}

// Test flag operations
func TestFlags(t *testing.T) {
	cpu := createTestCPU()

	// Test setting flags
	cpu.setFlag(FlagCarry, true)
	if !cpu.getFlag(FlagCarry) {
		t.Error("Carry flag should be set")
	}

	cpu.setFlag(FlagZero, true)
	if !cpu.getFlag(FlagZero) {
		t.Error("Zero flag should be set")
	}

	// Test clearing flags
	cpu.setFlag(FlagCarry, false)
	if cpu.getFlag(FlagCarry) {
		t.Error("Carry flag should be clear")
	}

	// Test multiple flags
	cpu.P = 0
	cpu.setFlag(FlagCarry, true)
	cpu.setFlag(FlagNegative, true)
	expected := uint8(FlagCarry | FlagNegative)
	if cpu.P != expected {
		t.Errorf("Expected P=%02X, got P=%02X", expected, cpu.P)
	}
}

// Test stack operations
func TestStack(t *testing.T) {
	cpu := createTestCPU()

	initialSP := cpu.SP

	// Test push
	cpu.push(0x42)
	if cpu.SP != initialSP-1 {
		t.Errorf("Expected SP=%02X, got SP=%02X", initialSP-1, cpu.SP)
	}

	// Test pop
	value := cpu.pop()
	if value != 0x42 {
		t.Errorf("Expected popped value=0x42, got %02X", value)
	}
	if cpu.SP != initialSP {
		t.Errorf("Expected SP=%02X, got SP=%02X", initialSP, cpu.SP)
	}

	// Test 16-bit operations
	cpu.push16(0x1234)
	result := cpu.pop16()
	if result != 0x1234 {
		t.Errorf("Expected 0x1234, got %04X", result)
	}
}

// Test addressing modes
func TestAddressingModes(t *testing.T) {
	cpu := createTestCPU()

	// Set up memory
	cpu.Memory.Write(0x00, 0x10) // Zero page
	cpu.Memory.Write(0x01, 0x20)
	cpu.Memory.Write(0x10, 0x30) // Target
	cpu.Memory.Write(0x1000, 0x40)
	cpu.Memory.Write(0x1001, 0x50)

	// Set registers
	cpu.X = 0x01
	cpu.Y = 0x02
	cpu.PC = 0x1000

	// Test immediate
	addr, _ := cpu.getOperandAddress(AddrImmediate)
	if addr != 0x1000 {
		t.Errorf("Immediate: expected addr=0x1000, got %04X", addr)
	}

	// Reset PC for next test
	cpu.PC = 0x1000

	// Test zero page
	addr, _ = cpu.getOperandAddress(AddrZeroPage)
	if addr != 0x40 {
		t.Errorf("Zero page: expected addr=0x40, got %04X", addr)
	}

	// Reset PC for next test
	cpu.PC = 0x1000

	// Test zero page,X
	addr, _ = cpu.getOperandAddress(AddrZeroPageX)
	if addr != 0x41 {
		t.Errorf("Zero page,X: expected addr=0x41, got %04X", addr)
	}
}

// Test addressing mode edge cases
func TestAddressingModeEdgeCases(t *testing.T) {
	cpu := createTestCPU()

	// Test zero page wraparound
	cpu.X = 0xFF
	cpu.PC = 0x1000
	cpu.Memory.Write(0x1000, 0xFF)

	addr, _ := cpu.getOperandAddress(AddrZeroPageX)
	if addr != 0xFE { // 0xFF + 0xFF = 0x1FE, but wrapped to 0xFE
		t.Errorf("Zero page X wraparound: expected addr=0xFE, got %04X", addr)
	}

	// Test page crossing detection
	cpu.PC = 0x1000
	cpu.Y = 0xFF
	cpu.Memory.Write(0x1000, 0xFF)
	cpu.Memory.Write(0x1001, 0x10) // Address 0x10FF

	addr, pageCrossed := cpu.getOperandAddress(AddrAbsoluteY)
	expectedAddr := uint16(0x10FF + 0xFF) // = 0x11FE
	if addr != expectedAddr {
		t.Errorf("Absolute,Y: expected addr=%04X, got %04X", expectedAddr, addr)
	}
	if !pageCrossed {
		t.Error("Page crossing should be detected")
	}
}

// Helper function to set up CPU with program
func setupCPUWithProgram(program []uint8) *CPU {
	cpu := createTestCPU()

	// Load program starting at 0x0200 (zero page + stack)
	// This is safe RAM area for testing
	startAddr := uint16(0x0200)
	for i, b := range program {
		addr := startAddr + uint16(i)
		cpu.Memory.Write(addr, b)
	}

	// Set PC to start of program
	cpu.PC = startAddr

	return cpu
}

// Test LDA instruction
func TestLDA(t *testing.T) {
	// Test LDA immediate
	cpu := setupCPUWithProgram([]uint8{0xA9, 0x42}) // LDA #$42

	cycles := cpu.Step()

	if cpu.A != 0x42 {
		t.Errorf("Expected A=0x42, got A=%02X", cpu.A)
	}
	if cycles != 2 {
		t.Errorf("Expected 2 cycles, got %d", cycles)
	}
	if cpu.getFlag(FlagZero) {
		t.Error("Zero flag should not be set")
	}
	if cpu.getFlag(FlagNegative) {
		t.Error("Negative flag should not be set")
	}

	// Test LDA with zero
	cpu = setupCPUWithProgram([]uint8{0xA9, 0x00}) // LDA #$00
	cpu.Step()

	if cpu.A != 0x00 {
		t.Errorf("Expected A=0x00, got A=%02X", cpu.A)
	}
	if !cpu.getFlag(FlagZero) {
		t.Error("Zero flag should be set")
	}

	// Test LDA with negative value
	cpu = setupCPUWithProgram([]uint8{0xA9, 0x80}) // LDA #$80
	cpu.Step()

	if cpu.A != 0x80 {
		t.Errorf("Expected A=0x80, got A=%02X", cpu.A)
	}
	if !cpu.getFlag(FlagNegative) {
		t.Error("Negative flag should be set")
	}
}

// Test LDX instruction
func TestLDX(t *testing.T) {
	cpu := setupCPUWithProgram([]uint8{0xA2, 0x33}) // LDX #$33

	cycles := cpu.Step()

	if cpu.X != 0x33 {
		t.Errorf("Expected X=0x33, got X=%02X", cpu.X)
	}
	if cycles != 2 {
		t.Errorf("Expected 2 cycles, got %d", cycles)
	}
}

// Test LDY instruction
func TestLDY(t *testing.T) {
	cpu := setupCPUWithProgram([]uint8{0xA0, 0x44}) // LDY #$44

	cycles := cpu.Step()

	if cpu.Y != 0x44 {
		t.Errorf("Expected Y=0x44, got Y=%02X", cpu.Y)
	}
	if cycles != 2 {
		t.Errorf("Expected 2 cycles, got %d", cycles)
	}
}

// Test STA instruction
func TestSTA(t *testing.T) {
	cpu := setupCPUWithProgram([]uint8{0x85, 0x10}) // STA $10
	cpu.A = 0x55

	cpu.Step()

	value := cpu.Memory.Read(0x10)
	if value != 0x55 {
		t.Errorf("Expected memory[0x10]=0x55, got %02X", value)
	}
}

// Test ADC instruction
func TestADC(t *testing.T) {
	// Test basic addition
	cpu := setupCPUWithProgram([]uint8{0x69, 0x10}) // ADC #$10
	cpu.A = 0x20

	cpu.Step()

	if cpu.A != 0x30 {
		t.Errorf("Expected A=0x30, got A=%02X", cpu.A)
	}
	if cpu.getFlag(FlagCarry) {
		t.Error("Carry flag should not be set")
	}

	// Test carry generation
	cpu = setupCPUWithProgram([]uint8{0x69, 0x80}) // ADC #$80
	cpu.A = 0x80

	cpu.Step()

	if cpu.A != 0x00 {
		t.Errorf("Expected A=0x00, got A=%02X", cpu.A)
	}
	if !cpu.getFlag(FlagCarry) {
		t.Error("Carry flag should be set")
	}
	if !cpu.getFlag(FlagZero) {
		t.Error("Zero flag should be set")
	}

	// Test overflow flag
	cpu = setupCPUWithProgram([]uint8{0x69, 0x01}) // ADC #$01
	cpu.A = 0x7F

	cpu.Step()

	if cpu.A != 0x80 {
		t.Errorf("Expected A=0x80, got A=%02X", cpu.A)
	}
	if !cpu.getFlag(FlagOverflow) {
		t.Error("Overflow flag should be set")
	}
	if !cpu.getFlag(FlagNegative) {
		t.Error("Negative flag should be set")
	}
}

// Test SBC instruction
func TestSBC(t *testing.T) {
	// Test basic subtraction
	cpu := setupCPUWithProgram([]uint8{0xE9, 0x10}) // SBC #$10
	cpu.A = 0x30
	cpu.setFlag(FlagCarry, true) // Set carry for normal subtraction

	cpu.Step()

	if cpu.A != 0x20 {
		t.Errorf("Expected A=0x20, got A=%02X", cpu.A)
	}
	if !cpu.getFlag(FlagCarry) {
		t.Error("Carry flag should be set (no borrow)")
	}
}

// Test CMP instruction
func TestCMP(t *testing.T) {
	// Test A > operand
	cpu := setupCPUWithProgram([]uint8{0xC9, 0x10}) // CMP #$10
	cpu.A = 0x20

	cpu.Step()

	if !cpu.getFlag(FlagCarry) {
		t.Error("Carry flag should be set (A >= operand)")
	}
	if cpu.getFlag(FlagZero) {
		t.Error("Zero flag should not be set")
	}

	// Test A == operand
	cpu = setupCPUWithProgram([]uint8{0xC9, 0x20}) // CMP #$20
	cpu.A = 0x20

	cpu.Step()

	if !cpu.getFlag(FlagCarry) {
		t.Error("Carry flag should be set (A >= operand)")
	}
	if !cpu.getFlag(FlagZero) {
		t.Error("Zero flag should be set (A == operand)")
	}
}

// Test transfer instructions
func TestTransferInstructions(t *testing.T) {
	cpu := setupCPUWithProgram([]uint8{0xAA}) // TAX
	cpu.A = 0x42

	cpu.Step()

	if cpu.X != 0x42 {
		t.Errorf("Expected X=0x42, got X=%02X", cpu.X)
	}

	// Test TXA
	cpu = setupCPUWithProgram([]uint8{0x8A}) // TXA
	cpu.X = 0x33
	cpu.A = 0x00

	cpu.Step()

	if cpu.A != 0x33 {
		t.Errorf("Expected A=0x33, got A=%02X", cpu.A)
	}
}

// Test flag instructions
func TestFlagInstructions(t *testing.T) {
	cpu := setupCPUWithProgram([]uint8{0x18}) // CLC
	cpu.setFlag(FlagCarry, true)

	cpu.Step()

	if cpu.getFlag(FlagCarry) {
		t.Error("Carry flag should be cleared")
	}

	// Test SEC
	cpu = setupCPUWithProgram([]uint8{0x38}) // SEC
	cpu.setFlag(FlagCarry, false)

	cpu.Step()

	if !cpu.getFlag(FlagCarry) {
		t.Error("Carry flag should be set")
	}
}

// Test stack instructions
func TestStackInstructions(t *testing.T) {
	cpu := setupCPUWithProgram([]uint8{0x48, 0x68}) // PHA, PLA
	cpu.A = 0x55
	initialSP := cpu.SP

	// Test PHA
	cpu.Step()

	if cpu.SP != initialSP-1 {
		t.Errorf("Expected SP=%02X, got SP=%02X", initialSP-1, cpu.SP)
	}

	// Test PLA
	cpu.A = 0x00
	cpu.Step()

	if cpu.A != 0x55 {
		t.Errorf("Expected A=0x55, got A=%02X", cpu.A)
	}
	if cpu.SP != initialSP {
		t.Errorf("Expected SP=%02X, got SP=%02X", initialSP, cpu.SP)
	}
}

// Test branch instructions - BEQ/BNE
func TestBranchEQ(t *testing.T) {
	// Test BEQ taken
	cpu := setupCPUWithProgram([]uint8{0xF0, 0x05}) // BEQ +5
	cpu.setFlag(FlagZero, true)
	initialPC := cpu.PC

	cycles := cpu.Step()

	expectedPC := initialPC + 2 + 5 // PC after instruction + offset
	if cpu.PC != expectedPC {
		t.Errorf("Expected PC=%04X, got PC=%04X", expectedPC, cpu.PC)
	}
	if cycles != 3 {
		t.Errorf("Expected 3 cycles for taken branch, got %d", cycles)
	}

	// Test BEQ not taken
	cpu = setupCPUWithProgram([]uint8{0xF0, 0x05}) // BEQ +5
	cpu.setFlag(FlagZero, false)
	initialPC = cpu.PC

	cycles = cpu.Step()

	expectedPC = initialPC + 2 // PC after instruction only
	if cpu.PC != expectedPC {
		t.Errorf("Expected PC=%04X, got PC=%04X", expectedPC, cpu.PC)
	}
	if cycles != 2 {
		t.Errorf("Expected 2 cycles for not taken branch, got %d", cycles)
	}

	// Test BNE taken
	cpu = setupCPUWithProgram([]uint8{0xD0, 0x03}) // BNE +3
	cpu.setFlag(FlagZero, false)
	initialPC = cpu.PC

	cycles = cpu.Step()

	expectedPC = initialPC + 2 + 3
	if cpu.PC != expectedPC {
		t.Errorf("Expected PC=%04X, got PC=%04X", expectedPC, cpu.PC)
	}
	if cycles != 3 {
		t.Errorf("Expected 3 cycles for taken branch, got %d", cycles)
	}
}

// Test branch instructions - BCC/BCS
func TestBranchCarry(t *testing.T) {
	// Test BCC taken (branch if carry clear)
	cpu := setupCPUWithProgram([]uint8{0x90, 0x10}) // BCC +16
	cpu.setFlag(FlagCarry, false)
	initialPC := cpu.PC

	cycles := cpu.Step()

	expectedPC := initialPC + 2 + 16
	if cpu.PC != expectedPC {
		t.Errorf("Expected PC=%04X, got PC=%04X", expectedPC, cpu.PC)
	}
	if cycles != 3 {
		t.Errorf("Expected 3 cycles for taken branch, got %d", cycles)
	}

	// Test BCS taken (branch if carry set)
	cpu = setupCPUWithProgram([]uint8{0xB0, 0x08}) // BCS +8
	cpu.setFlag(FlagCarry, true)
	initialPC = cpu.PC

	cycles = cpu.Step()

	expectedPC = initialPC + 2 + 8
	if cpu.PC != expectedPC {
		t.Errorf("Expected PC=%04X, got PC=%04X", expectedPC, cpu.PC)
	}
	if cycles != 3 {
		t.Errorf("Expected 3 cycles for taken branch, got %d", cycles)
	}
}

// Test branch instructions - BPL/BMI
func TestBranchSign(t *testing.T) {
	// Test BPL taken (branch if positive)
	cpu := setupCPUWithProgram([]uint8{0x10, 0x0A}) // BPL +10
	cpu.setFlag(FlagNegative, false)
	initialPC := cpu.PC

	cycles := cpu.Step()

	expectedPC := initialPC + 2 + 10
	if cpu.PC != expectedPC {
		t.Errorf("Expected PC=%04X, got PC=%04X", expectedPC, cpu.PC)
	}
	if cycles != 3 {
		t.Errorf("Expected 3 cycles for taken branch, got %d", cycles)
	}

	// Test BMI taken (branch if minus)
	cpu = setupCPUWithProgram([]uint8{0x30, 0x0C}) // BMI +12
	cpu.setFlag(FlagNegative, true)
	initialPC = cpu.PC

	cycles = cpu.Step()

	expectedPC = initialPC + 2 + 12
	if cpu.PC != expectedPC {
		t.Errorf("Expected PC=%04X, got PC=%04X", expectedPC, cpu.PC)
	}
	if cycles != 3 {
		t.Errorf("Expected 3 cycles for taken branch, got %d", cycles)
	}
}

// Test branch instructions - BVC/BVS
func TestBranchOverflow(t *testing.T) {
	// Test BVC taken (branch if overflow clear)
	cpu := setupCPUWithProgram([]uint8{0x50, 0x06}) // BVC +6
	cpu.setFlag(FlagOverflow, false)
	initialPC := cpu.PC

	cycles := cpu.Step()

	expectedPC := initialPC + 2 + 6
	if cpu.PC != expectedPC {
		t.Errorf("Expected PC=%04X, got PC=%04X", expectedPC, cpu.PC)
	}
	if cycles != 3 {
		t.Errorf("Expected 3 cycles for taken branch, got %d", cycles)
	}

	// Test BVS taken (branch if overflow set)
	cpu = setupCPUWithProgram([]uint8{0x70, 0x04}) // BVS +4
	cpu.setFlag(FlagOverflow, true)
	initialPC = cpu.PC

	cycles = cpu.Step()

	expectedPC = initialPC + 2 + 4
	if cpu.PC != expectedPC {
		t.Errorf("Expected PC=%04X, got PC=%04X", expectedPC, cpu.PC)
	}
	if cycles != 3 {
		t.Errorf("Expected 3 cycles for taken branch, got %d", cycles)
	}
}

// Test branch with negative offset
func TestBranchNegativeOffset(t *testing.T) {
	// Test backward branch without page crossing
	cpu := createTestCPU()
	cpu.PC = 0x0210
	cpu.Memory.Write(0x0210, 0xF0) // BEQ
	cpu.Memory.Write(0x0211, 0xFC) // -4
	cpu.setFlag(FlagZero, true)

	cycles := cpu.Step()

	expectedPC := uint16(0x0212 - 4) // PC after instruction + negative offset = 0x020E
	if cpu.PC != expectedPC {
		t.Errorf("Expected PC=%04X, got PC=%04X", expectedPC, cpu.PC)
	}
	// This should not cross page boundary (both 0x0212 and 0x020E are in page 2)
	if cycles != 3 {
		t.Errorf("Expected 3 cycles for taken branch, got %d", cycles)
	}
}

// Test branch page crossing
func TestBranchPageCrossing(t *testing.T) {
	// Set up CPU so that branch will cross page boundary
	cpu := createTestCPU()
	cpu.PC = 0x02FE
	cpu.Memory.Write(0x02FE, 0xF0) // BEQ
	cpu.Memory.Write(0x02FF, 0x04) // +4: PC after reading=0x300, +4=0x304 (same page)
	cpu.setFlag(FlagZero, true)

	cycles := cpu.Step()

	// This should NOT cross page (0x0300 -> 0x0304, both page 3)
	if cycles != 3 {
		t.Errorf("Expected 3 cycles for same-page branch, got %d", cycles)
	}

	// Test actual page crossing: from 0x02FE, branch to go beyond page boundary
	cpu = createTestCPU()
	cpu.PC = 0x02F0
	cpu.Memory.Write(0x02F0, 0xF0) // BEQ
	cpu.Memory.Write(0x02F1, 0x20) // +32: PC after reading=0x02F2, +32=0x0312 (crosses page)
	cpu.setFlag(FlagZero, true)

	cycles = cpu.Step()

	expectedPC := uint16(0x02F2 + 0x20) // 0x0312
	if cpu.PC != expectedPC {
		t.Errorf("Expected PC=%04X, got PC=%04X", expectedPC, cpu.PC)
	}
	if cycles != 4 {
		t.Errorf("Expected 4 cycles for page-crossing branch, got %d", cycles)
	}
}

// Test JMP absolute
func TestJMPAbsolute(t *testing.T) {
	cpu := setupCPUWithProgram([]uint8{0x4C, 0x34, 0x12}) // JMP $1234

	cycles := cpu.Step()

	if cpu.PC != 0x1234 {
		t.Errorf("Expected PC=1234, got PC=%04X", cpu.PC)
	}
	if cycles != 3 {
		t.Errorf("Expected 3 cycles for JMP absolute, got %d", cycles)
	}
}

// Test JMP indirect
func TestJMPIndirect(t *testing.T) {
	// Set up indirect jump
	cpu := createTestCPU()
	cpu.PC = 0x0200
	cpu.Memory.Write(0x0200, 0x6C) // JMP indirect
	cpu.Memory.Write(0x0201, 0x10) // Low byte of indirect address
	cpu.Memory.Write(0x0202, 0x03) // High byte of indirect address ($0310)

	// Set up target address at $0310
	cpu.Memory.Write(0x0310, 0x34) // Low byte of target
	cpu.Memory.Write(0x0311, 0x12) // High byte of target ($1234)

	cycles := cpu.Step()

	if cpu.PC != 0x1234 {
		t.Errorf("Expected PC=1234, got PC=%04X", cpu.PC)
	}
	if cycles != 5 {
		t.Errorf("Expected 5 cycles for JMP indirect, got %d", cycles)
	}
}

// Test JMP indirect page boundary bug
func TestJMPIndirectBug(t *testing.T) {
	// Set up indirect jump with page boundary bug
	cpu := createTestCPU()
	cpu.PC = 0x0200
	cpu.Memory.Write(0x0200, 0x6C) // JMP indirect
	cpu.Memory.Write(0x0201, 0xFF) // Low byte of indirect address
	cpu.Memory.Write(0x0202, 0x03) // High byte of indirect address ($03FF)

	// Set up target address - bug causes high byte to be read from $0300 instead of $0400
	cpu.Memory.Write(0x03FF, 0x34) // Low byte of target
	cpu.Memory.Write(0x0300, 0x12) // High byte read due to bug (should be $0400)
	cpu.Memory.Write(0x0400, 0x56) // This should be read but won't be due to bug

	cycles := cpu.Step()

	// Should jump to $1234, not $5634
	if cpu.PC != 0x1234 {
		t.Errorf("Expected PC=1234 (due to page boundary bug), got PC=%04X", cpu.PC)
	}
	if cycles != 5 {
		t.Errorf("Expected 5 cycles for JMP indirect, got %d", cycles)
	}
}

// Test JSR and RTS
func TestJSRRTS(t *testing.T) {
	cpu := createTestCPU()
	cpu.PC = 0x0200
	initialSP := cpu.SP

	// Set up JSR
	cpu.Memory.Write(0x0200, 0x20) // JSR
	cpu.Memory.Write(0x0201, 0x34) // Low byte of target
	cpu.Memory.Write(0x0202, 0x12) // High byte of target ($1234)

	// Set up RTS at target
	cpu.Memory.Write(0x1234, 0x60) // RTS

	// Execute JSR
	cycles := cpu.Step()

	if cpu.PC != 0x1234 {
		t.Errorf("Expected PC=1234 after JSR, got PC=%04X", cpu.PC)
	}
	if cycles != 6 {
		t.Errorf("Expected 6 cycles for JSR, got %d", cycles)
	}
	if cpu.SP != initialSP-2 {
		t.Errorf("Expected SP=%02X after JSR, got SP=%02X", initialSP-2, cpu.SP)
	}

	// Execute RTS
	cycles = cpu.Step()

	expectedPC := uint16(0x0203) // Return to instruction after JSR
	if cpu.PC != expectedPC {
		t.Errorf("Expected PC=%04X after RTS, got PC=%04X", expectedPC, cpu.PC)
	}
	if cycles != 6 {
		t.Errorf("Expected 6 cycles for RTS, got %d", cycles)
	}
	if cpu.SP != initialSP {
		t.Errorf("Expected SP=%02X after RTS, got SP=%02X", initialSP, cpu.SP)
	}
}

// Test AND instruction
func TestAND(t *testing.T) {
	cpu := setupCPUWithProgram([]uint8{0x29, 0x0F}) // AND #$0F
	cpu.A = 0xFF

	cycles := cpu.Step()

	if cpu.A != 0x0F {
		t.Errorf("Expected A=0F, got A=%02X", cpu.A)
	}
	if cpu.getFlag(FlagZero) {
		t.Error("Zero flag should not be set")
	}
	if cpu.getFlag(FlagNegative) {
		t.Error("Negative flag should not be set")
	}
	if cycles != 2 {
		t.Errorf("Expected 2 cycles for AND immediate, got %d", cycles)
	}

	// Test zero result
	cpu = setupCPUWithProgram([]uint8{0x29, 0x00}) // AND #$00
	cpu.A = 0xFF

	cpu.Step()

	if cpu.A != 0x00 {
		t.Errorf("Expected A=00, got A=%02X", cpu.A)
	}
	if !cpu.getFlag(FlagZero) {
		t.Error("Zero flag should be set")
	}
}

// Test ORA instruction
func TestORA(t *testing.T) {
	cpu := setupCPUWithProgram([]uint8{0x09, 0x0F}) // ORA #$0F
	cpu.A = 0xF0

	cycles := cpu.Step()

	if cpu.A != 0xFF {
		t.Errorf("Expected A=FF, got A=%02X", cpu.A)
	}
	if cpu.getFlag(FlagZero) {
		t.Error("Zero flag should not be set")
	}
	if !cpu.getFlag(FlagNegative) {
		t.Error("Negative flag should be set")
	}
	if cycles != 2 {
		t.Errorf("Expected 2 cycles for ORA immediate, got %d", cycles)
	}

	// Test zero result
	cpu = setupCPUWithProgram([]uint8{0x09, 0x00}) // ORA #$00
	cpu.A = 0x00

	cpu.Step()

	if cpu.A != 0x00 {
		t.Errorf("Expected A=00, got A=%02X", cpu.A)
	}
	if !cpu.getFlag(FlagZero) {
		t.Error("Zero flag should be set")
	}
}

// Test EOR instruction
func TestEOR(t *testing.T) {
	cpu := setupCPUWithProgram([]uint8{0x49, 0xFF}) // EOR #$FF
	cpu.A = 0xAA

	cycles := cpu.Step()

	if cpu.A != 0x55 {
		t.Errorf("Expected A=55, got A=%02X", cpu.A)
	}
	if cpu.getFlag(FlagZero) {
		t.Error("Zero flag should not be set")
	}
	if cpu.getFlag(FlagNegative) {
		t.Error("Negative flag should not be set")
	}
	if cycles != 2 {
		t.Errorf("Expected 2 cycles for EOR immediate, got %d", cycles)
	}

	// Test XOR with same value (should be zero)
	cpu = setupCPUWithProgram([]uint8{0x49, 0xAA}) // EOR #$AA
	cpu.A = 0xAA

	cpu.Step()

	if cpu.A != 0x00 {
		t.Errorf("Expected A=00, got A=%02X", cpu.A)
	}
	if !cpu.getFlag(FlagZero) {
		t.Error("Zero flag should be set")
	}
}

// Test ASL instruction
func TestASL(t *testing.T) {
	// Test ASL accumulator
	cpu := setupCPUWithProgram([]uint8{0x0A}) // ASL A
	cpu.A = 0x40
	cpu.setFlag(FlagCarry, false)

	cycles := cpu.Step()

	if cpu.A != 0x80 {
		t.Errorf("Expected A=80, got A=%02X", cpu.A)
	}
	if cpu.getFlag(FlagCarry) {
		t.Error("Carry flag should not be set")
	}
	if !cpu.getFlag(FlagNegative) {
		t.Error("Negative flag should be set")
	}
	if cycles != 2 {
		t.Errorf("Expected 2 cycles for ASL accumulator, got %d", cycles)
	}

	// Test carry flag
	cpu = setupCPUWithProgram([]uint8{0x0A}) // ASL A
	cpu.A = 0x80

	cpu.Step()

	if cpu.A != 0x00 {
		t.Errorf("Expected A=00, got A=%02X", cpu.A)
	}
	if !cpu.getFlag(FlagCarry) {
		t.Error("Carry flag should be set")
	}
	if !cpu.getFlag(FlagZero) {
		t.Error("Zero flag should be set")
	}
}

// Test LSR instruction
func TestLSR(t *testing.T) {
	// Test LSR accumulator
	cpu := setupCPUWithProgram([]uint8{0x4A}) // LSR A
	cpu.A = 0x81
	cpu.setFlag(FlagCarry, false)

	cycles := cpu.Step()

	if cpu.A != 0x40 {
		t.Errorf("Expected A=40, got A=%02X", cpu.A)
	}
	if !cpu.getFlag(FlagCarry) {
		t.Error("Carry flag should be set")
	}
	if cpu.getFlag(FlagNegative) {
		t.Error("Negative flag should not be set")
	}
	if cycles != 2 {
		t.Errorf("Expected 2 cycles for LSR accumulator, got %d", cycles)
	}

	// Test zero result
	cpu = setupCPUWithProgram([]uint8{0x4A}) // LSR A
	cpu.A = 0x01

	cpu.Step()

	if cpu.A != 0x00 {
		t.Errorf("Expected A=00, got A=%02X", cpu.A)
	}
	if !cpu.getFlag(FlagCarry) {
		t.Error("Carry flag should be set")
	}
	if !cpu.getFlag(FlagZero) {
		t.Error("Zero flag should be set")
	}
}

// Test ROL instruction
func TestROL(t *testing.T) {
	// Test ROL accumulator without carry
	cpu := setupCPUWithProgram([]uint8{0x2A}) // ROL A
	cpu.A = 0x40
	cpu.setFlag(FlagCarry, false)

	cycles := cpu.Step()

	if cpu.A != 0x80 {
		t.Errorf("Expected A=80, got A=%02X", cpu.A)
	}
	if cpu.getFlag(FlagCarry) {
		t.Error("Carry flag should not be set")
	}
	if !cpu.getFlag(FlagNegative) {
		t.Error("Negative flag should be set")
	}
	if cycles != 2 {
		t.Errorf("Expected 2 cycles for ROL accumulator, got %d", cycles)
	}

	// Test ROL with carry input
	cpu = setupCPUWithProgram([]uint8{0x2A}) // ROL A
	cpu.A = 0x40
	cpu.setFlag(FlagCarry, true)

	cpu.Step()

	if cpu.A != 0x81 {
		t.Errorf("Expected A=81, got A=%02X", cpu.A)
	}
	if cpu.getFlag(FlagCarry) {
		t.Error("Carry flag should not be set")
	}
	if !cpu.getFlag(FlagNegative) {
		t.Error("Negative flag should be set")
	}
}

// Test ROR instruction
func TestROR(t *testing.T) {
	// Test ROR accumulator without carry
	cpu := setupCPUWithProgram([]uint8{0x6A}) // ROR A
	cpu.A = 0x02
	cpu.setFlag(FlagCarry, false)

	cycles := cpu.Step()

	if cpu.A != 0x01 {
		t.Errorf("Expected A=01, got A=%02X", cpu.A)
	}
	if cpu.getFlag(FlagCarry) {
		t.Error("Carry flag should not be set")
	}
	if cpu.getFlag(FlagNegative) {
		t.Error("Negative flag should not be set")
	}
	if cycles != 2 {
		t.Errorf("Expected 2 cycles for ROR accumulator, got %d", cycles)
	}

	// Test ROR with carry input
	cpu = setupCPUWithProgram([]uint8{0x6A}) // ROR A
	cpu.A = 0x02
	cpu.setFlag(FlagCarry, true)

	cpu.Step()

	if cpu.A != 0x81 {
		t.Errorf("Expected A=81, got A=%02X", cpu.A)
	}
	if cpu.getFlag(FlagCarry) {
		t.Error("Carry flag should not be set")
	}
	if !cpu.getFlag(FlagNegative) {
		t.Error("Negative flag should be set")
	}
}

// Test shift operations on memory
func TestShiftMemory(t *testing.T) {
	// Test ASL zeropage
	cpu := createTestCPU()
	cpu.PC = 0x0200
	cpu.Memory.Write(0x0200, 0x06) // ASL $10
	cpu.Memory.Write(0x0201, 0x10)
	cpu.Memory.Write(0x0010, 0x40)

	cycles := cpu.Step()

	if cpu.Memory.Read(0x0010) != 0x80 {
		t.Errorf("Expected memory[0x10]=80, got %02X", cpu.Memory.Read(0x0010))
	}
	if cycles != 5 {
		t.Errorf("Expected 5 cycles for ASL zeropage, got %d", cycles)
	}
}

// Test INC/DEC instructions
func TestIncDec(t *testing.T) {
	// Test INX
	cpu := setupCPUWithProgram([]uint8{0xE8}) // INX
	cpu.X = 0x42

	cycles := cpu.Step()

	if cpu.X != 0x43 {
		t.Errorf("Expected X=43, got X=%02X", cpu.X)
	}
	if cycles != 2 {
		t.Errorf("Expected 2 cycles for INX, got %d", cycles)
	}

	// Test DEY
	cpu = setupCPUWithProgram([]uint8{0x88}) // DEY
	cpu.Y = 0x01

	cycles = cpu.Step()

	if cpu.Y != 0x00 {
		t.Errorf("Expected Y=00, got Y=%02X", cpu.Y)
	}
	if !cpu.getFlag(FlagZero) {
		t.Error("Zero flag should be set")
	}
	if cycles != 2 {
		t.Errorf("Expected 2 cycles for DEY, got %d", cycles)
	}
}

// Test CPX/CPY instructions
func TestCPXCPY(t *testing.T) {
	// Test CPX immediate
	cpu := setupCPUWithProgram([]uint8{0xE0, 0x42}) // CPX #$42
	cpu.X = 0x42

	cycles := cpu.Step()

	if !cpu.getFlag(FlagZero) {
		t.Error("Zero flag should be set when X == operand")
	}
	if !cpu.getFlag(FlagCarry) {
		t.Error("Carry flag should be set when X >= operand")
	}
	if cycles != 2 {
		t.Errorf("Expected 2 cycles for CPX immediate, got %d", cycles)
	}
}

// Test BIT instruction
func TestBIT(t *testing.T) {
	// Test BIT zeropage
	cpu := setupCPUWithProgram([]uint8{0x24, 0x10}) // BIT $10
	cpu.A = 0x0F
	cpu.Memory.Write(0x0010, 0xC0) // Bits 7 and 6 set

	cycles := cpu.Step()

	if !cpu.getFlag(FlagZero) {
		t.Error("Zero flag should be set (A & memory = 0)")
	}
	if !cpu.getFlag(FlagNegative) {
		t.Error("Negative flag should be set (bit 7 of memory)")
	}
	if !cpu.getFlag(FlagOverflow) {
		t.Error("Overflow flag should be set (bit 6 of memory)")
	}
	if cycles != 3 {
		t.Errorf("Expected 3 cycles for BIT zeropage, got %d", cycles)
	}
}
