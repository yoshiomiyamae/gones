package cpu

import "testing"

// runImm loads a single immediate-mode opcode with its operand and steps once.
func runImm(c *CPU, opcode, operand uint8) {
	c.PC = 0x0200
	c.Memory.Write(0x0200, opcode)
	c.Memory.Write(0x0201, operand)
	c.Step()
}

func TestIllegalImmediateOpcodes(t *testing.T) {
	// AAC/ANC (0x0B): A &= imm; carry = bit 7 of result.
	c := createTestCPU()
	c.A = 0xFF
	runImm(c, 0x0B, 0x80)
	if c.A != 0x80 || !c.getFlag(FlagCarry) || !c.getFlag(FlagNegative) {
		t.Errorf("AAC: A=%#02x C=%v N=%v, want 0x80/true/true", c.A, c.getFlag(FlagCarry), c.getFlag(FlagNegative))
	}
	// The $2B alias dispatches to the same handler.
	c = createTestCPU()
	c.A = 0x0F
	runImm(c, 0x2B, 0x01)
	if c.A != 0x01 || c.getFlag(FlagCarry) {
		t.Errorf("AAC($2B): A=%#02x C=%v, want 0x01/false", c.A, c.getFlag(FlagCarry))
	}

	// ASR/ALR (0x4B): A &= imm; then LSR.
	c = createTestCPU()
	c.A = 0x03
	runImm(c, 0x4B, 0x03)
	if c.A != 0x01 || !c.getFlag(FlagCarry) {
		t.Errorf("ASR: A=%#02x C=%v, want 0x01/true", c.A, c.getFlag(FlagCarry))
	}

	// ARR (0x6B): A &= imm; ROR; then the defining quirk — C = bit6 of the
	// result, V = bit6 ^ bit5. With A=0xFF, imm=0xFF, C=0: result 0x7F, so
	// C = 1 (bit6 set) and V = 1^1 = 0.
	c = createTestCPU()
	c.A = 0xFF
	c.setFlag(FlagCarry, false)
	runImm(c, 0x6B, 0xFF)
	if c.A != 0x7F || !c.getFlag(FlagCarry) || c.getFlag(FlagOverflow) {
		t.Errorf("ARR: A=%#02x C=%v V=%v, want 0x7F/true/false",
			c.A, c.getFlag(FlagCarry), c.getFlag(FlagOverflow))
	}

	// ATX/LXA (0xAB): A = X = imm.
	c = createTestCPU()
	runImm(c, 0xAB, 0x55)
	if c.A != 0x55 || c.X != 0x55 {
		t.Errorf("ATX: A=%#02x X=%#02x, want 0x55/0x55", c.A, c.X)
	}

	// AXS/SBX (0xCB): X = (A & X) - imm; carry set when no borrow.
	c = createTestCPU()
	c.A = 0xFF
	c.X = 0x0F
	runImm(c, 0xCB, 0x01)
	if c.X != 0x0E || !c.getFlag(FlagCarry) {
		t.Errorf("AXS: X=%#02x C=%v, want 0x0E/true", c.X, c.getFlag(FlagCarry))
	}
	// Borrow case: (A&X) < imm sets X to a wrapped value and clears carry.
	c = createTestCPU()
	c.A = 0x01
	c.X = 0x01
	runImm(c, 0xCB, 0x05)
	if c.getFlag(FlagCarry) {
		t.Error("AXS borrow: carry should be clear")
	}
}

// TestGetOperandAddressModes drives getOperandAddress through the addressing
// modes the test ROM corpus rarely exercises directly: relative branches, the
// JMP-indirect page-boundary bug, indexed page-cross dummy reads, and the
// no-match default.
func TestGetOperandAddressModes(t *testing.T) {
	c := createTestCPU()

	// Relative: PC after the operand byte + signed offset.
	c.PC = 0x0200
	c.Memory.Write(0x0200, 0x10) // +16; PC becomes 0x0201 then +16 = 0x0211
	if addr, _ := c.getOperandAddress(AddrRelative); addr != 0x0211 {
		t.Errorf("AddrRelative = %#04x, want 0x0211", addr)
	}

	// Indirect, normal pointer: [$0300] = $1234.
	c.PC = 0x0200
	c.Memory.Write(0x0200, 0x00)
	c.Memory.Write(0x0201, 0x03)
	c.Memory.Write(0x0300, 0x34)
	c.Memory.Write(0x0301, 0x12)
	if addr, _ := c.getOperandAddress(AddrIndirect); addr != 0x1234 {
		t.Errorf("AddrIndirect = %#04x, want 0x1234", addr)
	}

	// Indirect $xxFF page-wrap bug: high byte read from $0200, not $0300.
	c.PC = 0x0200
	c.Memory.Write(0x0200, 0xFF) // ptr lo; also serves as the wrapped high byte
	c.Memory.Write(0x0201, 0x02) // ptr = $02FF
	c.Memory.Write(0x02FF, 0x78) // lo of result
	// hi = read($0200) = 0xFF  ->  addr = 0xFF78 (the bug; correct HW would read $0300)
	if addr, _ := c.getOperandAddress(AddrIndirect); addr != 0xFF78 {
		t.Errorf("AddrIndirect page-wrap = %#04x, want 0xFF78", addr)
	}

	// Absolute,X crossing a page boundary: base $0201 + X 0xFF = $0300.
	c.PC, c.X = 0x0200, 0xFF
	c.Memory.Write(0x0200, 0x01)
	c.Memory.Write(0x0201, 0x02)
	if addr, crossed := c.getOperandAddress(AddrAbsoluteX); addr != 0x0300 || !crossed {
		t.Errorf("AddrAbsoluteX = (%#04x,%v), want (0x0300,true)", addr, crossed)
	}

	// Absolute,Y crossing a page boundary.
	c.PC, c.Y = 0x0200, 0xFF
	c.Memory.Write(0x0200, 0x01)
	c.Memory.Write(0x0201, 0x02)
	if addr, crossed := c.getOperandAddress(AddrAbsoluteY); addr != 0x0300 || !crossed {
		t.Errorf("AddrAbsoluteY = (%#04x,%v), want (0x0300,true)", addr, crossed)
	}

	// (zp),Y crossing a page boundary: [$10]=$0201, +Y 0xFF = $0300.
	c.PC, c.Y = 0x0200, 0xFF
	c.Memory.Write(0x0200, 0x10)
	c.Memory.Write(0x10, 0x01)
	c.Memory.Write(0x11, 0x02)
	if addr, crossed := c.getOperandAddress(AddrIndirectIndexed); addr != 0x0300 || !crossed {
		t.Errorf("AddrIndirectIndexed = (%#04x,%v), want (0x0300,true)", addr, crossed)
	}

	// A mode with no case returns (0, false).
	if addr, pc := c.getOperandAddress(AddrImplied); addr != 0 || pc {
		t.Errorf("AddrImplied = (%#04x,%v), want (0,false)", addr, pc)
	}
}

func TestTriggerIRQ(t *testing.T) {
	c := createTestCPU()
	if c.IRQ {
		t.Fatal("IRQ should start clear")
	}
	c.TriggerIRQ()
	if !c.IRQ {
		t.Error("TriggerIRQ should assert the IRQ line")
	}
}
