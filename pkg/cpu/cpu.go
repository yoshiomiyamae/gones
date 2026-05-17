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

	// pendingIRQ latches a successful IRQ poll from the previous
	// instruction's end. The CPU samples its IRQ line during each
	// instruction; if I=0 at the sample, the IRQ sequence runs after the
	// current instruction completes (and before the next one starts).
	pendingIRQ bool

	// iWriteLate is set by CLI/SEI/PLP to mark that this instruction's I
	// write happens AFTER the IRQ poll cycle. The end-of-instruction poll
	// uses the pre-instruction I instead of the just-written value. This
	// is the documented "CLI/SEI/PLP delay" on the 6502 — blargg's
	// cli_latency test exercises every variant.
	iWriteLate bool

	// pollIIsSet is true when pollI is meaningful (the last CPU.Step
	// finished a normal instruction and captured its IRQ-poll I value).
	// PollIRQ runs AFTER the PPU/APU have advanced for this instruction's
	// cycles, so it sees IRQ assertions that happened mid-instruction.
	pollI      bool
	pollIIsSet bool

	// suppressPostPoll skips the end-of-instruction IRQ poll for the
	// just-finished instruction. A taken non-page-crossing branch sets
	// this — the 6502's poll for that case lands on the dummy "fix-up"
	// cycle and is dropped, so an IRQ asserted during the branch is
	// taken only AFTER the following instruction runs (blargg's
	// branch_delays_irq test).
	suppressPostPoll bool

	// extraCycles is added to the next CPU.Step's returned cycle count.
	// Currently used for OAM DMA (STA $4014 halts the CPU for 513 extra
	// cycles while the DMA controller copies 256 bytes into OAM); the
	// Quietust scanline test calibrates NMI-handler scanline positions
	// against the full ~517 cycle cost.
	extraCycles int
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
	if c.NMI {
		logger.LogCPU("NMI triggered at PC=$%04X", c.PC)
		c.handleNMI()
		c.NMI = false
		c.pollIIsSet = false
		return 7
	}

	// IRQ pending from the previous instruction's poll fires at this
	// instruction boundary, before the next opcode is fetched.
	if c.pendingIRQ {
		c.pendingIRQ = false
		c.handleIRQ()
		c.pollIIsSet = false
		return 7
	}

	// Snapshot I before the instruction runs; CLI/SEI/PLP's poll uses
	// this pre-write value.
	preI := c.getFlag(FlagInterrupt)

	opcode := c.read(c.PC)
	c.PC++

	cycles := c.executeInstruction(opcode)
	// Add any extra cycles charged by side effects (e.g. OAM DMA on a
	// $4014 write, which stalls the CPU for 513 cycles).
	cycles += c.extraCycles
	c.extraCycles = 0
	c.Cycles += cycles

	// Capture the I value the post-instruction IRQ poll should use. The
	// poll itself runs in PollIRQ, after the bus has caught up with this
	// instruction's PPU/APU side effects.
	c.pollI = c.getFlag(FlagInterrupt)
	if c.iWriteLate {
		c.pollI = preI
		c.iWriteLate = false
	}
	c.pollIIsSet = !c.suppressPostPoll
	c.suppressPostPoll = false

	return cycles
}

// PollIRQ samples the IRQ line for the just-completed instruction. Called
// by NES.Step after the PPU and APU have advanced through this
// instruction's cycles, so IRQ assertions that happened mid-instruction
// (MMC3 scanline counter, APU frame IRQ) are visible to the poll. A
// successful poll latches pendingIRQ, which the next Step services
// before fetching the next opcode.
func (c *CPU) PollIRQ() {
	if !c.pollIIsSet {
		return
	}
	c.pollIIsSet = false
	if c.IRQ && !c.pollI {
		c.pendingIRQ = true
	}
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
	c.extraCycles += c.Memory.Write(addr, value)
}

// rmwRead is the read-side of a read-modify-write instruction's bus
// pattern: the CPU reads the byte, then issues a "dummy" write of that
// same value at cycle 5 before writing the modified value at cycle 6.
// On open-bus targets (PPU $2007, OAM $2004) the dummy write has user-
// visible side effects — blargg's cpu_dummy_writes_ppumem relies on it.
func (c *CPU) rmwRead(addr uint16) uint8 {
	value := c.read(addr)
	c.write(addr, value)
	return value
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
