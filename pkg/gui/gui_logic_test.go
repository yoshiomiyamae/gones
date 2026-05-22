package gui

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/yoshiomiyamaegones/pkg/input"
	"github.com/yoshiomiyamaegones/pkg/nes"
)

// newTestGUI builds a NESGUI with only the SDL-free fields populated. The
// window/renderer/texture/audio handles stay nil — tests must not call methods
// that touch them (NewNESGUI/Run/render/saveScreenshot/updateWindowTitle).
func newTestGUI(romPath string) *NESGUI {
	return &NESGUI{
		nes:     nes.NewNES(),
		romPath: romPath,
		running: true,
		showFPS: true,
		fpsTimer: time.Now(),
	}
}

func keyEvent(sym sdl.Keycode, mod uint16, pressed bool, repeat uint8) *sdl.KeyboardEvent {
	state := uint8(sdl.RELEASED)
	if pressed {
		state = sdl.PRESSED
	}
	return &sdl.KeyboardEvent{
		Type:   sdl.KEYDOWN,
		State:  state,
		Repeat: repeat,
		Keysym: sdl.Keysym{Sym: sym, Mod: mod},
	}
}

// --- input.go ---

func TestInputKeyboardMapping(t *testing.T) {
	system := nes.NewNES()
	im := NewInputManager(system)
	im.Initialize()
	ctrl := system.GetInput()

	keys := []struct {
		sym  sdl.Keycode
		mask uint8
	}{
		{sdl.K_z, input.ButtonMaskA},
		{sdl.K_x, input.ButtonMaskB},
		{sdl.K_a, input.ButtonMaskSelect},
		{sdl.K_s, input.ButtonMaskStart},
		{sdl.K_UP, input.ButtonMaskUp},
		{sdl.K_DOWN, input.ButtonMaskDown},
		{sdl.K_LEFT, input.ButtonMaskLeft},
		{sdl.K_RIGHT, input.ButtonMaskRight},
	}
	for _, k := range keys {
		im.HandleEvent(keyEvent(k.sym, 0, true, 0))
		if ctrl.GetButtons()&k.mask == 0 {
			t.Errorf("key %d press: button mask %#02x not set", k.sym, k.mask)
		}
		im.HandleEvent(keyEvent(k.sym, 0, false, 0))
		if ctrl.GetButtons()&k.mask != 0 {
			t.Errorf("key %d release: button mask %#02x still set", k.sym, k.mask)
		}
	}

	// An unmapped key is ignored without affecting state.
	im.HandleEvent(keyEvent(sdl.K_q, 0, true, 0))
	if ctrl.GetButtons() != 0 {
		t.Errorf("unmapped key changed state: %#02x", ctrl.GetButtons())
	}
}

func TestInputDispatchAndSlots(t *testing.T) {
	im := NewInputManager(nes.NewNES())

	// With no devices opened, slot lookups return -1 and the controller/
	// joystick handlers early-return (their dispatch arms still run).
	if im.gameControllerSlot(0) != -1 || im.joystickSlot(0) != -1 {
		t.Error("empty slot lookups should be -1")
	}
	events := []sdl.Event{
		&sdl.ControllerButtonEvent{Type: sdl.CONTROLLERBUTTONDOWN, Which: 0, Button: sdl.CONTROLLER_BUTTON_A, State: sdl.PRESSED},
		&sdl.ControllerAxisEvent{Type: sdl.CONTROLLERAXISMOTION, Which: 0, Axis: sdl.CONTROLLER_AXIS_LEFTX, Value: 9000},
		&sdl.JoyButtonEvent{Type: sdl.JOYBUTTONDOWN, Which: 0, Button: 0, State: sdl.PRESSED},
		&sdl.JoyAxisEvent{Type: sdl.JOYAXISMOTION, Which: 0, Axis: 0, Value: -9000},
		&sdl.JoyHatEvent{Type: sdl.JOYHATMOTION, Which: 0, Value: sdl.HAT_UP},
		&sdl.ControllerDeviceEvent{Type: sdl.CONTROLLERDEVICEREMOVED, Which: 0},
		&sdl.JoyDeviceRemovedEvent{Type: sdl.JOYDEVICEREMOVED, Which: 0},
	}
	for _, e := range events {
		if !im.HandleEvent(e) {
			t.Errorf("HandleEvent(%T) returned false", e)
		}
	}

	// An unhandled event type returns false.
	if im.HandleEvent(&sdl.MouseMotionEvent{Type: sdl.MOUSEMOTION}) {
		t.Error("HandleEvent(mouse) should return false")
	}

	// Cleanup with no devices is a safe no-op.
	im.Cleanup()
}

// --- recorder.go ---

func TestWAVRecorderRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rec.wav")
	rec, err := newWAVRecorder(path, 44100)
	if err != nil {
		t.Fatalf("newWAVRecorder: %v", err)
	}
	// Empty write is a no-op; values beyond ±1 clamp.
	if err := rec.WriteSamples(nil); err != nil {
		t.Fatalf("WriteSamples(nil): %v", err)
	}
	if err := rec.WriteSamples([]float32{0, 0.25, -0.25, 2.0, -2.0}); err != nil {
		t.Fatalf("WriteSamples: %v", err)
	}
	if err := rec.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		t.Fatalf("bad RIFF header")
	}
	// 5 samples × 2 bytes = 10 data bytes; header patches reflect that.
	gotData := binary.LittleEndian.Uint32(data[40:44])
	if gotData != 10 {
		t.Errorf("data size = %d, want 10", gotData)
	}
	gotRiff := binary.LittleEndian.Uint32(data[4:8])
	if gotRiff != 36+10 {
		t.Errorf("riff size = %d, want 46", gotRiff)
	}
	// Clamped sample 2.0 -> +full scale 32767.
	clamped := int16(binary.LittleEndian.Uint16(data[wavHeaderSize+6 : wavHeaderSize+8]))
	if clamped != 32767 {
		t.Errorf("clamped sample = %d, want 32767", clamped)
	}
}

func TestNewWAVRecorderBadPath(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "nope", "rec.wav")
	if _, err := newWAVRecorder(bad, 44100); err == nil {
		t.Error("newWAVRecorder with unwritable path should error")
	}
}

func TestRecordingPathAndToggle(t *testing.T) {
	// No ROM: falls back to a working-directory name.
	g := newTestGUI("")
	if p := g.recordingPath(); p == "" {
		t.Error("recordingPath should never be empty")
	}

	dir := t.TempDir()
	g = newTestGUI(filepath.Join(dir, "game.nes"))
	g.toggleRecording() // start
	if g.recorder == nil {
		t.Fatal("toggleRecording should start a recording")
	}
	_ = g.recorder.WriteSamples([]float32{0.1, -0.1})
	g.toggleRecording() // stop
	if g.recorder != nil {
		t.Fatal("toggleRecording should stop the recording")
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "game.*.wav"))
	if len(matches) != 1 {
		t.Errorf("expected 1 recording file, found %d", len(matches))
	}
}

// --- state.go ---

func TestStateSlotsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	g := newTestGUI(filepath.Join(dir, "game.nes"))

	if got := g.stateSlotPath(3); got != filepath.Join(dir, "game.state3") {
		t.Errorf("stateSlotPath = %q", got)
	}

	g.saveStateSlot(1)
	if _, err := os.Stat(g.stateSlotPath(1)); err != nil {
		t.Fatalf("save state file missing: %v", err)
	}
	g.loadStateSlot(1) // round-trips back through LoadState

	// Missing-ROM-path branches just log and return (no panic, no file).
	noRom := newTestGUI("")
	noRom.saveStateSlot(1)
	noRom.loadStateSlot(1)

	// Loading a non-existent slot logs an error and returns.
	g.loadStateSlot(9)
}

func TestLoadCheats(t *testing.T) {
	dir := t.TempDir()
	romPath := filepath.Join(dir, "game.nes")
	if err := os.WriteFile(filepath.Join(dir, "game.cht"), []byte("00FF:42\n0001:01\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	g := newTestGUI(romPath)
	g.loadCheats()
	if g.nes.Cheats.Count() != 2 {
		t.Errorf("loaded %d cheats, want 2", g.nes.Cheats.Count())
	}

	// No .cht file present → silent no-op.
	g2 := newTestGUI(filepath.Join(dir, "other.nes"))
	g2.loadCheats()
	if g2.nes.Cheats.Count() != 0 {
		t.Errorf("expected 0 cheats for missing file, got %d", g2.nes.Cheats.Count())
	}

	// No ROM path → returns immediately.
	newTestGUI("").loadCheats()
}

func TestSaveFramebufferAsRaw(t *testing.T) {
	g := newTestGUI("")
	path := filepath.Join(t.TempDir(), "fb.raw")
	g.saveFramebufferAsRaw(path, []byte{1, 2, 3, 4})
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(data) != 4 {
		t.Errorf("wrote %d bytes, want 4", len(data))
	}

	// Unwritable path logs an error and returns without panicking.
	g.saveFramebufferAsRaw(filepath.Join(t.TempDir(), "nope", "fb.raw"), []byte{1})
}

// --- hotkeys.go ---

func TestHotkeySimpleActions(t *testing.T) {
	g := newTestGUI("")

	if !g.handleHotkey(keyEvent(sdl.K_ESCAPE, 0, true, 0)) || g.running {
		t.Error("Esc should quit (running=false) and be consumed")
	}
	g.running = true

	if !g.handleHotkey(keyEvent(sdl.K_TAB, 0, true, 0)) || !g.turbo {
		t.Error("Tab should enable turbo and be consumed")
	}
	// Tab on key-repeat is NOT consumed (repeatOk=false).
	if g.handleHotkey(keyEvent(sdl.K_TAB, 0, true, 1)) {
		t.Error("Tab key-repeat should not be consumed")
	}

	if !g.handleHotkey(keyEvent(sdl.K_F11, 0, true, 0)) || g.showFPS {
		t.Error("F11 should toggle FPS off and be consumed")
	}

	if !g.handleHotkey(keyEvent(sdl.K_r, sdl.KMOD_CTRL, true, 0)) {
		t.Error("Ctrl+R (reset) should be consumed")
	}
	if !g.handleHotkey(keyEvent(sdl.K_h, sdl.KMOD_CTRL, true, 0)) {
		t.Error("Ctrl+H (cheats) should be consumed")
	}
	// Ctrl-less R is not the reset hotkey; r is also not a game key → false.
	if g.handleHotkey(keyEvent(sdl.K_r, 0, true, 0)) {
		t.Error("R without Ctrl should not match the reset hotkey")
	}

	for _, k := range []sdl.Keycode{sdl.K_6, sdl.K_7, sdl.K_8} {
		if !g.handleHotkey(keyEvent(k, 0, true, 0)) {
			t.Errorf("key %d should be consumed", k)
		}
	}
	if !g.nes.PPU.NoSpriteLimit {
		t.Error("K_8 should have toggled NoSpriteLimit on")
	}
}

func TestHotkeyChannelMute(t *testing.T) {
	g := newTestGUI("")
	for k := sdl.Keycode(sdl.K_1); k <= sdl.K_5; k++ {
		if !g.handleHotkey(keyEvent(k, 0, true, 0)) {
			t.Errorf("channel-mute key %d should be consumed", k)
		}
	}
	// Key-repeat for channel mute is ignored (not consumed by that branch).
	if g.handleHotkey(keyEvent(sdl.K_1, 0, true, 1)) {
		t.Error("channel-mute key-repeat should not be consumed")
	}
}

func TestHotkeyStateSlots(t *testing.T) {
	dir := t.TempDir()
	g := newTestGUI(filepath.Join(dir, "game.nes"))

	// F1 saves slot 1.
	if !g.handleHotkey(keyEvent(sdl.K_F1, 0, true, 0)) {
		t.Error("F1 should be consumed")
	}
	if _, err := os.Stat(g.stateSlotPath(1)); err != nil {
		t.Fatalf("F1 should have saved slot 1: %v", err)
	}
	// Ctrl+F1 loads slot 1.
	if !g.handleHotkey(keyEvent(sdl.K_F1, sdl.KMOD_CTRL, true, 0)) {
		t.Error("Ctrl+F1 should be consumed")
	}
}

func TestHotkeyReleaseConsumption(t *testing.T) {
	g := newTestGUI("")
	// Release of a hotkey key is consumed so the InputManager ignores it.
	if !g.handleHotkey(keyEvent(sdl.K_ESCAPE, 0, false, 0)) {
		t.Error("Esc release should be consumed")
	}
	// Release of a game key is NOT consumed (forwarded to InputManager).
	if g.handleHotkey(keyEvent(sdl.K_z, 0, false, 0)) {
		t.Error("game-key release should not be consumed")
	}
	// A plain unmapped press is not consumed.
	if g.handleHotkey(keyEvent(sdl.K_z, 0, true, 0)) {
		t.Error("game-key press should not be consumed by hotkey handler")
	}
}

func TestIsHotkeyKey(t *testing.T) {
	for _, k := range []sdl.Keycode{sdl.K_ESCAPE, sdl.K_TAB, sdl.K_F1, sdl.K_F12, sdl.K_1, sdl.K_8} {
		if !isHotkeyKey(k) {
			t.Errorf("isHotkeyKey(%d) = false, want true", k)
		}
	}
	for _, k := range []sdl.Keycode{sdl.K_z, sdl.K_9, sdl.K_q} {
		if isHotkeyKey(k) {
			t.Errorf("isHotkeyKey(%d) = true, want false", k)
		}
	}
}

// --- timing.go ---

func TestUpdateFPS(t *testing.T) {
	g := newTestGUI("")
	g.fpsTimer = time.Now().Add(-time.Second) // force the >=500ms update branch
	g.fpsCounter = 30
	g.updateFPS()
	if g.currentFPS <= 0 {
		t.Errorf("currentFPS = %f, want > 0", g.currentFPS)
	}
	if g.fpsCounter != 0 {
		t.Errorf("fpsCounter should reset to 0, got %d", g.fpsCounter)
	}
}

func TestWaitForNextFrame(t *testing.T) {
	g := newTestGUI("")

	// Turbo: must return effectively immediately (no frame-pacing sleep).
	g.turbo = true
	t0 := time.Now()
	g.waitForNextFrame(t0, t0, 1)
	if elapsed := time.Since(t0); elapsed > 50*time.Millisecond {
		t.Errorf("turbo waitForNextFrame slept %v, want ~0", elapsed)
	}

	// Non-turbo with a deadline already an hour in the past: also no sleep.
	// frameCount=60 exercises the periodic timing-deviation log branch.
	g.turbo = false
	t1 := time.Now()
	g.waitForNextFrame(t1.Add(-time.Hour), t1, 60)
	if elapsed := time.Since(t1); elapsed > 50*time.Millisecond {
		t.Errorf("past-deadline waitForNextFrame slept %v, want ~0", elapsed)
	}
}
