package cartridge

import (
	"bytes"
	"testing"

	"github.com/yoshiomiyamaegones/pkg/cartridge/mapper"
)

// buildINES synthesizes an iNES image for the given mapper with prg16k×16KB PRG
// and chr8k×8KB CHR ROM (CHR RAM is used when chr8k==0).
func buildINES(mapper uint8, prg16k, chr8k int) []byte {
	var buf bytes.Buffer
	buf.WriteString("NES\x1A")
	buf.WriteByte(byte(prg16k))
	buf.WriteByte(byte(chr8k))
	buf.WriteByte((mapper & 0x0F) << 4) // flags6: mapper low nibble
	buf.WriteByte(mapper & 0xF0)        // flags7: mapper high nibble
	buf.Write(make([]byte, 8))          // flags8-10 + padding
	buf.Write(make([]byte, prg16k*16384))
	buf.Write(make([]byte, chr8k*8192))
	return buf.Bytes()
}

func TestCartridgeWrappers(t *testing.T) {
	// MMC3 (mapper 4): HasIRQ true, A12/scanline/IRQ wrappers do real work.
	cart, err := LoadFromReader(bytes.NewReader(buildINES(4, 2, 1)))
	if err != nil {
		t.Fatalf("LoadFromReader: %v", err)
	}

	if !cart.HasIRQ() {
		t.Error("MMC3 cart should report HasIRQ")
	}
	if cart.HasExpansion() {
		t.Error("MMC3 cart should not decode expansion space")
	}

	// PRG/CHR pass-throughs.
	_ = cart.ReadPRG(0x8000)
	cart.WritePRG(0x8000, 0x00) // MMC3 bank-select register
	_ = cart.ReadCHR(0x0000)
	_ = cart.ReadCHRSprite(0x0000) // falls back to ReadCHR (no sprite reader)
	cart.WriteCHR(0x0000, 0x00)    // CHR ROM: ignored

	// IRQ / timing wrappers.
	cart.Step()
	cart.NotifyA12(0x1000, true)
	cart.NotifyScanline(10, true)
	cart.TickCPU(3)
	_ = cart.IsIRQPending()
	cart.ClearIRQ()
	if cart.AudioSample() != 0 {
		t.Error("non-expansion cart should produce 0 audio sample")
	}
	cart.SetSpriteSize(true) // no hinter -> no-op
	_ = cart.GetMirroring()

	// State round-trip must actually restore mapper registers: set MMC3
	// bank-select to a sentinel, save, clobber it, load, confirm it came back.
	cart.WritePRG(0x8000, 0x05)
	var sb bytes.Buffer
	if err := cart.SaveState(&sb); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	cart.WritePRG(0x8000, 0x02) // clobber before reload
	if err := cart.LoadState(bytes.NewReader(sb.Bytes())); err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	m4, ok := cart.Mapper.(*mapper.Mapper4)
	if !ok {
		t.Fatalf("expected *mapper.Mapper4, got %T", cart.Mapper)
	}
	if got := m4.GetBankSelect(); got != 0x05 {
		t.Errorf("bank-select after round-trip = %#02x, want 0x05", got)
	}
}

func TestCartridgeNonIRQMapper(t *testing.T) {
	// NROM (mapper 0): HasIRQ false, IsIRQPending false.
	cart, err := LoadFromReader(bytes.NewReader(buildINES(0, 2, 1)))
	if err != nil {
		t.Fatalf("LoadFromReader: %v", err)
	}
	if cart.HasIRQ() {
		t.Error("NROM should report no IRQ")
	}
	if cart.IsIRQPending() {
		t.Error("NROM should never have a pending IRQ")
	}
}

func TestLoadFromReaderRejectsBadMagic(t *testing.T) {
	if _, err := LoadFromReader(bytes.NewReader([]byte("not-an-ines-image"))); err == nil {
		t.Error("LoadFromReader should reject a bad magic header")
	}
}

// buildINESFlags is buildINES with explicit flags6 (mapper low nibble OR'd with
// the trainer/battery/four-screen bits) and an optional 512-byte trainer.
func buildINESFlags(mapper uint8, prg16k, chr8k int, flag6Extra uint8, trainer bool) []byte {
	var buf bytes.Buffer
	buf.WriteString("NES\x1A")
	buf.WriteByte(byte(prg16k))
	buf.WriteByte(byte(chr8k))
	buf.WriteByte((mapper&0x0F)<<4 | flag6Extra)
	buf.WriteByte(mapper & 0xF0)
	buf.Write(make([]byte, 8))
	if trainer {
		buf.Write(make([]byte, 512))
	}
	buf.Write(make([]byte, prg16k*16384))
	buf.Write(make([]byte, chr8k*8192))
	return buf.Bytes()
}

func TestLoadFromReaderHeaderVariants(t *testing.T) {
	// Trainer present (flags6 bit 2): the 512-byte trainer is skipped.
	if _, err := LoadFromReader(bytes.NewReader(buildINESFlags(0, 2, 1, 0x04, true))); err != nil {
		t.Errorf("trainer ROM: %v", err)
	}

	// Battery-backed (flags6 bit 1): allocates 32KB PRG RAM.
	cart, err := LoadFromReader(bytes.NewReader(buildINESFlags(1, 2, 1, 0x02, false)))
	if err != nil {
		t.Fatalf("battery ROM: %v", err)
	}
	if len(cart.PRGRAM) != 32768 {
		t.Errorf("battery PRG RAM = %d, want 32768", len(cart.PRGRAM))
	}

	// Four-screen mirroring (flags6 bit 3).
	if _, err := LoadFromReader(bytes.NewReader(buildINESFlags(0, 2, 1, 0x08, false))); err != nil {
		t.Errorf("four-screen ROM: %v", err)
	}

	// Vertical mirroring (flags6 bit 0).
	if _, err := LoadFromReader(bytes.NewReader(buildINESFlags(0, 2, 1, 0x01, false))); err != nil {
		t.Errorf("vertical-mirror ROM: %v", err)
	}

	// MMC3 with no CHR ROM allocates 32KB CHR RAM.
	cart, err = LoadFromReader(bytes.NewReader(buildINES(4, 2, 0)))
	if err != nil {
		t.Fatalf("MMC3 CHR-RAM ROM: %v", err)
	}
	if len(cart.CHRRAM) != 32768 {
		t.Errorf("MMC3 CHR RAM = %d, want 32768", len(cart.CHRRAM))
	}
}
