package cpu

import (
	"encoding/binary"
	"io"

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

	// irqInhibitOneInstruction is the CLI/SEI/PLP one-instruction-delay
	// quirk: on real 6502 the I-flag change from these opcodes takes effect
	// AFTER the next instruction completes, so an IRQ asserted while I was
	// previously set is serviced one instruction later than the flag change
	// alone would suggest. blargg's mmc3_test 4 (scanline_timing) relies on
	// this — the IRQ has to land after the `nop;nop;inc irq_flag` trio that
	// follows CLI, not immediately on the first NOP.
	irqInhibitOneInstruction bool
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

// Step executes one instruction and returns cycles taken
func (c *CPU) Step() int {
	// Handle interrupts
	if c.NMI {
		logger.LogCPU("NMI triggered at PC=$%04X", c.PC)
		c.handleNMI()
		c.NMI = false
		return 7
	}

	// CLI/SEI/PLP delay their I-flag effect by one instruction (the IRQ
	// poll for that next instruction sees the OLD I value). Model this as
	// a one-shot inhibit: the instruction that ran CLI sets the flag,
	// then the very next Step suppresses IRQ servicing, and the Step
	// after that resumes normal sampling against the now-current I.
	pollIRQ := c.IRQ && !c.getFlag(FlagInterrupt) && !c.irqInhibitOneInstruction
	c.irqInhibitOneInstruction = false

	if pollIRQ {
		c.handleIRQ()
		c.IRQ = false
		return 7
	}

	// Fetch instruction
	opcode := c.read(c.PC)

	c.PC++

	// Execute instruction
	cycles := c.executeInstruction(opcode)
	c.Cycles += cycles

	return cycles
}

// executeInstruction is implemented in instructions.go.
// handleNMI, handleIRQ, Reset, TriggerNMI, and TriggerIRQ are in interrupts.go.

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

// GetFlag returns the state of a flag (public method for testing)
func (c *CPU) GetFlag(flag uint8) bool {
	return c.getFlag(flag)
}

// cpuState is the on-disk layout for CPU state. Keeping it as a flat struct of
// fixed-size primitives lets binary.Write/Read handle the entire blob in one call.
type cpuState struct {
	A, X, Y, SP uint8
	PC          uint16
	P           uint8
	Cycles      int64 // widened from int for stable on-disk layout
	NMI, IRQ    bool
}

// SaveState writes the CPU's register / interrupt state to w.
func (c *CPU) SaveState(w io.Writer) error {
	return binary.Write(w, binary.LittleEndian, cpuState{
		A: c.A, X: c.X, Y: c.Y, SP: c.SP,
		PC: c.PC, P: c.P,
		Cycles: int64(c.Cycles),
		NMI:    c.NMI, IRQ: c.IRQ,
	})
}

// LoadState restores CPU state written by SaveState.
func (c *CPU) LoadState(r io.Reader) error {
	var s cpuState
	if err := binary.Read(r, binary.LittleEndian, &s); err != nil {
		return err
	}
	c.A, c.X, c.Y, c.SP = s.A, s.X, s.Y, s.SP
	c.PC, c.P = s.PC, s.P
	c.Cycles = int(s.Cycles)
	c.NMI, c.IRQ = s.NMI, s.IRQ
	return nil
}
