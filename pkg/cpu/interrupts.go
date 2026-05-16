// Package cpu — interrupts.go contains reset and interrupt-vector handling.
//
// The 6502 treats reset, NMI, and IRQ as variations on a single mechanism:
// jump through a hard-coded vector, optionally saving caller state. Keeping
// them together makes the family relationship explicit. Per-instruction
// dispatch (including the BRK opcode) lives in instructions.go.
package cpu

import (
	"github.com/yoshiomiyamaegones/pkg/logger"
)

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

// TriggerNMI triggers a Non-Maskable Interrupt
func (c *CPU) TriggerNMI() {
	c.NMI = true
}

// TriggerIRQ triggers an Interrupt Request
func (c *CPU) TriggerIRQ() {
	c.IRQ = true
}
