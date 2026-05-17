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

// Reset performs a power-on reset: A,X,Y=0, P=$34 (B|I|U set in the
// pushed copy), S=$FD, PC loaded from $FFFC. blargg's cpu_reset suite
// distinguishes this from SoftReset.
func (c *CPU) Reset() {
	c.A = 0
	c.X = 0
	c.Y = 0
	c.SP = 0xFD
	c.P = FlagUnused | FlagInterrupt
	c.PC = c.read16(0xFFFC)
	c.Cycles = 0
}

// SoftReset models the user pressing the reset button: A,X,Y are
// untouched, I is forced set, and S decrements by 3 (the reset
// "pushes" 3 bytes but the writes are suppressed by the reset line —
// the stack contents are preserved).
func (c *CPU) SoftReset() {
	c.SP -= 3
	c.setFlag(FlagInterrupt, true)
	c.PC = c.read16(0xFFFC)
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
