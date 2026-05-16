// Package cheat implements Game Genie and raw address-poke cheats. Cheats
// are read-intercepted: when the CPU reads a patched address, the manager
// returns the cheat byte instead of the underlying memory/ROM byte. This
// covers both Game Genie ROM patches ($8000-$FFFF) and PAR-style RAM
// pokes ($0000-$1FFF) with the same code path — without needing to write
// back into RAM each frame.
package cheat

import (
	"fmt"
)

// Cheat is a single decoded patch entry. The compare byte is honored only
// when HasCompare is true (8-letter Game Genie codes and explicit
// `AAAA:VV:CC` raw entries).
type Cheat struct {
	Address    uint16
	Value      uint8
	Compare    uint8
	HasCompare bool
	Enabled    bool
	Source     string // original code text, for logging / display
	Comment    string // optional human-readable label from the .cht file
}

// Manager owns the list of loaded cheats and the global enable switch.
// All public methods are safe to call from the GUI thread; reads run from
// the CPU thread, but the slice is mutated only by Add / Toggle which the
// emulator currently invokes between frames (no concurrent reads).
type Manager struct {
	cheats  []Cheat
	enabled bool // global on/off — ToggleAll flips this
}

func NewManager() *Manager {
	return &Manager{enabled: true}
}

// Add appends a cheat. The caller has already decoded / parsed it.
func (m *Manager) Add(c Cheat) {
	c.Enabled = true
	m.cheats = append(m.cheats, c)
}

// Count returns the number of loaded cheats.
func (m *Manager) Count() int { return len(m.cheats) }

// Enabled reports whether the global switch is on.
func (m *Manager) Enabled() bool { return m.enabled }

// ToggleAll flips the global enable switch and returns the new state.
func (m *Manager) ToggleAll() bool {
	m.enabled = !m.enabled
	return m.enabled
}

// Apply returns the patched byte for addr if a matching enabled cheat
// exists, otherwise it returns current unchanged. current is the byte the
// underlying memory subsystem would have returned — needed for the
// compare-gated patches (and surfaces it for future logging).
func (m *Manager) Apply(addr uint16, current uint8) uint8 {
	if !m.enabled {
		return current
	}
	for i := range m.cheats {
		c := &m.cheats[i]
		if !c.Enabled || c.Address != addr {
			continue
		}
		if c.HasCompare && c.Compare != current {
			continue
		}
		return c.Value
	}
	return current
}

// List returns the cheats for display / save-state. The returned slice
// shares storage with the manager — callers must not mutate.
func (m *Manager) List() []Cheat { return m.cheats }

// String renders a cheat as it would appear in a .cht file.
func (c Cheat) String() string {
	if c.Source != "" {
		return c.Source
	}
	if c.HasCompare {
		return fmt.Sprintf("%04X:%02X:%02X", c.Address, c.Value, c.Compare)
	}
	return fmt.Sprintf("%04X:%02X", c.Address, c.Value)
}
