package memory

import (
	"bytes"
	"testing"
)

// TestReadNilComponents covers the open-bus fallbacks taken when no PPU/APU/
// Input/Cartridge is attached — paths the integration tests never hit because
// they always wire up a full system.
func TestReadNilComponents(t *testing.T) {
	m := New()

	// RAM is always present and latches the bus.
	m.RAM[0x10] = 0x42
	if got := m.Read(0x0010); got != 0x42 {
		t.Errorf("RAM read = %#02x, want 0x42", got)
	}
	if got := m.Read(0x0810); got != 0x42 { // mirror of $0010
		t.Errorf("RAM mirror read = %#02x, want 0x42", got)
	}

	// $2000-$3FFF with no PPU returns the latched bus value, not a panic.
	_ = m.Read(0x2000)
	// $4015 with no APU, $4016 with no Input, write-only $4000-$4014, and the
	// unallocated $4018-$401F window all fall through to the open-bus latch.
	_ = m.Read(0x4015)
	_ = m.Read(0x4016)
	_ = m.Read(0x4000)
	_ = m.Read(0x401F)
	// $4020-$5FFF with no expansion cartridge.
	_ = m.Read(0x5000)
}

func TestHighMemFallback(t *testing.T) {
	m := New()
	// With no cartridge, $6000-$FFFF is backed by HighMem.
	m.Write(0x6000, 0xAB)
	if got := m.Read(0x6000); got != 0xAB {
		t.Errorf("HighMem $6000 = %#02x, want 0xAB", got)
	}
	m.Write(0xC123, 0xCD)
	if got := m.Read(0xC123); got != 0xCD {
		t.Errorf("HighMem $C123 = %#02x, want 0xCD", got)
	}
}

func TestWriteNilComponents(t *testing.T) {
	m := New()
	// These must be safe no-ops (or bus latches) with no devices attached.
	m.Write(0x2000, 0x01) // nil PPU
	m.Write(0x4000, 0x02) // nil APU
	m.Write(0x4016, 0x03) // nil Input
	m.Write(0x5000, 0x04) // no expansion cartridge

	// OAM DMA at $4014 charges the stall cost even with no PPU to receive it.
	if stall := m.Write(0x4014, 0x00); stall != oamDMAStallCycles {
		t.Errorf("OAM DMA stall = %d, want %d", stall, oamDMAStallCycles)
	}
}

func TestSaveLoadStateRAM(t *testing.T) {
	m := New()
	m.RAM[5] = 0x99
	var buf bytes.Buffer
	if err := m.SaveState(&buf); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	m2 := New()
	if err := m2.LoadState(&buf); err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if m2.RAM[5] != 0x99 {
		t.Errorf("restored RAM[5] = %#02x, want 0x99", m2.RAM[5])
	}
}
