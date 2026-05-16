package cpu

// AddressingMode represents different addressing modes for 6502 instructions
type AddressingMode int

const (
	AddrImplied AddressingMode = iota
	AddrAccumulator
	AddrImmediate
	AddrZeroPage
	AddrZeroPageX
	AddrZeroPageY
	AddrRelative
	AddrAbsolute
	AddrAbsoluteX
	AddrAbsoluteY
	AddrIndirect
	AddrIndexedIndirect
	AddrIndirectIndexed
)

// AddressingInfo contains information about an addressing mode
type AddressingInfo struct {
	Mode   AddressingMode
	Length int // Instruction length in bytes
	Cycles int // Base cycle count
}

// getAddressingInfo returns addressing mode information for an opcode
func getAddressingInfo(opcode uint8) AddressingInfo {
	// Addressing mode lookup table for all 256 opcodes
	addressingTable := [256]AddressingInfo{
		// 0x00-0x0F
		{AddrImplied, 1, 7},       // 0x00 BRK
		{AddrIndexedIndirect, 2, 6}, // 0x01 ORA
		{AddrImplied, 1, 2},       // 0x02 JAM
		{AddrIndexedIndirect, 2, 8}, // 0x03 SLO
		{AddrZeroPage, 2, 3},      // 0x04 NOP
		{AddrZeroPage, 2, 3},      // 0x05 ORA
		{AddrZeroPage, 2, 5},      // 0x06 ASL
		{AddrZeroPage, 2, 5},      // 0x07 SLO
		{AddrImplied, 1, 3},       // 0x08 PHP
		{AddrImmediate, 2, 2},     // 0x09 ORA
		{AddrAccumulator, 1, 2},   // 0x0A ASL
		{AddrImmediate, 2, 2},     // 0x0B ANC
		{AddrAbsolute, 3, 4},      // 0x0C NOP
		{AddrAbsolute, 3, 4},      // 0x0D ORA
		{AddrAbsolute, 3, 6},      // 0x0E ASL
		{AddrAbsolute, 3, 6},      // 0x0F SLO
		
		// 0x10-0x1F
		{AddrRelative, 2, 2},      // 0x10 BPL
		{AddrIndirectIndexed, 2, 5}, // 0x11 ORA
		{AddrImplied, 1, 2},       // 0x12 JAM
		{AddrIndirectIndexed, 2, 8}, // 0x13 SLO
		{AddrZeroPageX, 2, 4},     // 0x14 NOP
		{AddrZeroPageX, 2, 4},     // 0x15 ORA
		{AddrZeroPageX, 2, 6},     // 0x16 ASL
		{AddrZeroPageX, 2, 6},     // 0x17 SLO
		{AddrImplied, 1, 2},       // 0x18 CLC
		{AddrAbsoluteY, 3, 4},     // 0x19 ORA
		{AddrImplied, 1, 2},       // 0x1A NOP
		{AddrAbsoluteY, 3, 7},     // 0x1B SLO
		{AddrAbsoluteX, 3, 4},     // 0x1C NOP
		{AddrAbsoluteX, 3, 4},     // 0x1D ORA
		{AddrAbsoluteX, 3, 7},     // 0x1E ASL
		{AddrAbsoluteX, 3, 7},     // 0x1F SLO
		
		// Continue with more opcodes...
		// For now, adding key opcodes for testing
		
		// 0x20-0x2F
		{AddrAbsolute, 3, 6},      // 0x20 JSR
		{AddrIndexedIndirect, 2, 6}, // 0x21 AND
		{AddrImplied, 1, 2},       // 0x22 JAM
		{AddrIndexedIndirect, 2, 8}, // 0x23 RLA
		{AddrZeroPage, 2, 3},      // 0x24 BIT
		{AddrZeroPage, 2, 3},      // 0x25 AND
		{AddrZeroPage, 2, 5},      // 0x26 ROL
		{AddrZeroPage, 2, 5},      // 0x27 RLA
		{AddrImplied, 1, 4},       // 0x28 PLP
		{AddrImmediate, 2, 2},     // 0x29 AND
		{AddrAccumulator, 1, 2},   // 0x2A ROL
		{AddrImmediate, 2, 2},     // 0x2B ANC
		{AddrAbsolute, 3, 4},      // 0x2C BIT
		{AddrAbsolute, 3, 4},      // 0x2D AND
		{AddrAbsolute, 3, 6},      // 0x2E ROL
		{AddrAbsolute, 3, 6},      // 0x2F RLA
		
		// 0x30-0xFF - Add more opcodes as needed
		// Initialize remaining entries with defaults
	}
	
	// Complete addressing table for all opcodes we implement
	switch opcode {
	// LDA
	case 0xA9:
		return AddressingInfo{AddrImmediate, 2, 2}
	case 0xA5:
		return AddressingInfo{AddrZeroPage, 2, 3}
	case 0xB5:
		return AddressingInfo{AddrZeroPageX, 2, 4}
	case 0xAD:
		return AddressingInfo{AddrAbsolute, 3, 4}
	case 0xBD:
		return AddressingInfo{AddrAbsoluteX, 3, 4}
	case 0xB9:
		return AddressingInfo{AddrAbsoluteY, 3, 4}
	case 0xA1:
		return AddressingInfo{AddrIndexedIndirect, 2, 6}
	case 0xB1:
		return AddressingInfo{AddrIndirectIndexed, 2, 5}
		
	// LDX
	case 0xA2:
		return AddressingInfo{AddrImmediate, 2, 2}
	case 0xA6:
		return AddressingInfo{AddrZeroPage, 2, 3}
	case 0xB6:
		return AddressingInfo{AddrZeroPageY, 2, 4}
	case 0xAE:
		return AddressingInfo{AddrAbsolute, 3, 4}
	case 0xBE:
		return AddressingInfo{AddrAbsoluteY, 3, 4}
		
	// LDY
	case 0xA0:
		return AddressingInfo{AddrImmediate, 2, 2}
	case 0xA4:
		return AddressingInfo{AddrZeroPage, 2, 3}
	case 0xB4:
		return AddressingInfo{AddrZeroPageX, 2, 4}
	case 0xAC:
		return AddressingInfo{AddrAbsolute, 3, 4}
	case 0xBC:
		return AddressingInfo{AddrAbsoluteX, 3, 4}
		
	// STA
	case 0x85:
		return AddressingInfo{AddrZeroPage, 2, 3}
	case 0x95:
		return AddressingInfo{AddrZeroPageX, 2, 4}
	case 0x8D:
		return AddressingInfo{AddrAbsolute, 3, 4}
	case 0x9D:
		return AddressingInfo{AddrAbsoluteX, 3, 5}
	case 0x99:
		return AddressingInfo{AddrAbsoluteY, 3, 5}
	case 0x81:
		return AddressingInfo{AddrIndexedIndirect, 2, 6}
	case 0x91:
		return AddressingInfo{AddrIndirectIndexed, 2, 6}
		
	// ADC
	case 0x69:
		return AddressingInfo{AddrImmediate, 2, 2}
	case 0x65:
		return AddressingInfo{AddrZeroPage, 2, 3}
	case 0x75:
		return AddressingInfo{AddrZeroPageX, 2, 4}
	case 0x6D:
		return AddressingInfo{AddrAbsolute, 3, 4}
	case 0x7D:
		return AddressingInfo{AddrAbsoluteX, 3, 4}
	case 0x79:
		return AddressingInfo{AddrAbsoluteY, 3, 4}
	case 0x61:
		return AddressingInfo{AddrIndexedIndirect, 2, 6}
	case 0x71:
		return AddressingInfo{AddrIndirectIndexed, 2, 5}
		
	// SBC
	case 0xE9:
		return AddressingInfo{AddrImmediate, 2, 2}
	case 0xE5:
		return AddressingInfo{AddrZeroPage, 2, 3}
	case 0xF5:
		return AddressingInfo{AddrZeroPageX, 2, 4}
	case 0xED:
		return AddressingInfo{AddrAbsolute, 3, 4}
	case 0xFD:
		return AddressingInfo{AddrAbsoluteX, 3, 4}
	case 0xF9:
		return AddressingInfo{AddrAbsoluteY, 3, 4}
	case 0xE1:
		return AddressingInfo{AddrIndexedIndirect, 2, 6}
	case 0xF1:
		return AddressingInfo{AddrIndirectIndexed, 2, 5}
		
	// CMP
	case 0xC9:
		return AddressingInfo{AddrImmediate, 2, 2}
	case 0xC5:
		return AddressingInfo{AddrZeroPage, 2, 3}
	case 0xD5:
		return AddressingInfo{AddrZeroPageX, 2, 4}
	case 0xCD:
		return AddressingInfo{AddrAbsolute, 3, 4}
	case 0xDD:
		return AddressingInfo{AddrAbsoluteX, 3, 4}
	case 0xD9:
		return AddressingInfo{AddrAbsoluteY, 3, 4}
	case 0xC1:
		return AddressingInfo{AddrIndexedIndirect, 2, 6}
	case 0xD1:
		return AddressingInfo{AddrIndirectIndexed, 2, 5}
		
	// Transfer instructions
	case 0xAA, 0x8A, 0xA8, 0x98, 0x9A, 0xBA:
		return AddressingInfo{AddrImplied, 1, 2}
		
	// Flag instructions
	case 0x18, 0x38, 0x58, 0x78, 0xB8, 0xD8, 0xF8:
		return AddressingInfo{AddrImplied, 1, 2}
		
	// Stack instructions
	case 0x48, 0x68:
		return AddressingInfo{AddrImplied, 1, 3}
	case 0x08, 0x28:
		return AddressingInfo{AddrImplied, 1, 4}
		
	// NOP
	case 0xEA:
		return AddressingInfo{AddrImplied, 1, 2}
	}
	
	// For opcodes not in our table, return default
	if int(opcode) < len(addressingTable) {
		return addressingTable[opcode]
	}
	
	return AddressingInfo{AddrImplied, 1, 2}
}

// getOperandAddress resolves the operand address for an addressing mode
func (c *CPU) getOperandAddress(mode AddressingMode) (uint16, bool) {
	pageCrossed := false
	
	switch mode {
	case AddrImplied:
		return 0, false
		
	case AddrAccumulator:
		return 0, false
		
	case AddrImmediate:
		addr := c.PC
		c.PC++
		return addr, false
		
	case AddrZeroPage:
		addr := uint16(c.read(c.PC))
		c.PC++
		return addr, false
		
	case AddrZeroPageX:
		addr := uint16(c.read(c.PC) + c.X)
		c.PC++
		return addr & 0xFF, false
		
	case AddrZeroPageY:
		addr := uint16(c.read(c.PC) + c.Y)
		c.PC++
		return addr & 0xFF, false
		
	case AddrRelative:
		offset := int8(c.read(c.PC))
		c.PC++
		addr := uint16(int32(c.PC) + int32(offset))
		pageCrossed = (c.PC & 0xFF00) != (addr & 0xFF00)
		return addr, pageCrossed
		
	case AddrAbsolute:
		addr := c.read16(c.PC)
		c.PC += 2
		return addr, false
		
	case AddrAbsoluteX:
		base := c.read16(c.PC)
		c.PC += 2
		addr := base + uint16(c.X)
		pageCrossed = (base & 0xFF00) != (addr & 0xFF00)
		
		// Perform dummy read if page boundary is crossed
		if pageCrossed {
			// Dummy read from (base + X) without carry
			dummyAddr := (base & 0xFF00) | ((base + uint16(c.X)) & 0xFF)
			c.read(dummyAddr)
		}
		
		return addr, pageCrossed
		
	case AddrAbsoluteY:
		base := c.read16(c.PC)
		c.PC += 2
		addr := base + uint16(c.Y)
		pageCrossed = (base & 0xFF00) != (addr & 0xFF00)
		
		// Perform dummy read if page boundary is crossed
		if pageCrossed {
			// Dummy read from (base + Y) without carry
			dummyAddr := (base & 0xFF00) | ((base + uint16(c.Y)) & 0xFF)
			c.read(dummyAddr)
		}
		
		return addr, pageCrossed
		
	case AddrIndirect:
		// Used only by JMP - has page boundary bug
		ptr := c.read16(c.PC)
		c.PC += 2
		if ptr&0xFF == 0xFF {
			// Bug: crosses page boundary
			lo := c.read(ptr)
			hi := c.read(ptr & 0xFF00)
			return uint16(hi)<<8 | uint16(lo), false
		}
		return c.read16(ptr), false
		
	case AddrIndexedIndirect: // (zp,X)
		base := c.read(c.PC)
		c.PC++
		ptr := (uint16(base) + uint16(c.X)) & 0xFF
		lo := c.read(ptr)
		hi := c.read((ptr + 1) & 0xFF)
		addr := uint16(hi)<<8 | uint16(lo)
		// Debug logging
		//fmt.Printf("IndexedIndirect: base=%02X, X=%02X, ptr=%02X, lo=%02X, hi=%02X, addr=%04X\n", 
		//	base, c.X, ptr, lo, hi, addr)
		return addr, false
		
	case AddrIndirectIndexed: // (zp),Y
		base := c.read(c.PC)
		c.PC++
		lo := c.read(uint16(base))
		hi := c.read((uint16(base) + 1) & 0xFF)
		baseAddr := uint16(hi)<<8 | uint16(lo)
		addr := baseAddr + uint16(c.Y)
		pageCrossed = (baseAddr & 0xFF00) != (addr & 0xFF00)
		
		// Perform dummy read if page boundary is crossed
		if pageCrossed {
			// Dummy read from (baseAddr + Y) without carry
			dummyAddr := (baseAddr & 0xFF00) | ((baseAddr + uint16(c.Y)) & 0xFF)
			c.read(dummyAddr)
		}
		return addr, pageCrossed
	}
	
	return 0, false
}

// getOperand gets the operand value for an addressing mode
func (c *CPU) getOperand(mode AddressingMode) (uint8, bool) {
	switch mode {
	case AddrAccumulator:
		return c.A, false
		
	case AddrImmediate:
		addr, _ := c.getOperandAddress(mode)
		return c.read(addr), false
		
	default:
		addr, pageCrossed := c.getOperandAddress(mode)
		return c.read(addr), pageCrossed
	}
}