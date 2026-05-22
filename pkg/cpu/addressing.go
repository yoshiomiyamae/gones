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

// getWriteAddress resolves the target address for a store instruction,
// emitting the dummy read at the uncorrected target that the real 6502
// always issues on indexed addressing modes (abs,X / abs,Y / (zp),Y), even
// when no page boundary is crossed. blargg's test_ppu_read_buffer subtest
// 35 ("STA $2000,Y with Y=7 must trigger a $2007 dummy read") relies on
// this; getOperandAddress only emits the dummy read on actual page crosses
// (the read-instruction case).
func (c *CPU) getWriteAddress(mode AddressingMode) uint16 {
	switch mode {
	case AddrAbsoluteX:
		base := c.read16(c.PC)
		c.PC += 2
		return c.indexedWriteAddr(base, c.X)
	case AddrAbsoluteY:
		base := c.read16(c.PC)
		c.PC += 2
		return c.indexedWriteAddr(base, c.Y)
	case AddrIndirectIndexed: // (zp),Y
		base := c.read(c.PC)
		c.PC++
		lo := c.read(uint16(base))
		hi := c.read((uint16(base) + 1) & 0xFF)
		return c.indexedWriteAddr(uint16(hi)<<8|uint16(lo), c.Y)
	}
	addr, _ := c.getOperandAddress(mode)
	return addr
}

// indexedWriteAddr emits the uncorrected-address dummy read shared by every
// indexed-store mode and returns the corrected target.
func (c *CPU) indexedWriteAddr(base uint16, idx uint8) uint16 {
	addr := base + uint16(idx)
	c.read((base & 0xFF00) | (addr & 0xFF))
	return addr
}
