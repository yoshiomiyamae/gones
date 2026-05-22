package nes

import (
	"bytes"
	"errors"
	"testing"

	"github.com/yoshiomiyamaegones/pkg/cartridge"
)

// failWriter accepts bytes until `limit` total, then fails — used to drive
// SaveState's per-component error returns at different offsets.
type failWriter struct {
	written, limit int
}

func (w *failWriter) Write(p []byte) (int, error) {
	if w.written+len(p) > w.limit {
		rem := w.limit - w.written
		if rem < 0 {
			rem = 0
		}
		w.written = w.limit
		return rem, errors.New("forced write failure")
	}
	w.written += len(p)
	return len(p), nil
}

// testCartridge builds a minimal NROM (mapper 0) image: 16KB PRG filled with
// NOPs and a reset vector at $8000, plus 8KB CHR. Enough to run frames.
func testCartridge(t *testing.T) *cartridge.Cartridge {
	t.Helper()
	prg := make([]byte, 16384)
	for i := range prg {
		prg[i] = 0xEA // NOP
	}
	prg[0x3FFC] = 0x00 // reset vector low
	prg[0x3FFD] = 0x80 // reset vector high -> $8000
	chr := make([]byte, 8192)

	var buf bytes.Buffer
	buf.WriteString("NES\x1A")
	buf.WriteByte(1)             // PRG: 1×16KB
	buf.WriteByte(1)             // CHR: 1×8KB
	buf.WriteByte(0)             // flags6
	buf.WriteByte(0)             // flags7
	buf.Write(make([]byte, 8))   // flags8-10 + padding
	buf.Write(prg)
	buf.Write(chr)

	cart, err := cartridge.LoadFromReader(&buf)
	if err != nil {
		t.Fatalf("load synthetic cartridge: %v", err)
	}
	return cart
}

func TestNESRunAndGetters(t *testing.T) {
	n := NewNES()
	n.LoadCartridge(testCartridge(t))
	n.Reset()

	for i := 0; i < 3; i++ {
		n.StepFrame()
	}
	if got := n.GetFrame(); got != 3 {
		t.Errorf("GetFrame after 3 StepFrame = %d, want 3", got)
	}

	if fb := n.GetFramebuffer(); len(fb) != 256*240*4 {
		t.Errorf("GetFramebuffer len = %d, want %d", len(fb), 256*240*4)
	}
	if raw := n.GetFramebufferRaw(); len(raw) != 256*240 {
		t.Errorf("GetFramebufferRaw len = %d, want %d", len(raw), 256*240)
	}
	if disp := n.GetDisplayFramebufferRaw(); len(disp) != 256*240 {
		t.Errorf("GetDisplayFramebufferRaw len = %d", len(disp))
	}
	if n.GetInput() == nil {
		t.Error("GetInput returned nil")
	}
}

func TestNESResetVariants(t *testing.T) {
	n := NewNES()
	n.LoadCartridge(testCartridge(t))
	n.Reset()
	n.StepFrame()

	// SoftReset preserves RAM/registers; just ensure it doesn't panic and
	// re-initialises the interrupt pipeline.
	n.SoftReset()
}

func TestNESSaveLoadStateRoundTrip(t *testing.T) {
	n := NewNES()
	n.LoadCartridge(testCartridge(t))
	n.Reset()
	for i := 0; i < 2; i++ {
		n.StepFrame()
	}
	n.Memory.RAM[0x123] = 0xC5 // sentinel that must survive the round-trip

	var buf bytes.Buffer
	if err := n.SaveState(&buf); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	n2 := NewNES()
	n2.LoadCartridge(testCartridge(t))
	n2.Reset()
	if err := n2.LoadState(bytes.NewReader(buf.Bytes())); err != nil {
		t.Fatalf("LoadState: %v", err)
	}

	// Compare a field from every major component so a dropped sub-component
	// (not just the CPU PC) is caught.
	if n2.CPU.PC != n.CPU.PC || n2.CPU.A != n.CPU.A || n2.CPU.SP != n.CPU.SP || n2.CPU.P != n.CPU.P {
		t.Errorf("CPU regs not restored: got PC=%#04x A=%#02x SP=%#02x P=%#02x, want PC=%#04x A=%#02x SP=%#02x P=%#02x",
			n2.CPU.PC, n2.CPU.A, n2.CPU.SP, n2.CPU.P, n.CPU.PC, n.CPU.A, n.CPU.SP, n.CPU.P)
	}
	if n2.PPU.Frame != n.PPU.Frame {
		t.Errorf("restored PPU.Frame = %d, want %d", n2.PPU.Frame, n.PPU.Frame)
	}
	if n2.APU.Cycles != n.APU.Cycles {
		t.Errorf("restored APU.Cycles = %d, want %d", n2.APU.Cycles, n.APU.Cycles)
	}
	if n2.Memory.RAM[0x123] != 0xC5 {
		t.Errorf("restored RAM[0x123] = %#02x, want 0xC5", n2.Memory.RAM[0x123])
	}
}

func TestSaveStateWriteErrors(t *testing.T) {
	n := NewNES()
	n.LoadCartridge(testCartridge(t))
	n.Reset()
	n.StepFrame()

	// A full save establishes the total size; failing at offsets spread across
	// it exercises each component's "return ...: %w" error path in turn.
	var full bytes.Buffer
	if err := n.SaveState(&full); err != nil {
		t.Fatalf("baseline SaveState: %v", err)
	}
	size := full.Len()
	for _, limit := range []int{0, 4, 8, 40, size / 4, size / 2, size - 2} {
		if err := n.SaveState(&failWriter{limit: limit}); err == nil {
			t.Errorf("SaveState with write limit %d should fail", limit)
		}
	}
}

func TestLoadStateTruncatedErrors(t *testing.T) {
	n := NewNES()
	n.LoadCartridge(testCartridge(t))
	n.Reset()
	n.StepFrame()

	var full bytes.Buffer
	if err := n.SaveState(&full); err != nil {
		t.Fatalf("baseline SaveState: %v", err)
	}
	data := full.Bytes()
	// Truncating the saved blob at increasing lengths makes successive
	// component LoadState calls hit EOF, covering their error returns.
	for _, trunc := range []int{2, 6, 10, 50, len(data) / 4, len(data) / 2, len(data) - 1} {
		n2 := NewNES()
		n2.LoadCartridge(testCartridge(t))
		n2.Reset()
		if err := n2.LoadState(bytes.NewReader(data[:trunc])); err == nil {
			t.Errorf("LoadState of %d-byte truncation should fail", trunc)
		}
	}
}

func TestLoadStateRejectsBadMagic(t *testing.T) {
	n := NewNES()
	n.LoadCartridge(testCartridge(t))
	n.Reset()
	if err := n.LoadState(bytes.NewReader([]byte("not a state file at all"))); err == nil {
		t.Error("LoadState should reject a bad magic header")
	}
}

func TestCompanionFile(t *testing.T) {
	cases := []struct{ rom, suffix, want string }{
		{"game.nes", ".sav", "game.sav"},
		{"/path/to/game.nes", ".state1", "/path/to/game.state1"},
		{"noext", ".cht", "noext.cht"},
	}
	for _, tc := range cases {
		if got := CompanionFile(tc.rom, tc.suffix); got != tc.want {
			t.Errorf("CompanionFile(%q,%q) = %q, want %q", tc.rom, tc.suffix, got, tc.want)
		}
	}
}
