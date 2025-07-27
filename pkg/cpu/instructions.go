package cpu

// executeInstruction executes the given opcode
func (c *CPU) executeInstruction(opcode uint8) int {
	switch opcode {
	// LDA - Load Accumulator
	case 0xA9: // LDA #immediate
		return c.execLDAImmediate()
	case 0xA5: // LDA zeropage
		return c.execLDA(AddrZeroPage)
	case 0xB5: // LDA zeropage,X
		return c.execLDA(AddrZeroPageX)
	case 0xAD: // LDA absolute
		return c.execLDA(AddrAbsolute)
	case 0xBD: // LDA absolute,X
		return c.execLDA(AddrAbsoluteX)
	case 0xB9: // LDA absolute,Y
		return c.execLDA(AddrAbsoluteY)
	case 0xA1: // LDA (zeropage,X)
		return c.execLDA(AddrIndexedIndirect)
	case 0xB1: // LDA (zeropage),Y
		return c.execLDA(AddrIndirectIndexed)

	// LDX - Load X Register
	case 0xA2: // LDX #immediate
		return c.execLDX(AddrImmediate)
	case 0xA6: // LDX zeropage
		return c.execLDX(AddrZeroPage)
	case 0xB6: // LDX zeropage,Y
		return c.execLDX(AddrZeroPageY)
	case 0xAE: // LDX absolute
		return c.execLDX(AddrAbsolute)
	case 0xBE: // LDX absolute,Y
		return c.execLDX(AddrAbsoluteY)

	// LDY - Load Y Register
	case 0xA0: // LDY #immediate
		return c.execLDY(AddrImmediate)
	case 0xA4: // LDY zeropage
		return c.execLDY(AddrZeroPage)
	case 0xB4: // LDY zeropage,X
		return c.execLDY(AddrZeroPageX)
	case 0xAC: // LDY absolute
		return c.execLDY(AddrAbsolute)
	case 0xBC: // LDY absolute,X
		return c.execLDY(AddrAbsoluteX)

	// STA - Store Accumulator
	case 0x85: // STA zeropage
		return c.execSTA(AddrZeroPage)
	case 0x95: // STA zeropage,X
		return c.execSTA(AddrZeroPageX)
	case 0x8D: // STA absolute
		return c.execSTA(AddrAbsolute)
	case 0x9D: // STA absolute,X
		return c.execSTA(AddrAbsoluteX)
	case 0x99: // STA absolute,Y
		return c.execSTA(AddrAbsoluteY)
	case 0x81: // STA (zeropage,X)
		return c.execSTA(AddrIndexedIndirect)
	case 0x91: // STA (zeropage),Y
		return c.execSTA(AddrIndirectIndexed)

	// STX - Store X Register
	case 0x86: // STX zeropage
		return c.execSTX(AddrZeroPage)
	case 0x96: // STX zeropage,Y
		return c.execSTX(AddrZeroPageY)
	case 0x8E: // STX absolute
		return c.execSTX(AddrAbsolute)

	// STY - Store Y Register
	case 0x84: // STY zeropage
		return c.execSTY(AddrZeroPage)
	case 0x94: // STY zeropage,X
		return c.execSTY(AddrZeroPageX)
	case 0x8C: // STY absolute
		return c.execSTY(AddrAbsolute)

	// ADC - Add with Carry
	case 0x69: // ADC #immediate
		return c.execADC(AddrImmediate)
	case 0x65: // ADC zeropage
		return c.execADC(AddrZeroPage)
	case 0x75: // ADC zeropage,X
		return c.execADC(AddrZeroPageX)
	case 0x6D: // ADC absolute
		return c.execADC(AddrAbsolute)
	case 0x7D: // ADC absolute,X
		return c.execADC(AddrAbsoluteX)
	case 0x79: // ADC absolute,Y
		return c.execADC(AddrAbsoluteY)
	case 0x61: // ADC (zeropage,X)
		return c.execADC(AddrIndexedIndirect)
	case 0x71: // ADC (zeropage),Y
		return c.execADC(AddrIndirectIndexed)

	// SBC - Subtract with Carry
	case 0xE9: // SBC #immediate
		return c.execSBC(AddrImmediate)
	case 0xE5: // SBC zeropage
		return c.execSBC(AddrZeroPage)
	case 0xF5: // SBC zeropage,X
		return c.execSBC(AddrZeroPageX)
	case 0xED: // SBC absolute
		return c.execSBC(AddrAbsolute)
	case 0xFD: // SBC absolute,X
		return c.execSBC(AddrAbsoluteX)
	case 0xF9: // SBC absolute,Y
		return c.execSBC(AddrAbsoluteY)
	case 0xE1: // SBC (zeropage,X)
		return c.execSBC(AddrIndexedIndirect)
	case 0xF1: // SBC (zeropage),Y
		return c.execSBC(AddrIndirectIndexed)

	// CMP - Compare Accumulator
	case 0xC9: // CMP #immediate
		return c.execCMP(AddrImmediate)
	case 0xC5: // CMP zeropage
		return c.execCMP(AddrZeroPage)
	case 0xD5: // CMP zeropage,X
		return c.execCMP(AddrZeroPageX)
	case 0xCD: // CMP absolute
		return c.execCMP(AddrAbsolute)
	case 0xDD: // CMP absolute,X
		return c.execCMP(AddrAbsoluteX)
	case 0xD9: // CMP absolute,Y
		return c.execCMP(AddrAbsoluteY)
	case 0xC1: // CMP (zeropage,X)
		return c.execCMP(AddrIndexedIndirect)
	case 0xD1: // CMP (zeropage),Y
		return c.execCMP(AddrIndirectIndexed)

	// Transfer instructions
	case 0xAA: // TAX
		return c.execTAX()
	case 0x8A: // TXA
		return c.execTXA()
	case 0xA8: // TAY
		return c.execTAY()
	case 0x98: // TYA
		return c.execTYA()
	case 0x9A: // TXS
		return c.execTXS()
	case 0xBA: // TSX
		return c.execTSX()

	// Flag instructions
	case 0x18: // CLC
		return c.execCLC()
	case 0x38: // SEC
		return c.execSEC()
	case 0x58: // CLI
		return c.execCLI()
	case 0x78: // SEI
		return c.execSEI()
	case 0xB8: // CLV
		return c.execCLV()
	case 0xD8: // CLD
		return c.execCLD()
	case 0xF8: // SED
		return c.execSED()

	// Stack instructions
	case 0x48: // PHA
		return c.execPHA()
	case 0x68: // PLA
		return c.execPLA()
	case 0x08: // PHP
		return c.execPHP()
	case 0x28: // PLP
		return c.execPLP()

	// Branch instructions
	case 0x10: // BPL - Branch if Positive
		return c.execBPL()
	case 0x30: // BMI - Branch if Minus
		return c.execBMI()
	case 0x50: // BVC - Branch if Overflow Clear
		return c.execBVC()
	case 0x70: // BVS - Branch if Overflow Set
		return c.execBVS()
	case 0x90: // BCC - Branch if Carry Clear
		return c.execBCC()
	case 0xB0: // BCS - Branch if Carry Set
		return c.execBCS()
	case 0xD0: // BNE - Branch if Not Equal
		return c.execBNE()
	case 0xF0: // BEQ - Branch if Equal
		return c.execBEQ()

	// Jump instructions
	case 0x4C: // JMP absolute
		return c.execJMPAbsolute()
	case 0x6C: // JMP indirect
		return c.execJMPIndirect()
	case 0x20: // JSR - Jump to Subroutine
		return c.execJSR()
	case 0x60: // RTS - Return from Subroutine
		return c.execRTS()
	case 0x40: // RTI - Return from Interrupt
		return c.execRTI()

	// Logical operations
	case 0x29: // AND #immediate
		return c.execAND(AddrImmediate)
	case 0x25: // AND zeropage
		return c.execAND(AddrZeroPage)
	case 0x35: // AND zeropage,X
		return c.execAND(AddrZeroPageX)
	case 0x2D: // AND absolute
		return c.execAND(AddrAbsolute)
	case 0x3D: // AND absolute,X
		return c.execAND(AddrAbsoluteX)
	case 0x39: // AND absolute,Y
		return c.execAND(AddrAbsoluteY)
	case 0x21: // AND (zeropage,X)
		return c.execAND(AddrIndexedIndirect)
	case 0x31: // AND (zeropage),Y
		return c.execAND(AddrIndirectIndexed)

	case 0x09: // ORA #immediate
		return c.execORA(AddrImmediate)
	case 0x05: // ORA zeropage
		return c.execORA(AddrZeroPage)
	case 0x15: // ORA zeropage,X
		return c.execORA(AddrZeroPageX)
	case 0x0D: // ORA absolute
		return c.execORA(AddrAbsolute)
	case 0x1D: // ORA absolute,X
		return c.execORA(AddrAbsoluteX)
	case 0x19: // ORA absolute,Y
		return c.execORA(AddrAbsoluteY)
	case 0x01: // ORA (zeropage,X)
		return c.execORA(AddrIndexedIndirect)
	case 0x11: // ORA (zeropage),Y
		return c.execORA(AddrIndirectIndexed)

	case 0x49: // EOR #immediate
		return c.execEOR(AddrImmediate)
	case 0x45: // EOR zeropage
		return c.execEOR(AddrZeroPage)
	case 0x55: // EOR zeropage,X
		return c.execEOR(AddrZeroPageX)
	case 0x4D: // EOR absolute
		return c.execEOR(AddrAbsolute)
	case 0x5D: // EOR absolute,X
		return c.execEOR(AddrAbsoluteX)
	case 0x59: // EOR absolute,Y
		return c.execEOR(AddrAbsoluteY)
	case 0x41: // EOR (zeropage,X)
		return c.execEOR(AddrIndexedIndirect)
	case 0x51: // EOR (zeropage),Y
		return c.execEOR(AddrIndirectIndexed)

	// Shift and rotate instructions
	case 0x0A: // ASL accumulator
		return c.execASLAccumulator()
	case 0x06: // ASL zeropage
		return c.execASL(AddrZeroPage)
	case 0x16: // ASL zeropage,X
		return c.execASL(AddrZeroPageX)
	case 0x0E: // ASL absolute
		return c.execASL(AddrAbsolute)
	case 0x1E: // ASL absolute,X
		return c.execASL(AddrAbsoluteX)

	case 0x4A: // LSR accumulator
		return c.execLSRAccumulator()
	case 0x46: // LSR zeropage
		return c.execLSR(AddrZeroPage)
	case 0x56: // LSR zeropage,X
		return c.execLSR(AddrZeroPageX)
	case 0x4E: // LSR absolute
		return c.execLSR(AddrAbsolute)
	case 0x5E: // LSR absolute,X
		return c.execLSR(AddrAbsoluteX)

	case 0x2A: // ROL accumulator
		return c.execROLAccumulator()
	case 0x26: // ROL zeropage
		return c.execROL(AddrZeroPage)
	case 0x36: // ROL zeropage,X
		return c.execROL(AddrZeroPageX)
	case 0x2E: // ROL absolute
		return c.execROL(AddrAbsolute)
	case 0x3E: // ROL absolute,X
		return c.execROL(AddrAbsoluteX)

	case 0x6A: // ROR accumulator
		return c.execRORAccumulator()
	case 0x66: // ROR zeropage
		return c.execROR(AddrZeroPage)
	case 0x76: // ROR zeropage,X
		return c.execROR(AddrZeroPageX)
	case 0x6E: // ROR absolute
		return c.execROR(AddrAbsolute)
	case 0x7E: // ROR absolute,X
		return c.execROR(AddrAbsoluteX)

	// Increment/Decrement instructions
	case 0xE6: // INC zeropage
		return c.execINC(AddrZeroPage)
	case 0xF6: // INC zeropage,X
		return c.execINC(AddrZeroPageX)
	case 0xEE: // INC absolute
		return c.execINC(AddrAbsolute)
	case 0xFE: // INC absolute,X
		return c.execINC(AddrAbsoluteX)

	case 0xC6: // DEC zeropage
		return c.execDEC(AddrZeroPage)
	case 0xD6: // DEC zeropage,X
		return c.execDEC(AddrZeroPageX)
	case 0xCE: // DEC absolute
		return c.execDEC(AddrAbsolute)
	case 0xDE: // DEC absolute,X
		return c.execDEC(AddrAbsoluteX)

	case 0xE8: // INX
		return c.execINX()
	case 0xCA: // DEX
		return c.execDEX()
	case 0xC8: // INY
		return c.execINY()
	case 0x88: // DEY
		return c.execDEY()

	// Compare instructions
	case 0xE0: // CPX #immediate
		return c.execCPX(AddrImmediate)
	case 0xE4: // CPX zeropage
		return c.execCPX(AddrZeroPage)
	case 0xEC: // CPX absolute
		return c.execCPX(AddrAbsolute)

	case 0xC0: // CPY #immediate
		return c.execCPY(AddrImmediate)
	case 0xC4: // CPY zeropage
		return c.execCPY(AddrZeroPage)
	case 0xCC: // CPY absolute
		return c.execCPY(AddrAbsolute)

	// Bit test instruction
	case 0x24: // BIT zeropage
		return c.execBIT(AddrZeroPage)
	case 0x2C: // BIT absolute
		return c.execBIT(AddrAbsolute)

	// Interrupt instructions
	case 0x00: // BRK
		return c.execBRK()

	// NOP - official
	case 0xEA: // NOP
		return c.execNOP()

	// Illegal NOPs (undocumented opcodes that act like NOP)
	case 0x1A, 0x3A, 0x5A, 0x7A, 0xDA, 0xFA: // NOP (implied)
		return c.execNOP()
	case 0x80, 0x82, 0x89, 0xC2, 0xE2: // NOP #imm (immediate)
		c.PC++ // Skip immediate operand
		return 2
	case 0x04, 0x44, 0x64: // NOP zp (zero page)
		c.PC++ // Skip zero page address
		return 3
	case 0x14, 0x34, 0x54, 0x74, 0xD4, 0xF4: // NOP zp,X (zero page,X)
		c.PC++ // Skip zero page address
		return 4
	case 0x0C: // NOP abs (absolute)
		c.PC += 2 // Skip absolute address
		return 4
	case 0x1C, 0x3C, 0x5C, 0x7C, 0xDC, 0xFC: // NOP abs,X (absolute,X)
		c.PC += 2 // Skip absolute address
		return 4  // May be 5 if page boundary crossed, but simplified for now

	// Illegal opcodes that perform actual operations
	// LAX - Load A and X
	case 0xAF: // LAX abs
		return c.execLAX(AddrAbsolute)
	case 0xBF: // LAX abs,Y
		return c.execLAX(AddrAbsoluteY)
	case 0xA7: // LAX zp
		return c.execLAX(AddrZeroPage)
	case 0xB7: // LAX zp,Y
		return c.execLAX(AddrZeroPageY)
	case 0xA3: // LAX (zp,X)
		return c.execLAX(AddrIndexedIndirect)
	case 0xB3: // LAX (zp),Y
		return c.execLAX(AddrIndirectIndexed)

	// SAX - Store A AND X
	case 0x8F: // SAX abs
		return c.execSAX(AddrAbsolute)
	case 0x87: // SAX zp
		return c.execSAX(AddrZeroPage)
	case 0x97: // SAX zp,Y
		return c.execSAX(AddrZeroPageY)
	case 0x83: // SAX (zp,X)
		return c.execSAX(AddrIndexedIndirect)

	// SBC immediate (illegal opcode 0xEB)
	case 0xEB: // SBC #imm (same as 0xE9)
		return c.execSBC(AddrImmediate)

	// AAC - AND accumulator with immediate (same as AND but sets carry)
	case 0x0B, 0x2B: // AAC #imm
		return c.execAAC()

	// ASR - AND with immediate, then LSR
	case 0x4B: // ASR #imm
		return c.execASR()

	// ARR - AND with immediate, then ROR
	case 0x6B: // ARR #imm
		return c.execARR()

	// ATX - AND X register with immediate, transfer to A
	case 0xAB: // ATX #imm
		return c.execATX()

	// AXS - AND X with A, then subtract immediate
	case 0xCB: // AXS #imm
		return c.execAXS()

	// DCP - Decrement and Compare
	case 0xCF: // DCP abs
		return c.execDCP(AddrAbsolute)
	case 0xDF: // DCP abs,X
		return c.execDCP(AddrAbsoluteX)
	case 0xDB: // DCP abs,Y
		return c.execDCP(AddrAbsoluteY)
	case 0xC7: // DCP zp
		return c.execDCP(AddrZeroPage)
	case 0xD7: // DCP zp,X
		return c.execDCP(AddrZeroPageX)
	case 0xC3: // DCP (zp,X)
		return c.execDCP(AddrIndexedIndirect)
	case 0xD3: // DCP (zp),Y
		return c.execDCP(AddrIndirectIndexed)

	// ISB - Increment and Subtract with Borrow
	case 0xEF: // ISB abs
		return c.execISB(AddrAbsolute)
	case 0xFF: // ISB abs,X
		return c.execISB(AddrAbsoluteX)
	case 0xFB: // ISB abs,Y
		return c.execISB(AddrAbsoluteY)
	case 0xE7: // ISB zp
		return c.execISB(AddrZeroPage)
	case 0xF7: // ISB zp,X
		return c.execISB(AddrZeroPageX)
	case 0xE3: // ISB (zp,X)
		return c.execISB(AddrIndexedIndirect)
	case 0xF3: // ISB (zp),Y
		return c.execISB(AddrIndirectIndexed)

	// SLO - Shift Left and OR
	case 0x0F: // SLO abs
		return c.execSLO(AddrAbsolute)
	case 0x1F: // SLO abs,X
		return c.execSLO(AddrAbsoluteX)
	case 0x1B: // SLO abs,Y
		return c.execSLO(AddrAbsoluteY)
	case 0x07: // SLO zp
		return c.execSLO(AddrZeroPage)
	case 0x17: // SLO zp,X
		return c.execSLO(AddrZeroPageX)
	case 0x03: // SLO (zp,X)
		return c.execSLO(AddrIndexedIndirect)
	case 0x13: // SLO (zp),Y
		return c.execSLO(AddrIndirectIndexed)

	// RLA - Rotate Left and AND
	case 0x2F: // RLA abs
		return c.execRLA(AddrAbsolute)
	case 0x3F: // RLA abs,X
		return c.execRLA(AddrAbsoluteX)
	case 0x3B: // RLA abs,Y
		return c.execRLA(AddrAbsoluteY)
	case 0x27: // RLA zp
		return c.execRLA(AddrZeroPage)
	case 0x37: // RLA zp,X
		return c.execRLA(AddrZeroPageX)
	case 0x23: // RLA (zp,X)
		return c.execRLA(AddrIndexedIndirect)
	case 0x33: // RLA (zp),Y
		return c.execRLA(AddrIndirectIndexed)

	// SRE - Shift Right and EOR
	case 0x4F: // SRE abs
		return c.execSRE(AddrAbsolute)
	case 0x5F: // SRE abs,X
		return c.execSRE(AddrAbsoluteX)
	case 0x5B: // SRE abs,Y
		return c.execSRE(AddrAbsoluteY)
	case 0x47: // SRE zp
		return c.execSRE(AddrZeroPage)
	case 0x57: // SRE zp,X
		return c.execSRE(AddrZeroPageX)
	case 0x43: // SRE (zp,X)
		return c.execSRE(AddrIndexedIndirect)
	case 0x53: // SRE (zp),Y
		return c.execSRE(AddrIndirectIndexed)

	// RRA - Rotate Right and Add
	case 0x6F: // RRA abs
		return c.execRRA(AddrAbsolute)
	case 0x7F: // RRA abs,X
		return c.execRRA(AddrAbsoluteX)
	case 0x7B: // RRA abs,Y
		return c.execRRA(AddrAbsoluteY)
	case 0x67: // RRA zp
		return c.execRRA(AddrZeroPage)
	case 0x77: // RRA zp,X
		return c.execRRA(AddrZeroPageX)
	case 0x63: // RRA (zp,X)
		return c.execRRA(AddrIndexedIndirect)
	case 0x73: // RRA (zp),Y
		return c.execRRA(AddrIndirectIndexed)

	default:
		// Unknown instruction - just consume cycles
		return 2
	}
}

// LDA - Load Accumulator
func (c *CPU) execLDA(mode AddressingMode) int {
	value, pageCrossed := c.getOperand(mode)
	c.A = value
	c.setZN(c.A)

	// Return cycles based on addressing mode
	switch mode {
	case AddrImmediate:
		return 2
	case AddrZeroPage:
		return 3
	case AddrZeroPageX:
		return 4
	case AddrAbsolute:
		return 4
	case AddrAbsoluteX, AddrAbsoluteY:
		cycles := 4
		if pageCrossed {
			cycles++
		}
		return cycles
	case AddrIndexedIndirect:
		return 6
	case AddrIndirectIndexed:
		cycles := 5
		if pageCrossed {
			cycles++
		}
		return cycles
	default:
		return 2
	}
}

// execLDAImmediate - LDA immediate mode
func (c *CPU) execLDAImmediate() int {
	return c.execLDA(AddrImmediate)
}

// LDX - Load X Register
func (c *CPU) execLDX(mode AddressingMode) int {
	value, pageCrossed := c.getOperand(mode)
	c.X = value
	c.setZN(c.X)

	// Return cycles based on addressing mode
	switch mode {
	case AddrImmediate:
		return 2
	case AddrZeroPage:
		return 3
	case AddrZeroPageY:
		return 4
	case AddrAbsolute:
		return 4
	case AddrAbsoluteY:
		cycles := 4
		if pageCrossed {
			cycles++
		}
		return cycles
	default:
		return 2
	}
}

// LDY - Load Y Register
func (c *CPU) execLDY(mode AddressingMode) int {
	value, pageCrossed := c.getOperand(mode)
	c.Y = value
	c.setZN(c.Y)

	cycles := getLoadCycles(mode)
	if pageCrossed && (mode == AddrAbsoluteX || mode == AddrIndirectIndexed) {
		cycles++
	}
	return cycles
}

// Helper function to get cycles for load operations
func getLoadCycles(mode AddressingMode) int {
	switch mode {
	case AddrImmediate:
		return 2
	case AddrZeroPage:
		return 3
	case AddrZeroPageX, AddrZeroPageY:
		return 4
	case AddrAbsolute:
		return 4
	case AddrAbsoluteX, AddrAbsoluteY:
		return 4 // +1 if page crossed (handled by caller)
	case AddrIndexedIndirect:
		return 6
	case AddrIndirectIndexed:
		return 5 // +1 if page crossed (handled by caller)
	default:
		return 2
	}
}

// STA - Store Accumulator
func (c *CPU) execSTA(mode AddressingMode) int {
	addr, _ := c.getOperandAddress(mode)
	c.write(addr, c.A)
	return getStoreCycles(mode)
}

// STX - Store X Register
func (c *CPU) execSTX(mode AddressingMode) int {
	addr, _ := c.getOperandAddress(mode)
	c.write(addr, c.X)
	return getStoreCycles(mode)
}

// STY - Store Y Register
func (c *CPU) execSTY(mode AddressingMode) int {
	addr, _ := c.getOperandAddress(mode)
	c.write(addr, c.Y)
	return getStoreCycles(mode)
}

// Helper function to get cycles for store operations
func getStoreCycles(mode AddressingMode) int {
	switch mode {
	case AddrZeroPage:
		return 3
	case AddrZeroPageX, AddrZeroPageY:
		return 4
	case AddrAbsolute:
		return 4
	case AddrAbsoluteX, AddrAbsoluteY:
		return 5
	case AddrIndexedIndirect:
		return 6
	case AddrIndirectIndexed:
		return 6
	default:
		return 3
	}
}

// ADC - Add with Carry
func (c *CPU) execADC(mode AddressingMode) int {
	value, pageCrossed := c.getOperand(mode)

	carry := uint8(0)
	if c.getFlag(FlagCarry) {
		carry = 1
	}

	// NES CPU (2A03/2A07) does not support decimal mode
	// Always use binary mode
	result := uint16(c.A) + uint16(value) + uint16(carry)

	// Set flags
	c.setFlag(FlagCarry, result > 0xFF)
	c.setFlag(FlagOverflow, (c.A^uint8(result))&(value^uint8(result))&0x80 != 0)

	c.A = uint8(result)
	c.setZN(c.A)

	cycles := getLoadCycles(mode)
	if pageCrossed && (mode == AddrAbsoluteX || mode == AddrAbsoluteY || mode == AddrIndirectIndexed) {
		cycles++
	}
	return cycles
}

// SBC - Subtract with Carry
func (c *CPU) execSBC(mode AddressingMode) int {
	value, pageCrossed := c.getOperand(mode)

	carry := uint8(0)
	if c.getFlag(FlagCarry) {
		carry = 1
	}

	// NES CPU (2A03/2A07) does not support decimal mode
	// Always use binary mode
	result := uint16(c.A) - uint16(value) - uint16(1-carry)

	// Set flags
	c.setFlag(FlagCarry, result <= 0xFF)
	c.setFlag(FlagOverflow, (c.A^uint8(result))&((c.A^value)&0x80) != 0)

	c.A = uint8(result)
	c.setZN(c.A)

	// Return cycles based on addressing mode
	switch mode {
	case AddrImmediate:
		return 2
	case AddrZeroPage:
		return 3
	case AddrZeroPageX:
		return 4
	case AddrAbsolute:
		return 4
	case AddrAbsoluteX, AddrAbsoluteY:
		cycles := 4
		if pageCrossed {
			cycles++
		}
		return cycles
	case AddrIndexedIndirect:
		return 6
	case AddrIndirectIndexed:
		cycles := 5
		if pageCrossed {
			cycles++
		}
		return cycles
	default:
		return 2
	}
}

// CMP - Compare Accumulator
func (c *CPU) execCMP(mode AddressingMode) int {
	value, pageCrossed := c.getOperand(mode)

	result := c.A - value
	c.setFlag(FlagCarry, c.A >= value)
	c.setZN(result)

	cycles := getAddressingInfo(0xC9).Cycles // Base cycles for CMP
	if pageCrossed {
		cycles++
	}
	return cycles
}

// Transfer instructions
func (c *CPU) execTAX() int {
	c.X = c.A
	c.setZN(c.X)
	return 2
}

func (c *CPU) execTXA() int {
	c.A = c.X
	c.setZN(c.A)
	return 2
}

func (c *CPU) execTAY() int {
	c.Y = c.A
	c.setZN(c.Y)
	return 2
}

func (c *CPU) execTYA() int {
	c.A = c.Y
	c.setZN(c.A)
	return 2
}

func (c *CPU) execTXS() int {
	c.SP = c.X
	return 2
}

func (c *CPU) execTSX() int {
	c.X = c.SP
	c.setZN(c.X)
	return 2
}

// Flag instructions
func (c *CPU) execCLC() int {
	c.setFlag(FlagCarry, false)
	return 2
}

func (c *CPU) execSEC() int {
	c.setFlag(FlagCarry, true)
	return 2
}

func (c *CPU) execCLI() int {
	c.setFlag(FlagInterrupt, false)
	return 2
}

func (c *CPU) execSEI() int {
	c.setFlag(FlagInterrupt, true)
	return 2
}

func (c *CPU) execCLV() int {
	c.setFlag(FlagOverflow, false)
	return 2
}

func (c *CPU) execCLD() int {
	c.setFlag(FlagDecimal, false)
	return 2
}

func (c *CPU) execSED() int {
	c.setFlag(FlagDecimal, true)
	return 2
}

// Stack instructions
func (c *CPU) execPHA() int {
	c.push(c.A)
	return 3
}

func (c *CPU) execPLA() int {
	c.A = c.pop()
	c.setZN(c.A)
	return 4
}

func (c *CPU) execPHP() int {
	c.push(c.P | FlagBreak)
	return 3
}

func (c *CPU) execPLP() int {
	c.P = c.pop()
	c.P |= FlagUnused
	c.P &^= FlagBreak
	return 4
}

// Branch instructions
func (c *CPU) execBEQ() int {
	return c.branch(c.getFlag(FlagZero))
}

func (c *CPU) execBNE() int {
	return c.branch(!c.getFlag(FlagZero))
}

func (c *CPU) execBCC() int {
	return c.branch(!c.getFlag(FlagCarry))
}

func (c *CPU) execBCS() int {
	return c.branch(c.getFlag(FlagCarry))
}

func (c *CPU) execBPL() int {
	return c.branch(!c.getFlag(FlagNegative))
}

func (c *CPU) execBMI() int {
	return c.branch(c.getFlag(FlagNegative))
}

func (c *CPU) execBVC() int {
	return c.branch(!c.getFlag(FlagOverflow))
}

func (c *CPU) execBVS() int {
	return c.branch(c.getFlag(FlagOverflow))
}

// branch helper function - handles relative addressing and timing
func (c *CPU) branch(condition bool) int {
	offset := int8(c.read(c.PC))
	c.PC++

	if condition {
		oldPC := c.PC
		newPC := uint16(int32(c.PC) + int32(offset))
		c.PC = newPC

		// Branch taken: 3 cycles base, +1 if page crossed
		cycles := 3
		if (oldPC & 0xFF00) != (newPC & 0xFF00) {
			cycles = 4 // Page boundary crossed
		}
		return cycles
	}

	// Branch not taken: 2 cycles
	return 2
}

// Jump instructions
func (c *CPU) execJMPAbsolute() int {
	low := c.read(c.PC)
	c.PC++
	high := c.read(c.PC)
	c.PC = uint16(high)<<8 | uint16(low)
	return 3
}

func (c *CPU) execJMPIndirect() int {
	// Read indirect address
	low := c.read(c.PC)
	c.PC++
	high := c.read(c.PC)
	indirectAddr := uint16(high)<<8 | uint16(low)

	// Read actual jump address with 6502 page boundary bug
	// If indirect address low byte is 0xFF, high byte is read from same page
	var actualLow, actualHigh uint8
	actualLow = c.read(indirectAddr)
	if (indirectAddr & 0xFF) == 0xFF {
		// Bug: reads from same page instead of next page
		actualHigh = c.read(indirectAddr & 0xFF00)
	} else {
		actualHigh = c.read(indirectAddr + 1)
	}

	c.PC = uint16(actualHigh)<<8 | uint16(actualLow)
	return 5
}

func (c *CPU) execJSR() int {
	// Read target address
	low := c.read(c.PC)
	c.PC++
	high := c.read(c.PC)

	// Push return address - 1 (PC is currently pointing to high byte)
	returnAddr := c.PC
	c.push(uint8(returnAddr >> 8))   // Push high byte
	c.push(uint8(returnAddr & 0xFF)) // Push low byte

	// Jump to subroutine
	c.PC = uint16(high)<<8 | uint16(low)
	return 6
}

func (c *CPU) execRTS() int {
	// Pop return address
	low := c.pop()
	high := c.pop()
	c.PC = (uint16(high)<<8 | uint16(low)) + 1
	return 6
}

func (c *CPU) execRTI() int {
	// Pop status register
	c.P = c.pop()
	c.P |= FlagUnused
	c.P &^= FlagBreak

	// Pop return address
	low := c.pop()
	high := c.pop()
	c.PC = uint16(high)<<8 | uint16(low)
	return 6
}

// Logical operations
func (c *CPU) execAND(mode AddressingMode) int {
	value, pageCrossed := c.getOperand(mode)
	c.A = c.A & value
	c.setZN(c.A)

	cycles := getLogicalCycles(mode)
	if pageCrossed && (mode == AddrAbsoluteX || mode == AddrAbsoluteY || mode == AddrIndirectIndexed) {
		cycles++
	}
	return cycles
}

func (c *CPU) execORA(mode AddressingMode) int {
	value, pageCrossed := c.getOperand(mode)
	c.A = c.A | value
	c.setZN(c.A)

	cycles := getLogicalCycles(mode)
	if pageCrossed && (mode == AddrAbsoluteX || mode == AddrAbsoluteY || mode == AddrIndirectIndexed) {
		cycles++
	}
	return cycles
}

func (c *CPU) execEOR(mode AddressingMode) int {
	value, pageCrossed := c.getOperand(mode)
	c.A = c.A ^ value
	c.setZN(c.A)

	cycles := getLogicalCycles(mode)
	if pageCrossed && (mode == AddrAbsoluteX || mode == AddrAbsoluteY || mode == AddrIndirectIndexed) {
		cycles++
	}
	return cycles
}

// Helper function to get cycles for logical operations
func getLogicalCycles(mode AddressingMode) int {
	switch mode {
	case AddrImmediate:
		return 2
	case AddrZeroPage:
		return 3
	case AddrZeroPageX:
		return 4
	case AddrAbsolute:
		return 4
	case AddrAbsoluteX, AddrAbsoluteY:
		return 4 // +1 if page crossed (handled by caller)
	case AddrIndexedIndirect:
		return 6
	case AddrIndirectIndexed:
		return 5 // +1 if page crossed (handled by caller)
	default:
		return 2
	}
}

// Shift and rotate instructions
func (c *CPU) execASLAccumulator() int {
	c.setFlag(FlagCarry, c.A&0x80 != 0)
	c.A = c.A << 1
	c.setZN(c.A)
	return 2
}

func (c *CPU) execASL(mode AddressingMode) int {
	addr, _ := c.getOperandAddress(mode)
	value := c.read(addr)

	c.setFlag(FlagCarry, value&0x80 != 0)
	result := value << 1
	c.setZN(result)

	c.write(addr, result)
	return getShiftCycles(mode)
}

func (c *CPU) execLSRAccumulator() int {
	c.setFlag(FlagCarry, c.A&0x01 != 0)
	c.A = c.A >> 1
	c.setZN(c.A)
	return 2
}

func (c *CPU) execLSR(mode AddressingMode) int {
	addr, _ := c.getOperandAddress(mode)
	value := c.read(addr)

	c.setFlag(FlagCarry, value&0x01 != 0)
	result := value >> 1
	c.setZN(result)

	c.write(addr, result)
	return getShiftCycles(mode)
}

func (c *CPU) execROLAccumulator() int {
	oldCarry := uint8(0)
	if c.getFlag(FlagCarry) {
		oldCarry = 1
	}

	c.setFlag(FlagCarry, c.A&0x80 != 0)
	c.A = (c.A << 1) | oldCarry
	c.setZN(c.A)
	return 2
}

func (c *CPU) execROL(mode AddressingMode) int {
	addr, _ := c.getOperandAddress(mode)
	value := c.read(addr)

	oldCarry := uint8(0)
	if c.getFlag(FlagCarry) {
		oldCarry = 1
	}

	c.setFlag(FlagCarry, value&0x80 != 0)
	result := (value << 1) | oldCarry
	c.setZN(result)

	c.write(addr, result)
	return getShiftCycles(mode)
}

func (c *CPU) execRORAccumulator() int {
	oldCarry := uint8(0)
	if c.getFlag(FlagCarry) {
		oldCarry = 0x80
	}

	c.setFlag(FlagCarry, c.A&0x01 != 0)
	c.A = (c.A >> 1) | oldCarry
	c.setZN(c.A)
	return 2
}

func (c *CPU) execROR(mode AddressingMode) int {
	addr, _ := c.getOperandAddress(mode)
	value := c.read(addr)

	oldCarry := uint8(0)
	if c.getFlag(FlagCarry) {
		oldCarry = 0x80
	}

	c.setFlag(FlagCarry, value&0x01 != 0)
	result := (value >> 1) | oldCarry
	c.setZN(result)

	c.write(addr, result)
	return getShiftCycles(mode)
}

// Helper function to get cycles for shift/rotate operations
func getShiftCycles(mode AddressingMode) int {
	switch mode {
	case AddrZeroPage:
		return 5
	case AddrZeroPageX:
		return 6
	case AddrAbsolute:
		return 6
	case AddrAbsoluteX:
		return 7
	default:
		return 2
	}
}

// Increment/Decrement instructions
func (c *CPU) execINC(mode AddressingMode) int {
	addr, _ := c.getOperandAddress(mode)
	value := c.read(addr)
	result := value + 1
	c.setZN(result)
	c.write(addr, result)
	return getShiftCycles(mode)
}

func (c *CPU) execDEC(mode AddressingMode) int {
	addr, _ := c.getOperandAddress(mode)
	value := c.read(addr)
	result := value - 1

	c.setZN(result)

	c.write(addr, result)
	return getShiftCycles(mode)
}

func (c *CPU) execINX() int {
	c.X++
	c.setZN(c.X)
	return 2
}

func (c *CPU) execDEX() int {
	c.X--
	c.setZN(c.X)
	return 2
}

func (c *CPU) execINY() int {
	c.Y++
	c.setZN(c.Y)
	return 2
}

func (c *CPU) execDEY() int {
	c.Y--
	c.setZN(c.Y)
	return 2
}

// Compare instructions
func (c *CPU) execCPX(mode AddressingMode) int {
	value, pageCrossed := c.getOperand(mode)
	result := c.X - value
	c.setFlag(FlagCarry, c.X >= value)
	c.setZN(result)

	cycles := getLogicalCycles(mode)
	if pageCrossed && (mode == AddrAbsoluteX || mode == AddrAbsoluteY || mode == AddrIndirectIndexed) {
		cycles++
	}
	return cycles
}

func (c *CPU) execCPY(mode AddressingMode) int {
	value, pageCrossed := c.getOperand(mode)
	result := c.Y - value
	c.setFlag(FlagCarry, c.Y >= value)
	c.setZN(result)

	cycles := getLogicalCycles(mode)
	if pageCrossed && (mode == AddrAbsoluteX || mode == AddrAbsoluteY || mode == AddrIndirectIndexed) {
		cycles++
	}
	return cycles
}

// Bit test instruction
func (c *CPU) execBIT(mode AddressingMode) int {
	value, _ := c.getOperand(mode)
	result := c.A & value

	c.setFlag(FlagZero, result == 0)
	c.setFlag(FlagNegative, value&0x80 != 0) // Bit 7 of memory
	c.setFlag(FlagOverflow, value&0x40 != 0) // Bit 6 of memory

	return getLogicalCycles(mode)
}

// BRK instruction - software interrupt
func (c *CPU) execBRK() int {
	c.PC++ // BRK is effectively a 2-byte instruction
	c.push16(c.PC)
	c.push(c.P | FlagBreak)
	c.setFlag(FlagInterrupt, true)
	c.PC = c.read16(0xFFFE) // IRQ vector
	return 7
}

// NOP
func (c *CPU) execNOP() int {
	return 2
}

// Helper function to set Zero and Negative flags
func (c *CPU) setZN(value uint8) {
	c.setFlag(FlagZero, value == 0)
	c.setFlag(FlagNegative, value&0x80 != 0)
}

// Illegal opcodes implementation

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

// ISB - Increment and Subtract with Borrow
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

// AAC - AND accumulator with immediate (also sets carry flag)
func (c *CPU) execAAC() int {
	value := c.read(c.PC)
	c.PC++

	c.A &= value
	c.setZN(c.A)
	c.setFlag(FlagCarry, c.A&0x80 != 0) // Set carry flag based on bit 7

	return 2
}

// ASR - AND with immediate, then LSR
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

// ATX - Load immediate to A and X (also known as LXA)
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

// AXS - AND X with A, then subtract immediate (without borrow)
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
