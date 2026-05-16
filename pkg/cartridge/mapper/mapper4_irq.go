package mapper

// This file hosts the MMC3 IRQ subsystem: the A12 rising-edge notification
// path, the IRQ register-write handlers ($C000/$C001/$E000/$E001), and the
// public IRQ status API (IsIRQPending / ClearIRQ). The IRQ counter is
// clocked by:
//   - Step() at PPU cycle 260 of each rendering-active scanline (covers
//     the BG→sprite A12 rising edge in the common BG=$0000/Sprites=$1000
//     layout); and
//   - NotifyA12 on CPU-driven $2006/$2007 accesses that flip A12 from 0→1
//     (blargg's mmc3_test suite drives the counter exclusively through this
//     path with rendering disabled).

import (
	"github.com/yoshiomiyamaegones/pkg/logger"
)

// writeIRQLatch handles a write to $C000 (IRQ latch / reload value).
func (m *Mapper4) writeIRQLatch(value uint8) {
	m.irqReloadValue = value
	logger.LogMapper("MMC3 IRQ latch set: %d", value)
}

// writeIRQReload handles a write to $C001 (request a counter reload on
// the next A12 rising edge).
func (m *Mapper4) writeIRQReload() {
	m.irqReloadFlag = true
	m.irqCounter = 0 // Clear counter
	logger.LogMapper("MMC3 IRQ reload triggered")
}

// writeIRQDisable handles a write to $E000 (disable IRQ and acknowledge
// any pending IRQ).
func (m *Mapper4) writeIRQDisable() {
	m.irqEnabled = false
	m.irqPending = false
	logger.LogMapper("MMC3 IRQ disabled")
}

// writeIRQEnable handles a write to $E001 (enable IRQ generation).
func (m *Mapper4) writeIRQEnable() {
	m.irqEnabled = true
	logger.LogMapper("MMC3 IRQ enabled")
}

// clockIRQ is the shared counter-tick used by both the per-scanline Step()
// path and the CPU-driven A12 rising-edge path. It implements the canonical
// MMC3 reload/decrement sequence: on reload-flag or zero, refill from the
// latch (without firing IRQ from that refill alone); otherwise decrement;
// fire IRQ when the post-tick counter is zero and IRQ is enabled.
func (m *Mapper4) clockIRQ() {
	if m.irqReloadFlag || m.irqCounter == 0 {
		m.irqCounter = m.irqReloadValue
		m.irqReloadFlag = false
	} else {
		m.irqCounter--
	}
	if m.irqCounter == 0 && m.irqEnabled {
		m.irqPending = true
	}
}

// NotifyA12 is called by the PPU on CPU-driven accesses that change the v
// register's A12 bit ($2006 second write, $2007 increment after R/W). The
// per-scanline rendering path uses Step() instead — the two never both fire
// for the same rising edge because rendering doesn't touch the CPU-side
// lastA12High tracker, and Step() resets it to false so the next CPU-driven
// $2006 = $1xxx write is correctly recognised as a fresh rising edge.
func (m *Mapper4) NotifyA12(chrAddr uint16, renderingEnabled bool) {
	_ = renderingEnabled
	newA12 := (chrAddr & 0x1000) != 0
	if newA12 && !m.lastA12High {
		m.clockIRQ()
	}
	m.lastA12High = newA12
}

// IsIRQPending returns true if an IRQ is pending
func (m *Mapper4) IsIRQPending() bool {
	return m.irqPending
}

// ClearIRQ clears the pending IRQ
func (m *Mapper4) ClearIRQ() {
	m.irqPending = false
}
