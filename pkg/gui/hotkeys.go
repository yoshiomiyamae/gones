// Package gui — emulator hotkey dispatch.
//
// handleHotkey decides which key events are emulator commands (Esc/Tab/Fn/
// number keys with optional modifiers) vs. game inputs that should pass
// through to the InputManager. Release events for any key the table claims
// are consumed too, so the InputManager never sees a half-press it would
// interpret as a stuck button release.
package gui

import (
	"github.com/veandco/go-sdl2/sdl"
	"github.com/yoshiomiyamaegones/pkg/logger"
)

// hotkey is one entry in hotkeyTable. The handler runs when key+modMask
// matches a key-down event; repeatOk controls whether OS key-repeats fire
// it again or only the initial press does.
type hotkey struct {
	key      sdl.Keycode   // e.g. sdl.K_F12
	modMask  uint16        // required modifier bits (0 = none required); matches Keysym.Mod
	action   func(*NESGUI) // what to do on press
	repeatOk bool          // true: also fires on key-repeat; false: initial press only
}

// quit, toggleTurbo, toggleFPS, etc. are tiny adapters so the table can be
// expressed as plain function values. Inlined methods would also work but
// these read more directly.
func (g *NESGUI) quit()        { g.running = false }
func (g *NESGUI) toggleTurbo() { g.turbo = !g.turbo; logger.LogInfo("Turbo mode: %v", g.turbo) }
func (g *NESGUI) toggleFPS()   { g.showFPS = !g.showFPS }
func (g *NESGUI) resetNES()    { g.nes.SoftReset(); logger.LogInfo("NES reset") }
func (g *NESGUI) toggleCheats() {
	on := g.nes.Cheats.ToggleAll()
	logger.LogInfo("Cheats (%d loaded): %v", g.nes.Cheats.Count(), on)
}
func (g *NESGUI) toggleFilter() {
	logger.LogInfo("Analog filter chain: %v", g.nes.APU.ToggleFilter())
}

// hotkeyTable lists the simple, fixed-modifier hotkeys. F1-F10 (variable
// Ctrl modifier for save vs. load) is handled inline in handleHotkey
// because expressing "Ctrl optional" in this table would hurt readability
// more than it helps.
var hotkeyTable = []hotkey{
	{sdl.K_ESCAPE, 0, (*NESGUI).quit, true},
	{sdl.K_TAB, 0, (*NESGUI).toggleTurbo, false},
	{sdl.K_F11, 0, (*NESGUI).toggleFPS, true},
	{sdl.K_F12, 0, (*NESGUI).saveScreenshot, true},
	{sdl.K_r, sdl.KMOD_CTRL, (*NESGUI).resetNES, false},
	{sdl.K_h, sdl.KMOD_CTRL, (*NESGUI).toggleCheats, false},
	{sdl.K_e, sdl.KMOD_CTRL, (*NESGUI).toggleRecording, false},
	{sdl.K_6, 0, (*NESGUI).toggleFilter, false},
}

// isHotkeyKey reports whether a key is one of those the emulator owns. Used
// for release-event consumption so the InputManager doesn't see a phantom
// game-button release for a key that was never a game button to begin with.
func isHotkeyKey(k sdl.Keycode) bool {
	return k == sdl.K_ESCAPE || k == sdl.K_TAB ||
		(k >= sdl.K_F1 && k <= sdl.K_F12) ||
		(k >= sdl.K_1 && k <= sdl.K_6)
}

// handleHotkey processes emulator-level hotkeys (Esc/Tab/F1-F12/1-6).
// Returns true if the event was consumed; false means it's a game input and
// should be forwarded to the InputManager.
func (g *NESGUI) handleHotkey(e *sdl.KeyboardEvent) bool {
	if e.State != sdl.PRESSED {
		// Release events for hotkey keys are still "consumed" so the input
		// manager doesn't see them as game button releases.
		return isHotkeyKey(e.Keysym.Sym)
	}

	// Table-driven simple hotkeys. Mirrors the original switch's behaviour:
	// a row that matches key+modMask but fails the repeat gate is *not*
	// consumed — the original switch falls through, returning false, so the
	// event is forwarded to the InputManager (which ignores it anyway since
	// none of these keys are game buttons, but the contract is preserved).
	for _, hk := range hotkeyTable {
		if e.Keysym.Sym != hk.key {
			continue
		}
		if hk.modMask != 0 && e.Keysym.Mod&hk.modMask == 0 {
			continue
		}
		if !hk.repeatOk && e.Repeat != 0 {
			continue
		}
		hk.action(g)
		return true
	}

	// Channel mute (1-5): toggle channel `key - 1` on the APU.
	if e.Keysym.Sym >= sdl.K_1 && e.Keysym.Sym <= sdl.K_5 && e.Repeat == 0 {
		muted, name := g.nes.APU.ToggleChannelMute(int(e.Keysym.Sym - sdl.K_1))
		state := "ON"
		if muted {
			state = "MUTED"
		}
		logger.LogInfo("Channel %s: %s", name, state)
		return true
	}

	// Save/load state slots (F1-F10). Ctrl+Fn loads, Fn alone saves. Kept
	// out of the table because "modifier is optional and selects behaviour"
	// doesn't model as a single table row cleanly.
	if e.Keysym.Sym >= sdl.K_F1 && e.Keysym.Sym <= sdl.K_F10 {
		slot := int(e.Keysym.Sym-sdl.K_F1) + 1
		if e.Keysym.Mod&sdl.KMOD_CTRL != 0 {
			g.loadStateSlot(slot)
		} else {
			g.saveStateSlot(slot)
		}
		return true
	}

	return false
}
