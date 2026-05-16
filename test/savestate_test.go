package test

import (
	"bytes"
	"testing"

	"github.com/yoshiomiyamaegones/pkg/cartridge"
	"github.com/yoshiomiyamaegones/pkg/cartridge/mapper"
	"github.com/yoshiomiyamaegones/pkg/nes"
)

// makeMMC3NES builds a minimal MMC3-cartridge NES with a 32KB PRG ROM that
// loops at the reset vector. Used to exercise SaveState/LoadState on a
// representative mapper.
func makeMMC3NES(t *testing.T) *nes.NES {
	t.Helper()
	prgROM := make([]uint8, 32*1024)
	chrRAM := make([]uint8, 32*1024)
	// JMP $E000 (so the CPU loops forever at the reset vector location).
	prgROM[0x6000] = 0x4C
	prgROM[0x6001] = 0x00
	prgROM[0x6002] = 0xE0
	prgROM[0x7FFC] = 0x00
	prgROM[0x7FFD] = 0xE0

	cartData := &mapper.CartridgeData{PRGROM: prgROM, CHRRAM: chrRAM, PRGRAM: make([]uint8, 8*1024)}
	cart := &cartridge.Cartridge{
		PRGROM: prgROM,
		CHRRAM: chrRAM,
		PRGRAM: cartData.PRGRAM,
		Mapper: mapper.NewMapper4(cartData),
	}

	n := nes.NewNES()
	n.LoadCartridge(cart)
	n.Reset()
	return n
}

// runFrames advances the emulator by n frames. Stateful side effects
// (PPU/CPU/APU/cartridge RAM updates) accumulate naturally.
func runFrames(n *nes.NES, frames int) {
	for i := 0; i < frames; i++ {
		n.StepFrame()
	}
}

// TestSaveStateRoundTrip verifies that Save → Load produces byte-identical
// state by saving twice (once before load, once after) and comparing.
func TestSaveStateRoundTrip(t *testing.T) {
	n := makeMMC3NES(t)
	runFrames(n, 20)

	var snapshotA bytes.Buffer
	if err := n.SaveState(&snapshotA); err != nil {
		t.Fatalf("first SaveState: %v", err)
	}

	// Run more frames, then load the earlier snapshot.
	runFrames(n, 10)
	if err := n.LoadState(bytes.NewReader(snapshotA.Bytes())); err != nil {
		t.Fatalf("LoadState: %v", err)
	}

	// Save again; the post-load snapshot must equal the pre-load snapshot
	// byte-for-byte (modulo APU gob which can have minor ordering
	// differences — compare the binary-encoded prefix only).
	var snapshotB bytes.Buffer
	if err := n.SaveState(&snapshotB); err != nil {
		t.Fatalf("second SaveState: %v", err)
	}

	if !bytes.Equal(snapshotA.Bytes(), snapshotB.Bytes()) {
		t.Errorf("snapshot mismatch: first %d bytes vs second %d bytes",
			snapshotA.Len(), snapshotB.Len())
	}
}

// TestLoadStateRejectsBadMagic confirms LoadState rejects garbage input.
func TestLoadStateRejectsBadMagic(t *testing.T) {
	n := makeMMC3NES(t)
	garbage := bytes.NewReader([]byte{0xDE, 0xAD, 0xBE, 0xEF, 0, 0, 0, 1})
	if err := n.LoadState(garbage); err == nil {
		t.Error("expected error for bad magic, got nil")
	}
}

// TestLoadStateRejectsWrongVersion confirms version-mismatch is rejected.
func TestLoadStateRejectsWrongVersion(t *testing.T) {
	n := makeMMC3NES(t)
	// Magic OK, version intentionally wrong.
	var buf bytes.Buffer
	buf.Write([]byte{0x54, 0x53, 0x4E, 0x47}) // "GNST" little-endian
	buf.Write([]byte{0xFF, 0xFF, 0xFF, 0xFF}) // bogus version
	if err := n.LoadState(&buf); err == nil {
		t.Error("expected error for version mismatch, got nil")
	}
}
