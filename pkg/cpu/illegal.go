package cpu

// This file contains implementations of the 6502's illegal/undocumented
// opcodes. They are split out from instructions.go purely for readability;
// each method remains on *CPU and shares package state with the legal
// instruction implementations.

// LAX - Load Accumulator and X register
func (c *CPU) execLAX(mode AddressingMode) int {
	value, pageCrossed := c.getOperand(mode)
	c.A = value
	c.X = value
	c.setZN(value)

	baseCycles := map[AddressingMode]int{
		AddrAbsolute:        4,
		AddrAbsoluteY:       4,
		AddrZeroPage:        3,
		AddrZeroPageY:       4,
		AddrIndexedIndirect: 6,
		AddrIndirectIndexed: 5,
	}[mode]

	if pageCrossed && (mode == AddrAbsoluteY || mode == AddrIndirectIndexed) {
		baseCycles++
	}
	return baseCycles
}

// SAX - Store A AND X
func (c *CPU) execSAX(mode AddressingMode) int {
	addr, _ := c.getOperandAddress(mode)
	result := c.A & c.X
	c.write(addr, result)

	return map[AddressingMode]int{
		AddrAbsolute:        4,
		AddrZeroPage:        3,
		AddrZeroPageY:       4,
		AddrIndexedIndirect: 6,
	}[mode]
}

// DCP - Decrement and Compare
func (c *CPU) execDCP(mode AddressingMode) int {
	addr, _ := c.getOperandAddress(mode)
	value := c.read(addr)
	value--
	c.write(addr, value)

	// Compare with A register
	result := uint16(c.A) - uint16(value)
	c.setFlag(FlagCarry, result < 0x100)
	c.setZN(uint8(result))

	baseCycles := map[AddressingMode]int{
		AddrAbsolute:        6,
		AddrAbsoluteX:       7,
		AddrAbsoluteY:       7,
		AddrZeroPage:        5,
		AddrZeroPageX:       6,
		AddrIndexedIndirect: 8,
		AddrIndirectIndexed: 8,
	}[mode]

	return baseCycles
}

// ISB - Increment and Subtract with Borrow (also known as ISC)
func (c *CPU) execISB(mode AddressingMode) int {
	addr, _ := c.getOperandAddress(mode)
	value := c.read(addr)
	value++
	c.write(addr, value)

	// Perform SBC with the incremented value
	c.performSBC(value)

	baseCycles := map[AddressingMode]int{
		AddrAbsolute:        6,
		AddrAbsoluteX:       7,
		AddrAbsoluteY:       7,
		AddrZeroPage:        5,
		AddrZeroPageX:       6,
		AddrIndexedIndirect: 8,
		AddrIndirectIndexed: 8,
	}[mode]

	return baseCycles
}

// SLO - Shift Left and OR
func (c *CPU) execSLO(mode AddressingMode) int {
	addr, _ := c.getOperandAddress(mode)
	value := c.read(addr)

	// Shift left
	c.setFlag(FlagCarry, value&0x80 != 0)
	value <<= 1
	c.write(addr, value)

	// OR with A
	c.A |= value
	c.setZN(c.A)

	baseCycles := map[AddressingMode]int{
		AddrAbsolute:        6,
		AddrAbsoluteX:       7,
		AddrAbsoluteY:       7,
		AddrZeroPage:        5,
		AddrZeroPageX:       6,
		AddrIndexedIndirect: 8,
		AddrIndirectIndexed: 8,
	}[mode]

	return baseCycles
}

// RLA - Rotate Left and AND
func (c *CPU) execRLA(mode AddressingMode) int {
	addr, _ := c.getOperandAddress(mode)
	value := c.read(addr)

	// Rotate left through carry
	newCarry := value&0x80 != 0
	carryBit := uint8(0)
	if c.getFlag(FlagCarry) {
		carryBit = 1
	}
	value = (value << 1) | carryBit
	c.setFlag(FlagCarry, newCarry)
	c.write(addr, value)

	// AND with A
	c.A &= value
	c.setZN(c.A)

	baseCycles := map[AddressingMode]int{
		AddrAbsolute:        6,
		AddrAbsoluteX:       7,
		AddrAbsoluteY:       7,
		AddrZeroPage:        5,
		AddrZeroPageX:       6,
		AddrIndexedIndirect: 8,
		AddrIndirectIndexed: 8,
	}[mode]

	return baseCycles
}

// SRE - Shift Right and EOR
func (c *CPU) execSRE(mode AddressingMode) int {
	addr, _ := c.getOperandAddress(mode)
	value := c.read(addr)

	// Shift right
	c.setFlag(FlagCarry, value&0x01 != 0)
	value >>= 1
	c.write(addr, value)

	// EOR with A
	c.A ^= value
	c.setZN(c.A)

	baseCycles := map[AddressingMode]int{
		AddrAbsolute:        6,
		AddrAbsoluteX:       7,
		AddrAbsoluteY:       7,
		AddrZeroPage:        5,
		AddrZeroPageX:       6,
		AddrIndexedIndirect: 8,
		AddrIndirectIndexed: 8,
	}[mode]

	return baseCycles
}

// RRA - Rotate Right and Add
func (c *CPU) execRRA(mode AddressingMode) int {
	addr, _ := c.getOperandAddress(mode)
	value := c.read(addr)

	// Rotate right through carry
	newCarry := value&0x01 != 0
	carryBit := uint8(0)
	if c.getFlag(FlagCarry) {
		carryBit = 0x80
	}
	value = (value >> 1) | carryBit
	c.setFlag(FlagCarry, newCarry)
	c.write(addr, value)

	// Add to A with carry
	c.performADC(value)

	baseCycles := map[AddressingMode]int{
		AddrAbsolute:        6,
		AddrAbsoluteX:       7,
		AddrAbsoluteY:       7,
		AddrZeroPage:        5,
		AddrZeroPageX:       6,
		AddrIndexedIndirect: 8,
		AddrIndirectIndexed: 8,
	}[mode]

	return baseCycles
}

// Helper function for SBC operation (used by ISB)
func (c *CPU) performSBC(value uint8) {
	// SBC is equivalent to ADC with inverted value
	c.performADC(^value)
}

// Helper function for ADC operation (used by RRA)
func (c *CPU) performADC(value uint8) {
	carryValue := uint16(0)
	if c.getFlag(FlagCarry) {
		carryValue = 1
	}
	result := uint16(c.A) + uint16(value) + carryValue

	// Set overflow flag
	overflow := (c.A^value)&0x80 == 0 && (c.A^uint8(result))&0x80 != 0
	c.setFlag(FlagOverflow, overflow)

	// Set carry flag
	c.setFlag(FlagCarry, result > 0xFF)

	c.A = uint8(result)
	c.setZN(c.A)
}

// AAC - AND accumulator with immediate (also sets carry flag).
// Also known as ANC in some references.
func (c *CPU) execAAC() int {
	value := c.read(c.PC)
	c.PC++

	c.A &= value
	c.setZN(c.A)
	c.setFlag(FlagCarry, c.A&0x80 != 0) // Set carry flag based on bit 7

	return 2
}

// ASR - AND with immediate, then LSR (also known as ALR).
func (c *CPU) execASR() int {
	value := c.read(c.PC)
	c.PC++

	// AND with immediate
	c.A &= value

	// Then LSR (logical shift right)
	c.setFlag(FlagCarry, c.A&0x01 != 0)
	c.A >>= 1
	c.setZN(c.A)

	return 2
}

// ARR - AND with immediate, then ROR
func (c *CPU) execARR() int {
	value := c.read(c.PC)
	c.PC++

	// AND with immediate
	c.A &= value

	// Then ROR (rotate right through carry)
	newCarry := c.A&0x01 != 0
	carryBit := uint8(0)
	if c.getFlag(FlagCarry) {
		carryBit = 0x80
	}
	c.A = (c.A >> 1) | carryBit
	c.setFlag(FlagCarry, newCarry)
	c.setZN(c.A)

	// ARR sets overflow and carry flags in a special way
	// V = bit 6 XOR bit 5 of result
	c.setFlag(FlagOverflow, ((c.A>>6)&1)^((c.A>>5)&1) != 0)
	// C = bit 6 of result
	c.setFlag(FlagCarry, c.A&0x40 != 0)

	return 2
}

// ATX - Load immediate to A and X (also known as LXA).
func (c *CPU) execATX() int {
	value := c.read(c.PC)
	c.PC++

	// ATX (LXA) loads immediate value to both A and X
	// Simple implementation: just load the value
	c.A = value
	c.X = value
	c.setZN(c.A)

	return 2
}

// AXS - AND X with A, then subtract immediate (without borrow).
// Also known as SBX.
func (c *CPU) execAXS() int {
	value := c.read(c.PC)
	c.PC++

	// AND X with A
	temp := c.A & c.X

	// Subtract immediate (without borrow)
	result := uint16(temp) - uint16(value)
	c.X = uint8(result)

	// Set flags
	c.setFlag(FlagCarry, result < 0x100) // Set carry if no borrow
	c.setZN(c.X)

	return 2
}
