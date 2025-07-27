package cpu

import (
	"github.com/yoshiomiyamaegones/pkg/logger"
	"github.com/yoshiomiyamaegones/pkg/memory"
)

// CPU represents the 6502 processor
type CPU struct {
	// Registers
	A  uint8  // Accumulator
	X  uint8  // X register
	Y  uint8  // Y register
	SP uint8  // Stack pointer
	PC uint16 // Program counter
	P  uint8  // Status register

	// Memory interface
	Memory *memory.Memory

	// Cycle counting
	Cycles int

	// Interrupt flags
	NMI bool
	IRQ bool

	// Debug fields for freeze detection
	lastPC       uint16
	stuckCounter int
}

// Status flag bits
const (
	FlagCarry     = 1 << 0 // C
	FlagZero      = 1 << 1 // Z
	FlagInterrupt = 1 << 2 // I
	FlagDecimal   = 1 << 3 // D
	FlagBreak     = 1 << 4 // B
	FlagUnused    = 1 << 5 // -
	FlagOverflow  = 1 << 6 // V
	FlagNegative  = 1 << 7 // N
)

// New creates a new CPU instance
func New(mem *memory.Memory) *CPU {
	return &CPU{
		Memory: mem,
		SP:     0xFD,
		P:      FlagUnused | FlagInterrupt,
	}
}

// Reset resets the CPU to initial state
func (c *CPU) Reset() {
	c.A = 0
	c.X = 0
	c.Y = 0
	c.SP = 0xFD
	c.P = FlagUnused | FlagInterrupt

	// Read reset vector
	resetVector := c.read16(0xFFFC)
	c.PC = resetVector
	c.Cycles = 0
}

// Step executes one instruction and returns cycles taken
func (c *CPU) Step() int {
	// Handle interrupts
	if c.NMI {
		logger.LogCPU("NMI triggered at PC=$%04X", c.PC)
		c.handleNMI()
		c.NMI = false
		return 7
	}

	if c.IRQ && !c.getFlag(FlagInterrupt) {
		// Temporarily disable IRQ handling to prevent freezes
		c.IRQ = false // Clear IRQ to prevent infinite loop
		logger.LogCPU("IRQ triggered but disabled to prevent freeze at PC=$%04X", c.PC)
		// c.handleIRQ()
		// return 7
	}

	// Optimized debug check - only in debug builds
	// Removed frequent stuck detection for performance

	// Fetch instruction
	opcode := c.read(c.PC)

	c.PC++

	// Execute instruction
	cycles := c.executeInstruction(opcode)
	c.Cycles += cycles

	return cycles
}

// executeInstruction is implemented in instructions.go

// handleNMI handles Non-Maskable Interrupt
func (c *CPU) handleNMI() {
	logger.LogCPU("NMI triggered: PC=$%04X, pushing to stack", c.PC)
	c.push16(c.PC)
	c.push(c.P)
	c.setFlag(FlagInterrupt, true)
	nmiVector := c.read16(0xFFFA)
	logger.LogCPU("NMI vector: $%04X, jumping to NMI handler", nmiVector)
	c.PC = nmiVector
}

// handleIRQ handles Interrupt Request
func (c *CPU) handleIRQ() {
	c.push16(c.PC)
	c.push(c.P)
	c.setFlag(FlagInterrupt, true)
	c.PC = c.read16(0xFFFE)
}

// Flag operations
func (c *CPU) getFlag(flag uint8) bool {
	return c.P&flag != 0
}

func (c *CPU) setFlag(flag uint8, value bool) {
	if value {
		c.P |= flag
	} else {
		c.P &^= flag
	}
}

// Memory operations
func (c *CPU) read(addr uint16) uint8 {
	return c.Memory.Read(addr)
}

func (c *CPU) write(addr uint16, value uint8) {
	c.Memory.Write(addr, value)
}

func (c *CPU) read16(addr uint16) uint16 {
	lo := uint16(c.read(addr))
	hi := uint16(c.read(addr + 1))
	return hi<<8 | lo
}

// Stack operations
func (c *CPU) push(value uint8) {
	c.write(0x100|uint16(c.SP), value)
	c.SP--
}

func (c *CPU) pop() uint8 {
	c.SP++
	return c.read(0x100 | uint16(c.SP))
}

func (c *CPU) push16(value uint16) {
	c.push(uint8(value >> 8))
	c.push(uint8(value & 0xFF))
}

func (c *CPU) pop16() uint16 {
	lo := uint16(c.pop())
	hi := uint16(c.pop())
	return hi<<8 | lo
}

// TriggerNMI triggers a Non-Maskable Interrupt
func (c *CPU) TriggerNMI() {
	c.NMI = true
}

// TriggerIRQ triggers an Interrupt Request
func (c *CPU) TriggerIRQ() {
	c.IRQ = true
}

// GetFlag returns the state of a flag (public method for testing)
func (c *CPU) GetFlag(flag uint8) bool {
	return c.getFlag(flag)
}
