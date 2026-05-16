package input

import "testing"

// TestSerialReadSequence checks the standard NES polling pattern:
// strobe high → strobe low → 8 reads emit A, B, Select, Start, U, D, L, R.
func TestSerialReadSequence(t *testing.T) {
	c := New()
	c.SetButton(0, 1, true) // B
	c.SetButton(0, 2, true) // Select

	c.Write(1) // strobe high
	c.Write(0) // strobe low

	want := []uint8{0, 1, 1, 0, 0, 0, 0, 0}
	for i, w := range want {
		got := c.Read() & 1
		if got != w {
			t.Errorf("read %d: got %d want %d (buttons=$%02X)", i, got, w, c.buttons)
		}
	}
	if c.Read() != 1 {
		t.Errorf("read 9 after exhaustion: want 1")
	}
}

// TestStrobeHighLocks verifies that while strobe is high, reads return the
// current A-button bit and the shift register does not advance.
func TestStrobeHighLocks(t *testing.T) {
	c := New()
	c.SetButton(0, 0, true) // A
	c.Write(1)              // strobe high (continuous sample)

	for i := 0; i < 4; i++ {
		if got := c.Read() & 1; got != 1 {
			t.Errorf("strobe-high read %d: got %d want 1", i, got)
		}
	}

	// Now drop strobe: subsequent reads should emit A, then 0, 0, 0...
	c.Write(0)
	if got := c.Read() & 1; got != 1 {
		t.Errorf("first read after strobe-low: got %d want 1 (A)", got)
	}
	for i := 0; i < 7; i++ {
		if got := c.Read() & 1; got != 0 {
			t.Errorf("read %d after A: got %d want 0", i+2, got)
		}
	}
}
