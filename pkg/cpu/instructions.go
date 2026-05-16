package cpu

// opcodeTable maps each opcode byte to its execution function. Populated by
// init() at package load. A nil entry means the opcode is unimplemented and
// dispatch falls through to execUnknown.
var opcodeTable [256]func(c *CPU) int

// modeExec returns a closure that runs op for the given addressing mode. It
// exists purely to keep the dispatch table compact: exec* methods take an
// AddressingMode, but the table stores func(*CPU) int.
func modeExec(op func(c *CPU, mode AddressingMode) int, mode AddressingMode) func(*CPU) int {
	return func(c *CPU) int { return op(c, mode) }
}

func init() {
	// LDA - A9/A5/B5/AD/BD/B9/A1/B1
	opcodeTable[0xA9] = (*CPU).execLDAImmediate
	opcodeTable[0xA5] = modeExec((*CPU).execLDA, AddrZeroPage)
	opcodeTable[0xB5] = modeExec((*CPU).execLDA, AddrZeroPageX)
	opcodeTable[0xAD] = modeExec((*CPU).execLDA, AddrAbsolute)
	opcodeTable[0xBD] = modeExec((*CPU).execLDA, AddrAbsoluteX)
	opcodeTable[0xB9] = modeExec((*CPU).execLDA, AddrAbsoluteY)
	opcodeTable[0xA1] = modeExec((*CPU).execLDA, AddrIndexedIndirect)
	opcodeTable[0xB1] = modeExec((*CPU).execLDA, AddrIndirectIndexed)

	// LDX - A2/A6/B6/AE/BE
	opcodeTable[0xA2] = modeExec((*CPU).execLDX, AddrImmediate)
	opcodeTable[0xA6] = modeExec((*CPU).execLDX, AddrZeroPage)
	opcodeTable[0xB6] = modeExec((*CPU).execLDX, AddrZeroPageY)
	opcodeTable[0xAE] = modeExec((*CPU).execLDX, AddrAbsolute)
	opcodeTable[0xBE] = modeExec((*CPU).execLDX, AddrAbsoluteY)

	// LDY - A0/A4/B4/AC/BC
	opcodeTable[0xA0] = modeExec((*CPU).execLDY, AddrImmediate)
	opcodeTable[0xA4] = modeExec((*CPU).execLDY, AddrZeroPage)
	opcodeTable[0xB4] = modeExec((*CPU).execLDY, AddrZeroPageX)
	opcodeTable[0xAC] = modeExec((*CPU).execLDY, AddrAbsolute)
	opcodeTable[0xBC] = modeExec((*CPU).execLDY, AddrAbsoluteX)

	// STA - 85/95/8D/9D/99/81/91
	opcodeTable[0x85] = modeExec((*CPU).execSTA, AddrZeroPage)
	opcodeTable[0x95] = modeExec((*CPU).execSTA, AddrZeroPageX)
	opcodeTable[0x8D] = modeExec((*CPU).execSTA, AddrAbsolute)
	opcodeTable[0x9D] = modeExec((*CPU).execSTA, AddrAbsoluteX)
	opcodeTable[0x99] = modeExec((*CPU).execSTA, AddrAbsoluteY)
	opcodeTable[0x81] = modeExec((*CPU).execSTA, AddrIndexedIndirect)
	opcodeTable[0x91] = modeExec((*CPU).execSTA, AddrIndirectIndexed)

	// STX - 86/96/8E
	opcodeTable[0x86] = modeExec((*CPU).execSTX, AddrZeroPage)
	opcodeTable[0x96] = modeExec((*CPU).execSTX, AddrZeroPageY)
	opcodeTable[0x8E] = modeExec((*CPU).execSTX, AddrAbsolute)

	// STY - 84/94/8C
	opcodeTable[0x84] = modeExec((*CPU).execSTY, AddrZeroPage)
	opcodeTable[0x94] = modeExec((*CPU).execSTY, AddrZeroPageX)
	opcodeTable[0x8C] = modeExec((*CPU).execSTY, AddrAbsolute)

	// ADC - 69/65/75/6D/7D/79/61/71
	opcodeTable[0x69] = modeExec((*CPU).execADC, AddrImmediate)
	opcodeTable[0x65] = modeExec((*CPU).execADC, AddrZeroPage)
	opcodeTable[0x75] = modeExec((*CPU).execADC, AddrZeroPageX)
	opcodeTable[0x6D] = modeExec((*CPU).execADC, AddrAbsolute)
	opcodeTable[0x7D] = modeExec((*CPU).execADC, AddrAbsoluteX)
	opcodeTable[0x79] = modeExec((*CPU).execADC, AddrAbsoluteY)
	opcodeTable[0x61] = modeExec((*CPU).execADC, AddrIndexedIndirect)
	opcodeTable[0x71] = modeExec((*CPU).execADC, AddrIndirectIndexed)

	// SBC - E9/E5/F5/ED/FD/F9/E1/F1 (+EB illegal alias)
	opcodeTable[0xE9] = modeExec((*CPU).execSBC, AddrImmediate)
	opcodeTable[0xE5] = modeExec((*CPU).execSBC, AddrZeroPage)
	opcodeTable[0xF5] = modeExec((*CPU).execSBC, AddrZeroPageX)
	opcodeTable[0xED] = modeExec((*CPU).execSBC, AddrAbsolute)
	opcodeTable[0xFD] = modeExec((*CPU).execSBC, AddrAbsoluteX)
	opcodeTable[0xF9] = modeExec((*CPU).execSBC, AddrAbsoluteY)
	opcodeTable[0xE1] = modeExec((*CPU).execSBC, AddrIndexedIndirect)
	opcodeTable[0xF1] = modeExec((*CPU).execSBC, AddrIndirectIndexed)
	opcodeTable[0xEB] = modeExec((*CPU).execSBC, AddrImmediate) // illegal SBC #imm

	// CMP - C9/C5/D5/CD/DD/D9/C1/D1
	opcodeTable[0xC9] = modeExec((*CPU).execCMP, AddrImmediate)
	opcodeTable[0xC5] = modeExec((*CPU).execCMP, AddrZeroPage)
	opcodeTable[0xD5] = modeExec((*CPU).execCMP, AddrZeroPageX)
	opcodeTable[0xCD] = modeExec((*CPU).execCMP, AddrAbsolute)
	opcodeTable[0xDD] = modeExec((*CPU).execCMP, AddrAbsoluteX)
	opcodeTable[0xD9] = modeExec((*CPU).execCMP, AddrAbsoluteY)
	opcodeTable[0xC1] = modeExec((*CPU).execCMP, AddrIndexedIndirect)
	opcodeTable[0xD1] = modeExec((*CPU).execCMP, AddrIndirectIndexed)

	// CPX - E0/E4/EC
	opcodeTable[0xE0] = modeExec((*CPU).execCPX, AddrImmediate)
	opcodeTable[0xE4] = modeExec((*CPU).execCPX, AddrZeroPage)
	opcodeTable[0xEC] = modeExec((*CPU).execCPX, AddrAbsolute)

	// CPY - C0/C4/CC
	opcodeTable[0xC0] = modeExec((*CPU).execCPY, AddrImmediate)
	opcodeTable[0xC4] = modeExec((*CPU).execCPY, AddrZeroPage)
	opcodeTable[0xCC] = modeExec((*CPU).execCPY, AddrAbsolute)

	// Transfers - AA/8A/A8/98/9A/BA
	opcodeTable[0xAA] = (*CPU).execTAX
	opcodeTable[0x8A] = (*CPU).execTXA
	opcodeTable[0xA8] = (*CPU).execTAY
	opcodeTable[0x98] = (*CPU).execTYA
	opcodeTable[0x9A] = (*CPU).execTXS
	opcodeTable[0xBA] = (*CPU).execTSX

	// Flag ops - 18/38/58/78/B8/D8/F8
	opcodeTable[0x18] = (*CPU).execCLC
	opcodeTable[0x38] = (*CPU).execSEC
	opcodeTable[0x58] = (*CPU).execCLI
	opcodeTable[0x78] = (*CPU).execSEI
	opcodeTable[0xB8] = (*CPU).execCLV
	opcodeTable[0xD8] = (*CPU).execCLD
	opcodeTable[0xF8] = (*CPU).execSED

	// Stack - 48/68/08/28
	opcodeTable[0x48] = (*CPU).execPHA
	opcodeTable[0x68] = (*CPU).execPLA
	opcodeTable[0x08] = (*CPU).execPHP
	opcodeTable[0x28] = (*CPU).execPLP

	// Branches - 10/30/50/70/90/B0/D0/F0
	opcodeTable[0x10] = (*CPU).execBPL
	opcodeTable[0x30] = (*CPU).execBMI
	opcodeTable[0x50] = (*CPU).execBVC
	opcodeTable[0x70] = (*CPU).execBVS
	opcodeTable[0x90] = (*CPU).execBCC
	opcodeTable[0xB0] = (*CPU).execBCS
	opcodeTable[0xD0] = (*CPU).execBNE
	opcodeTable[0xF0] = (*CPU).execBEQ

	// Jumps & subroutines - 4C/6C/20/60/40
	opcodeTable[0x4C] = (*CPU).execJMPAbsolute
	opcodeTable[0x6C] = (*CPU).execJMPIndirect
	opcodeTable[0x20] = (*CPU).execJSR
	opcodeTable[0x60] = (*CPU).execRTS
	opcodeTable[0x40] = (*CPU).execRTI

	// AND - 29/25/35/2D/3D/39/21/31
	opcodeTable[0x29] = modeExec((*CPU).execAND, AddrImmediate)
	opcodeTable[0x25] = modeExec((*CPU).execAND, AddrZeroPage)
	opcodeTable[0x35] = modeExec((*CPU).execAND, AddrZeroPageX)
	opcodeTable[0x2D] = modeExec((*CPU).execAND, AddrAbsolute)
	opcodeTable[0x3D] = modeExec((*CPU).execAND, AddrAbsoluteX)
	opcodeTable[0x39] = modeExec((*CPU).execAND, AddrAbsoluteY)
	opcodeTable[0x21] = modeExec((*CPU).execAND, AddrIndexedIndirect)
	opcodeTable[0x31] = modeExec((*CPU).execAND, AddrIndirectIndexed)

	// ORA - 09/05/15/0D/1D/19/01/11
	opcodeTable[0x09] = modeExec((*CPU).execORA, AddrImmediate)
	opcodeTable[0x05] = modeExec((*CPU).execORA, AddrZeroPage)
	opcodeTable[0x15] = modeExec((*CPU).execORA, AddrZeroPageX)
	opcodeTable[0x0D] = modeExec((*CPU).execORA, AddrAbsolute)
	opcodeTable[0x1D] = modeExec((*CPU).execORA, AddrAbsoluteX)
	opcodeTable[0x19] = modeExec((*CPU).execORA, AddrAbsoluteY)
	opcodeTable[0x01] = modeExec((*CPU).execORA, AddrIndexedIndirect)
	opcodeTable[0x11] = modeExec((*CPU).execORA, AddrIndirectIndexed)

	// EOR - 49/45/55/4D/5D/59/41/51
	opcodeTable[0x49] = modeExec((*CPU).execEOR, AddrImmediate)
	opcodeTable[0x45] = modeExec((*CPU).execEOR, AddrZeroPage)
	opcodeTable[0x55] = modeExec((*CPU).execEOR, AddrZeroPageX)
	opcodeTable[0x4D] = modeExec((*CPU).execEOR, AddrAbsolute)
	opcodeTable[0x5D] = modeExec((*CPU).execEOR, AddrAbsoluteX)
	opcodeTable[0x59] = modeExec((*CPU).execEOR, AddrAbsoluteY)
	opcodeTable[0x41] = modeExec((*CPU).execEOR, AddrIndexedIndirect)
	opcodeTable[0x51] = modeExec((*CPU).execEOR, AddrIndirectIndexed)

	// ASL - 0A/06/16/0E/1E
	opcodeTable[0x0A] = (*CPU).execASLAccumulator
	opcodeTable[0x06] = modeExec((*CPU).execASL, AddrZeroPage)
	opcodeTable[0x16] = modeExec((*CPU).execASL, AddrZeroPageX)
	opcodeTable[0x0E] = modeExec((*CPU).execASL, AddrAbsolute)
	opcodeTable[0x1E] = modeExec((*CPU).execASL, AddrAbsoluteX)

	// LSR - 4A/46/56/4E/5E
	opcodeTable[0x4A] = (*CPU).execLSRAccumulator
	opcodeTable[0x46] = modeExec((*CPU).execLSR, AddrZeroPage)
	opcodeTable[0x56] = modeExec((*CPU).execLSR, AddrZeroPageX)
	opcodeTable[0x4E] = modeExec((*CPU).execLSR, AddrAbsolute)
	opcodeTable[0x5E] = modeExec((*CPU).execLSR, AddrAbsoluteX)

	// ROL - 2A/26/36/2E/3E
	opcodeTable[0x2A] = (*CPU).execROLAccumulator
	opcodeTable[0x26] = modeExec((*CPU).execROL, AddrZeroPage)
	opcodeTable[0x36] = modeExec((*CPU).execROL, AddrZeroPageX)
	opcodeTable[0x2E] = modeExec((*CPU).execROL, AddrAbsolute)
	opcodeTable[0x3E] = modeExec((*CPU).execROL, AddrAbsoluteX)

	// ROR - 6A/66/76/6E/7E
	opcodeTable[0x6A] = (*CPU).execRORAccumulator
	opcodeTable[0x66] = modeExec((*CPU).execROR, AddrZeroPage)
	opcodeTable[0x76] = modeExec((*CPU).execROR, AddrZeroPageX)
	opcodeTable[0x6E] = modeExec((*CPU).execROR, AddrAbsolute)
	opcodeTable[0x7E] = modeExec((*CPU).execROR, AddrAbsoluteX)

	// INC - E6/F6/EE/FE
	opcodeTable[0xE6] = modeExec((*CPU).execINC, AddrZeroPage)
	opcodeTable[0xF6] = modeExec((*CPU).execINC, AddrZeroPageX)
	opcodeTable[0xEE] = modeExec((*CPU).execINC, AddrAbsolute)
	opcodeTable[0xFE] = modeExec((*CPU).execINC, AddrAbsoluteX)

	// DEC - C6/D6/CE/DE
	opcodeTable[0xC6] = modeExec((*CPU).execDEC, AddrZeroPage)
	opcodeTable[0xD6] = modeExec((*CPU).execDEC, AddrZeroPageX)
	opcodeTable[0xCE] = modeExec((*CPU).execDEC, AddrAbsolute)
	opcodeTable[0xDE] = modeExec((*CPU).execDEC, AddrAbsoluteX)

	// INX/DEX/INY/DEY - E8/CA/C8/88
	opcodeTable[0xE8] = (*CPU).execINX
	opcodeTable[0xCA] = (*CPU).execDEX
	opcodeTable[0xC8] = (*CPU).execINY
	opcodeTable[0x88] = (*CPU).execDEY

	// BIT - 24/2C
	opcodeTable[0x24] = modeExec((*CPU).execBIT, AddrZeroPage)
	opcodeTable[0x2C] = modeExec((*CPU).execBIT, AddrAbsolute)

	// BRK / NOP - 00/EA
	opcodeTable[0x00] = (*CPU).execBRK
	opcodeTable[0xEA] = (*CPU).execNOP

	// Illegal NOPs (implied): 1A/3A/5A/7A/DA/FA -> NOP, 2 cycles, no operand
	for _, op := range []uint8{0x1A, 0x3A, 0x5A, 0x7A, 0xDA, 0xFA} {
		opcodeTable[op] = (*CPU).execNOP
	}
	// Illegal NOP #imm: 80/82/89/C2/E2 -> skip operand, 2 cycles
	for _, op := range []uint8{0x80, 0x82, 0x89, 0xC2, 0xE2} {
		opcodeTable[op] = (*CPU).execNopImmediate
	}
	// Illegal NOP zp: 04/44/64 -> skip operand, 3 cycles
	for _, op := range []uint8{0x04, 0x44, 0x64} {
		opcodeTable[op] = (*CPU).execNopZeroPage
	}
	// Illegal NOP zp,X: 14/34/54/74/D4/F4 -> skip operand, 4 cycles
	for _, op := range []uint8{0x14, 0x34, 0x54, 0x74, 0xD4, 0xF4} {
		opcodeTable[op] = (*CPU).execNopZeroPageX
	}
	// Illegal NOP abs: 0C -> skip 2-byte operand, 4 cycles
	opcodeTable[0x0C] = (*CPU).execNopAbsolute
	// Illegal NOP abs,X: 1C/3C/5C/7C/DC/FC -> skip 2-byte operand, 4 cycles
	for _, op := range []uint8{0x1C, 0x3C, 0x5C, 0x7C, 0xDC, 0xFC} {
		opcodeTable[op] = (*CPU).execNopAbsoluteX
	}

	// LAX (illegal) - AF/BF/A7/B7/A3/B3
	opcodeTable[0xAF] = modeExec((*CPU).execLAX, AddrAbsolute)
	opcodeTable[0xBF] = modeExec((*CPU).execLAX, AddrAbsoluteY)
	opcodeTable[0xA7] = modeExec((*CPU).execLAX, AddrZeroPage)
	opcodeTable[0xB7] = modeExec((*CPU).execLAX, AddrZeroPageY)
	opcodeTable[0xA3] = modeExec((*CPU).execLAX, AddrIndexedIndirect)
	opcodeTable[0xB3] = modeExec((*CPU).execLAX, AddrIndirectIndexed)

	// SAX (illegal) - 8F/87/97/83
	opcodeTable[0x8F] = modeExec((*CPU).execSAX, AddrAbsolute)
	opcodeTable[0x87] = modeExec((*CPU).execSAX, AddrZeroPage)
	opcodeTable[0x97] = modeExec((*CPU).execSAX, AddrZeroPageY)
	opcodeTable[0x83] = modeExec((*CPU).execSAX, AddrIndexedIndirect)

	// AAC/ANC (illegal) - 0B/2B
	opcodeTable[0x0B] = (*CPU).execAAC
	opcodeTable[0x2B] = (*CPU).execAAC

	// ASR/ALR (illegal) - 4B
	opcodeTable[0x4B] = (*CPU).execASR
	// ARR (illegal) - 6B
	opcodeTable[0x6B] = (*CPU).execARR
	// ATX/LXA (illegal) - AB
	opcodeTable[0xAB] = (*CPU).execATX
	// AXS/SBX (illegal) - CB
	opcodeTable[0xCB] = (*CPU).execAXS

	// DCP (illegal) - CF/DF/DB/C7/D7/C3/D3
	opcodeTable[0xCF] = modeExec((*CPU).execDCP, AddrAbsolute)
	opcodeTable[0xDF] = modeExec((*CPU).execDCP, AddrAbsoluteX)
	opcodeTable[0xDB] = modeExec((*CPU).execDCP, AddrAbsoluteY)
	opcodeTable[0xC7] = modeExec((*CPU).execDCP, AddrZeroPage)
	opcodeTable[0xD7] = modeExec((*CPU).execDCP, AddrZeroPageX)
	opcodeTable[0xC3] = modeExec((*CPU).execDCP, AddrIndexedIndirect)
	opcodeTable[0xD3] = modeExec((*CPU).execDCP, AddrIndirectIndexed)

	// ISB/ISC (illegal) - EF/FF/FB/E7/F7/E3/F3
	opcodeTable[0xEF] = modeExec((*CPU).execISB, AddrAbsolute)
	opcodeTable[0xFF] = modeExec((*CPU).execISB, AddrAbsoluteX)
	opcodeTable[0xFB] = modeExec((*CPU).execISB, AddrAbsoluteY)
	opcodeTable[0xE7] = modeExec((*CPU).execISB, AddrZeroPage)
	opcodeTable[0xF7] = modeExec((*CPU).execISB, AddrZeroPageX)
	opcodeTable[0xE3] = modeExec((*CPU).execISB, AddrIndexedIndirect)
	opcodeTable[0xF3] = modeExec((*CPU).execISB, AddrIndirectIndexed)

	// SLO (illegal) - 0F/1F/1B/07/17/03/13
	opcodeTable[0x0F] = modeExec((*CPU).execSLO, AddrAbsolute)
	opcodeTable[0x1F] = modeExec((*CPU).execSLO, AddrAbsoluteX)
	opcodeTable[0x1B] = modeExec((*CPU).execSLO, AddrAbsoluteY)
	opcodeTable[0x07] = modeExec((*CPU).execSLO, AddrZeroPage)
	opcodeTable[0x17] = modeExec((*CPU).execSLO, AddrZeroPageX)
	opcodeTable[0x03] = modeExec((*CPU).execSLO, AddrIndexedIndirect)
	opcodeTable[0x13] = modeExec((*CPU).execSLO, AddrIndirectIndexed)

	// RLA (illegal) - 2F/3F/3B/27/37/23/33
	opcodeTable[0x2F] = modeExec((*CPU).execRLA, AddrAbsolute)
	opcodeTable[0x3F] = modeExec((*CPU).execRLA, AddrAbsoluteX)
	opcodeTable[0x3B] = modeExec((*CPU).execRLA, AddrAbsoluteY)
	opcodeTable[0x27] = modeExec((*CPU).execRLA, AddrZeroPage)
	opcodeTable[0x37] = modeExec((*CPU).execRLA, AddrZeroPageX)
	opcodeTable[0x23] = modeExec((*CPU).execRLA, AddrIndexedIndirect)
	opcodeTable[0x33] = modeExec((*CPU).execRLA, AddrIndirectIndexed)

	// SRE (illegal) - 4F/5F/5B/47/57/43/53
	opcodeTable[0x4F] = modeExec((*CPU).execSRE, AddrAbsolute)
	opcodeTable[0x5F] = modeExec((*CPU).execSRE, AddrAbsoluteX)
	opcodeTable[0x5B] = modeExec((*CPU).execSRE, AddrAbsoluteY)
	opcodeTable[0x47] = modeExec((*CPU).execSRE, AddrZeroPage)
	opcodeTable[0x57] = modeExec((*CPU).execSRE, AddrZeroPageX)
	opcodeTable[0x43] = modeExec((*CPU).execSRE, AddrIndexedIndirect)
	opcodeTable[0x53] = modeExec((*CPU).execSRE, AddrIndirectIndexed)

	// RRA (illegal) - 6F/7F/7B/67/77/63/73
	opcodeTable[0x6F] = modeExec((*CPU).execRRA, AddrAbsolute)
	opcodeTable[0x7F] = modeExec((*CPU).execRRA, AddrAbsoluteX)
	opcodeTable[0x7B] = modeExec((*CPU).execRRA, AddrAbsoluteY)
	opcodeTable[0x67] = modeExec((*CPU).execRRA, AddrZeroPage)
	opcodeTable[0x77] = modeExec((*CPU).execRRA, AddrZeroPageX)
	opcodeTable[0x63] = modeExec((*CPU).execRRA, AddrIndexedIndirect)
	opcodeTable[0x73] = modeExec((*CPU).execRRA, AddrIndirectIndexed)
}

// executeInstruction dispatches one opcode through opcodeTable. Unmapped
// entries fall through to execUnknown, preserving the previous switch's
// default behaviour.
func (c *CPU) executeInstruction(opcode uint8) int {
	if fn := opcodeTable[opcode]; fn != nil {
		return fn(c)
	}
	return c.execUnknown(opcode)
}

// execUnknown handles opcodes with no dispatch-table entry. The original
// switch's default arm simply consumed 2 cycles, so we preserve that exactly.
func (c *CPU) execUnknown(opcode uint8) int {
	return 2
}

// Illegal NOP variants. These mirror the inline cases the original switch
// handled, kept as small named methods so the dispatch table can reference
// them directly.
func (c *CPU) execNopImmediate() int {
	c.PC++ // Skip immediate operand
	return 2
}

func (c *CPU) execNopZeroPage() int {
	c.PC++ // Skip zero page address
	return 3
}

func (c *CPU) execNopZeroPageX() int {
	c.PC++ // Skip zero page address
	return 4
}

func (c *CPU) execNopAbsolute() int {
	c.PC += 2 // Skip absolute address
	return 4
}

func (c *CPU) execNopAbsoluteX() int {
	c.PC += 2 // Skip absolute address
	return 4  // May be 5 if page boundary crossed, but simplified for now
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
