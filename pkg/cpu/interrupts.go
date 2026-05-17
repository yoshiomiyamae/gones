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
	c.vector(0xFFFA)
}

// handleIRQ handles Interrupt Request
func (c *CPU) handleIRQ() {
	c.push16(c.PC)
	c.push(c.P)
	c.vector(0xFFFE)
}

// vector loads PC from the requested interrupt vector, redirecting to the
// NMI vector when NMI is asserted right now (the "NMI hijack" quirk).
//
// Reachability: with whole-instruction stepping the c.NMI flag is only
// updated between CPU.Step calls, so the hijack catches NMIs already
// pending when handleIRQ/execBRK starts — but those are picked off
// first by the NMI-priority check in CPU.Step. Modelling a true hijack
// (NMI asserted between an IRQ/BRK's push and its vector fetch) needs
// sub-instruction CPU/PPU interleaving, which is why blargg's
// nmi_and_brk / nmi_and_irq remain skipped. The routing is correct, the
// timing isn't.
func (c *CPU) vector(addr uint16) {
	if c.NMI {
		c.NMI = false
		addr = 0xFFFA
	}
	c.setFlag(FlagInterrupt, true)
	c.PC = c.read16(addr)
}

// TriggerNMI triggers a Non-Maskable Interrupt
func (c *CPU) TriggerNMI() {
	c.NMI = true
}

// TriggerIRQ triggers an Interrupt Request
func (c *CPU) TriggerIRQ() {
	c.IRQ = true
}
