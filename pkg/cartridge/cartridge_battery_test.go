package cartridge

import (
	"bytes"
	"testing"
)

func newBatteryCart(t *testing.T) *Cartridge {
	t.Helper()
	c := &Cartridge{}
	c.Header.Flags6 = 0x02 // battery flag
	c.PRGRAM = make([]uint8, 32768)
	for i := range c.PRGRAM {
		c.PRGRAM[i] = uint8(i * 7) // deterministic non-zero pattern
	}
	return c
}

func TestHasBattery(t *testing.T) {
	c := &Cartridge{}
	if c.HasBattery() {
		t.Error("expected HasBattery=false when flag bit 1 is clear")
	}
	c.Header.Flags6 = 0x02
	if !c.HasBattery() {
		t.Error("expected HasBattery=true when flag bit 1 is set")
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	c := newBatteryCart(t)
	var buf bytes.Buffer
	if err := c.SaveRAM(&buf); err != nil {
		t.Fatalf("SaveRAM: %v", err)
	}
	if got, want := buf.Len(), len(c.PRGRAM); got != want {
		t.Fatalf("saved size = %d, want %d", got, want)
	}

	loaded := &Cartridge{PRGRAM: make([]uint8, 32768)}
	if err := loaded.LoadRAM(&buf); err != nil {
		t.Fatalf("LoadRAM: %v", err)
	}
	if !bytes.Equal(loaded.PRGRAM, c.PRGRAM) {
		t.Error("round-tripped PRGRAM differs from source")
	}
}

func TestLoadRAMShortFile(t *testing.T) {
	c := &Cartridge{PRGRAM: make([]uint8, 32768)}
	// File smaller than PRGRAM — should partially load without error.
	short := bytes.Repeat([]byte{0xAB}, 100)
	if err := c.LoadRAM(bytes.NewReader(short)); err != nil {
		t.Fatalf("LoadRAM on short file: %v", err)
	}
	for i := 0; i < 100; i++ {
		if c.PRGRAM[i] != 0xAB {
			t.Fatalf("byte %d = %#x, want 0xAB", i, c.PRGRAM[i])
		}
	}
}

func TestSaveLoadEmptyRAM(t *testing.T) {
	c := &Cartridge{} // no PRGRAM
	var buf bytes.Buffer
	if err := c.SaveRAM(&buf); err != nil {
		t.Errorf("SaveRAM with no RAM: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("SaveRAM with no RAM wrote %d bytes", buf.Len())
	}
	if err := c.LoadRAM(bytes.NewReader([]byte{1, 2, 3})); err != nil {
		t.Errorf("LoadRAM with no RAM: %v", err)
	}
}
