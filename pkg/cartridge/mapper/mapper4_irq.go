package mapper

// This file hosts the MMC3 IRQ subsystem: the A12 rising-edge notification
// path, the IRQ register-write handlers ($C000/$C001/$E000/$E001), and the
// public IRQ status API (IsIRQPending / ClearIRQ). The IRQ counter itself
// is clocked once per scanline from Step() in mapper4.go; NotifyA12 is kept
// here for interface compatibility but is intentionally a no-op (see the
// detailed comment on the function).

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

// NotifyA12 is retained for interface compatibility but no longer clocks the
// IRQ counter. The PPU now drives the MMC3 IRQ counter once per scanline at
// cycle 260 via Step(); routing every CHR access through here as well caused
// the counter to be clocked many times per scanline, firing IRQs at the wrong
// raster lines and breaking split-scroll/HUD rendering (e.g. SMB3 status bar).
func (m *Mapper4) NotifyA12(chrAddr uint16, renderingEnabled bool) {
	_ = chrAddr
	_ = renderingEnabled
}

// IsIRQPending returns true if an IRQ is pending
func (m *Mapper4) IsIRQPending() bool {
	return m.irqPending
}

// ClearIRQ clears the pending IRQ
func (m *Mapper4) ClearIRQ() {
	m.irqPending = false
}
