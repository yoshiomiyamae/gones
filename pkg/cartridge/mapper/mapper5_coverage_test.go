package mapper

import (
	"bytes"
	"testing"
)

// mmc5Data builds an MMC5 cartridge: 8×8KB PRG (tagged 0xA0+bank at each 8KB
// boundary), 64×1KB CHR, plus PRG/CHR RAM.
func mmc5Data() *CartridgeData {
	prg := make([]uint8, 8*8192)
	for b := 0; b < 8; b++ {
		prg[b*8192] = uint8(0xA0 + b)
	}
	chr := make([]uint8, 64*1024)
	for b := 0; b < 64; b++ {
		chr[b*1024] = uint8(b)
	}
	return &CartridgeData{
		PRGROM: prg,
		CHRROM: chr,
		PRGRAM: make([]uint8, 16384),
		CHRRAM: make([]uint8, 8192),
	}
}

func TestMapper5PRGModes(t *testing.T) {
	m := NewMapper5(mmc5Data())

	// Mode 0: single 32KB bank via $5117.
	m.WritePRG(0x5100, 0x00)      // prgMode 0
	m.WritePRG(0x5117, 0x80) // $5117 = ROM (bit 7) + bank 0
	if got := m.ReadPRG(0x8000); got != 0xA0 {
		t.Errorf("mode0 $8000 = %#02x, want 0xA0", got)
	}

	// Mode 3: four 8KB slots ($5114-$5117 -> prgBanks[1..4]).
	m.WritePRG(0x5100, 0x03)
	m.WritePRG(0x5114, 0x80|0x01) // $8000 -> bank 1
	m.WritePRG(0x5115, 0x80|0x02) // $A000 -> bank 2
	m.WritePRG(0x5116, 0x80|0x03) // $C000 -> bank 3
	m.WritePRG(0x5117, 0x80|0x04) // $E000 -> bank 4
	checks := map[uint16]uint8{0x8000: 0xA1, 0xA000: 0xA2, 0xC000: 0xA3, 0xE000: 0xA4}
	for addr, want := range checks {
		if got := m.ReadPRG(addr); got != want {
			t.Errorf("mode3 %#04x = %#02x, want %#02x", addr, got, want)
		}
	}

	// Modes 1 and 2 just need to resolve without panicking (region math).
	m.WritePRG(0x5100, 0x01)
	_ = m.ReadPRG(0x8000)
	_ = m.ReadPRG(0xC000)
	m.WritePRG(0x5100, 0x02)
	_ = m.ReadPRG(0x8000)
	_ = m.ReadPRG(0xC000)
	_ = m.ReadPRG(0xE000)
}

func TestMapper5PRGRAMAndExRAM(t *testing.T) {
	m := NewMapper5(mmc5Data())

	// PRG RAM is locked by default: the write is dropped, so the byte stays 0.
	m.WritePRG(0x6000, 0x11)
	if got := m.ReadPRG(0x6000); got != 0 {
		t.Errorf("locked PRG RAM write should be dropped, read=%#02x want 0", got)
	}
	// Unlock sequence: $5102=0x02, $5103=0x01.
	m.WritePRG(0x5102, 0x02)
	m.WritePRG(0x5103, 0x01)
	m.WritePRG(0x5113, 0x00) // bank 0 at $6000
	m.WritePRG(0x6000, 0x5A)
	if got := m.ReadPRG(0x6000); got != 0x5A {
		t.Errorf("unlocked PRG RAM = %#02x, want 0x5A", got)
	}

	// ExRAM: mode 2 makes it general-purpose R/W.
	m.WritePRG(0x5104, 0x02)
	m.WritePRG(0x5C00, 0x7E)
	if got := m.ReadPRG(0x5C00); got != 0x7E {
		t.Errorf("ExRAM = %#02x, want 0x7E", got)
	}
	// Mode 3 makes ExRAM read-only: a further write is dropped.
	m.WritePRG(0x5104, 0x03)
	m.WritePRG(0x5C00, 0x00)
	if got := m.ReadPRG(0x5C00); got != 0x7E {
		t.Errorf("ExRAM mode3 should be read-only, got %#02x", got)
	}
}

func TestMapper5Multiplier(t *testing.T) {
	m := NewMapper5(mmc5Data())
	m.WritePRG(0x5205, 0x10) // multA
	m.WritePRG(0x5206, 0x10) // multB; 0x10*0x10 = 0x0100
	if lo := m.ReadPRG(0x5205); lo != 0x00 {
		t.Errorf("mult low = %#02x, want 0x00", lo)
	}
	if hi := m.ReadPRG(0x5206); hi != 0x01 {
		t.Errorf("mult high = %#02x, want 0x01", hi)
	}
}

func TestMapper5CHRModes(t *testing.T) {
	m := NewMapper5(mmc5Data())
	// Program the 'A' and 'B' CHR bank sets.
	for i := uint16(0); i < 8; i++ {
		m.WritePRG(0x5120+i, uint8(i)) // chrA[i]
	}
	for i := uint16(0); i < 4; i++ {
		m.WritePRG(0x5128+i, uint8(i)) // chrB[i]
	}
	// Each CHR mode resolves $0000 and $1000 through different bank math.
	for mode := uint8(0); mode <= 3; mode++ {
		m.WritePRG(0x5101, mode)
		_ = m.ReadCHR(0x0000)
		_ = m.ReadCHR(0x1000)
		_ = m.ReadCHRSprite(0x0000)
	}

	// Verify the resolved bank byte (mmc5Data tags CHR bank b's first byte = b).
	m.WritePRG(0x5101, 3) // 1KB granularity: $1000 -> chrA[(0x1000>>10)&7 = 4] = 4
	if got := m.ReadCHR(0x1000); got != 4 {
		t.Errorf("chrMode3 $1000 = %#02x, want 0x04 (bank 4)", got)
	}
	m.WritePRG(0x5101, 0) // 8KB: bank = chrA[7]*8 = 56
	if got := m.ReadCHR(0x0000); got != 56 {
		t.Errorf("chrMode0 $0000 = %#02x, want 56 (bank 56)", got)
	}

	// 8×16 sprite mode routes BG fetches through the 'B' set.
	m.SetSpriteSize(true)
	_ = m.ReadCHR(0x0000)
	m.SetSpriteSize(false)

	// CHR-RAM fallback path (no CHR ROM).
	ramData := mmc5Data()
	ramData.CHRROM = nil
	mr := NewMapper5(ramData)
	mr.WriteCHR(0x0000, 0x99)
	if got := mr.ReadCHR(0x0000); got != 0x99 {
		t.Errorf("CHR RAM fallback = %#02x, want 0x99", got)
	}
	_ = mr.ReadCHRSprite(0x0000)
}

func TestMapper5IRQAndMisc(t *testing.T) {
	m := NewMapper5(mmc5Data())
	m.WritePRG(0x5105, 0x55) // nametable mapping
	m.WritePRG(0x5106, 0xAA) // fill tile
	m.WritePRG(0x5107, 0x02) // fill attribute
	m.WritePRG(0x5203, 16)   // IRQ target scanline
	m.WritePRG(0x5204, 0x80) // IRQ enable

	// Drive scanlines past the target; the scanline-match latch must fire and,
	// with the enable bit set, surface through IsIRQPending.
	for s := 0; s < 20; s++ {
		m.NotifyScanline(s, true)
		m.Step()
	}
	if !m.IsIRQPending() {
		t.Error("MMC5 IRQ should be pending after the scanline counter reaches the target")
	}
	// Disabling rendering drops the in-frame flag (rendering-off branch).
	m.NotifyScanline(5, false)
	_ = m.GetMirroringMode()

	m.DecodesExpansion()
	m.NotifyA12(0x1000, true)
	m.ClearIRQ()
	m.IRQCapable()

	// State round-trip.
	var buf bytes.Buffer
	if err := m.SaveState(&buf); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	if err := m.LoadState(bytes.NewReader(buf.Bytes())); err != nil {
		t.Fatalf("LoadState: %v", err)
	}
}
