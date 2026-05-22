package mapper

import (
	"bytes"
	"io"
	"testing"
)

// makeData builds a CartridgeData with PRG/CHR ROM sized to the given bank
// counts, each 16KB PRG bank / 4KB CHR bank tagged with its index in its first
// byte so tests can verify which bank a read resolved to. PRG/CHR RAM are
// always allocated (8KB) so RAM-path branches are exercisable.
func makeData(prg16k, chr4k int) *CartridgeData {
	prg := make([]uint8, prg16k*16384)
	for b := 0; b < prg16k; b++ {
		prg[b*16384] = uint8(0xA0 + b)
	}
	chr := make([]uint8, chr4k*4096)
	for b := 0; b < chr4k; b++ {
		chr[b*4096] = uint8(0xC0 + b)
	}
	return &CartridgeData{
		PRGROM: prg,
		CHRROM: chr,
		PRGRAM: make([]uint8, 8192),
		CHRRAM: make([]uint8, 8192),
	}
}

// --- NewMapper factory ---

func TestNewMapperFactory(t *testing.T) {
	data := makeData(2, 2)
	for _, num := range []uint8{0, 1, 2, 3, 4, 5, 10, 69, 70} {
		m, err := NewMapper(num, data)
		if err != nil || m == nil {
			t.Errorf("NewMapper(%d): m=%v err=%v", num, m, err)
		}
	}
	if _, err := NewMapper(255, data); err == nil {
		t.Error("NewMapper(255) should return an unsupported-mapper error")
	}
}

// --- MMC4 (mapper10) ---

func mmc4Data() *CartridgeData {
	// 4 PRG banks (16KB), 4 CHR banks (4KB), each tagged at offset 0.
	d := makeData(4, 4)
	return d
}

func TestMapper10PRGBanking(t *testing.T) {
	m := NewMapper10(mmc4Data())

	// Default prgBank 0 at $8000; last bank fixed at $C000.
	if got := m.ReadPRG(0x8000); got != 0xA0 {
		t.Errorf("$8000 default = %#02x, want 0xA0 (bank 0)", got)
	}
	if got := m.ReadPRG(0xC000); got != 0xA3 {
		t.Errorf("$C000 fixed = %#02x, want 0xA3 (last bank)", got)
	}

	// Switch the $8000 bank to 2 via $A000.
	m.WritePRG(0xA000, 0x02)
	if got := m.ReadPRG(0x8000); got != 0xA2 {
		t.Errorf("$8000 after bank=2 = %#02x, want 0xA2", got)
	}

	// PRG RAM round-trip at $6000.
	m.WritePRG(0x6000, 0x77)
	if got := m.ReadPRG(0x6000); got != 0x77 {
		t.Errorf("PRG RAM = %#02x, want 0x77", got)
	}
}

func TestMapper10CHRLatch(t *testing.T) {
	m := NewMapper10(mmc4Data())
	// Assign distinct CHR banks to each latch register.
	m.WritePRG(0xB000, 0x00) // chrBank0FD = 0
	m.WritePRG(0xC000, 0x01) // chrBank0FE = 1
	m.WritePRG(0xD000, 0x02) // chrBank1FD = 2
	m.WritePRG(0xE000, 0x03) // chrBank1FE = 3

	// latch0 defaults to $FE -> bank0FE (1) for $0000-$0FFF.
	if got := m.ReadCHR(0x0000); got != 0xC1 {
		t.Errorf("$0000 (latch0=FE) = %#02x, want 0xC1 (bank 1)", got)
	}
	// Reading the $0FD8-$0FDF trigger flips latch0 to $FD for *subsequent* reads.
	m.ReadCHR(0x0FD8)
	if got := m.ReadCHR(0x0000); got != 0xC0 {
		t.Errorf("$0000 (latch0=FD) = %#02x, want 0xC0 (bank 0)", got)
	}
	// $0FE8 trigger flips latch0 back to $FE.
	m.ReadCHR(0x0FE8)
	if got := m.ReadCHR(0x0000); got != 0xC1 {
		t.Errorf("$0000 (latch0=FE again) = %#02x, want 0xC1", got)
	}

	// Latch1 controls $1000-$1FFF; default $FE -> bank1FE (3).
	if got := m.ReadCHR(0x1000); got != 0xC3 {
		t.Errorf("$1000 (latch1=FE) = %#02x, want 0xC3 (bank 3)", got)
	}
	m.ReadCHR(0x1FD8)
	if got := m.ReadCHR(0x1000); got != 0xC2 {
		t.Errorf("$1000 (latch1=FD) = %#02x, want 0xC2 (bank 2)", got)
	}
	m.ReadCHR(0x1FE8)
	if got := m.ReadCHR(0x1000); got != 0xC3 {
		t.Errorf("$1000 (latch1=FE again) = %#02x, want 0xC3", got)
	}
}

func TestMapper10MiscAndState(t *testing.T) {
	m := NewMapper10(mmc4Data())

	// Mirroring: bit 0 = 0 -> PPU vertical(1); = 1 -> PPU horizontal(0).
	m.WritePRG(0xF000, 0x00)
	if m.GetMirroringMode() != 1 {
		t.Errorf("mirroring 0 -> %d, want 1 (vertical)", m.GetMirroringMode())
	}
	m.WritePRG(0xF000, 0x01)
	if m.GetMirroringMode() != 0 {
		t.Errorf("mirroring 1 -> %d, want 0 (horizontal)", m.GetMirroringMode())
	}

	// No IRQ source.
	m.Step()
	m.ClearIRQ()
	if m.IsIRQPending() {
		t.Error("MMC4 should never have a pending IRQ")
	}

	// CHR RAM write path (CHRROM present means chrFetch uses ROM, but WriteCHR
	// still targets CHRRAM when allocated).
	m.WriteCHR(0x0010, 0xEE)
	if m.cartridge.CHRRAM[0x0010] != 0xEE {
		t.Errorf("WriteCHR to CHR RAM failed: %#02x", m.cartridge.CHRRAM[0x0010])
	}

	// State round-trip.
	m.WritePRG(0xA000, 0x03)
	m.ReadCHR(0x0FD8) // latch0 = FD
	var buf bytes.Buffer
	if err := m.SaveState(&buf); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	m2 := NewMapper10(mmc4Data())
	if err := m2.LoadState(&buf); err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if m2.prgBank != 0x03 || m2.latch0 != 0xFD {
		t.Errorf("restored state mismatch: prgBank=%#02x latch0=%#02x", m2.prgBank, m2.latch0)
	}
}

// --- MMC1 (mapper1) ---

// mmc1Serial writes a 5-bit value to an MMC1 register via the serial port.
func mmc1Serial(m *Mapper1, addr uint16, val uint8) {
	for i := 0; i < 5; i++ {
		m.WritePRG(addr, (val>>uint(i))&1)
	}
}

func TestMapper1PRGModes(t *testing.T) {
	m := NewMapper1(makeData(4, 2)) // 4×16KB PRG

	// Default control 0x0C -> prgMode 3 (last bank fixed at $C000).
	mmc1Serial(m, 0xE000, 0x01) // prgBank = 1
	if got := m.ReadPRG(0x8000); got != 0xA1 {
		t.Errorf("mode3 $8000 (bank1) = %#02x, want 0xA1", got)
	}
	if got := m.ReadPRG(0xC000); got != 0xA3 {
		t.Errorf("mode3 $C000 (last) = %#02x, want 0xA3", got)
	}

	// prgMode 2: first bank fixed at $8000, switchable at $C000.
	mmc1Serial(m, 0x8000, 0x08) // control: prgMode=(0x08>>2)&3 = 2
	mmc1Serial(m, 0xE000, 0x02) // prgBank = 2
	if got := m.ReadPRG(0x8000); got != 0xA0 {
		t.Errorf("mode2 $8000 (fixed first) = %#02x, want 0xA0", got)
	}
	if got := m.ReadPRG(0xC000); got != 0xA2 {
		t.Errorf("mode2 $C000 (bank2) = %#02x, want 0xA2", got)
	}

	// prgMode 0 (32KB): bank = prgBank>>1.
	mmc1Serial(m, 0x8000, 0x00) // control prgMode=0
	mmc1Serial(m, 0xE000, 0x02) // prgBank=2 -> 32KB bank 1 -> offset 0x8000 -> tag 0xA2
	if got := m.ReadPRG(0x8000); got != 0xA2 {
		t.Errorf("mode0 $8000 (32KB bank1) = %#02x, want 0xA2", got)
	}
}

func TestMapper1Mirroring(t *testing.T) {
	m := NewMapper1(makeData(2, 2))
	wantByCtrl := map[uint8]uint8{0: 2, 1: 3, 2: 1, 3: 0}
	for ctrl, want := range wantByCtrl {
		mmc1Serial(m, 0x8000, ctrl) // control low 2 bits = mirroring
		if got := m.GetMirroringMode(); got != want {
			t.Errorf("mirroring ctrl=%d -> %d, want %d", ctrl, got, want)
		}
	}
}

func TestMapper1CHRAndRAM(t *testing.T) {
	m := NewMapper1(makeData(2, 4)) // 4×4KB CHR -> 0xC0..0xC3 tags

	// 8KB CHR mode (chrMode 0, default): bank = chrBank0>>1.
	mmc1Serial(m, 0xA000, 0x02) // chrBank0 = 2 -> 8KB bank 1 -> CHR offset 0x2000 -> tag 0xC2
	if got := m.ReadCHR(0x0000); got != 0xC2 {
		t.Errorf("8KB CHR = %#02x, want 0xC2", got)
	}

	// 4KB CHR mode (chrMode 1): independent 4KB banks.
	mmc1Serial(m, 0x8000, 0x10) // control bit4 -> chrMode 1
	mmc1Serial(m, 0xA000, 0x01) // chrBank0 = 1 -> $0000 tag 0xC1
	mmc1Serial(m, 0xC000, 0x03) // chrBank1 = 3 -> $1000 tag 0xC3
	if got := m.ReadCHR(0x0000); got != 0xC1 {
		t.Errorf("4KB CHR bank0 = %#02x, want 0xC1", got)
	}
	if got := m.ReadCHR(0x1000); got != 0xC3 {
		t.Errorf("4KB CHR bank1 = %#02x, want 0xC3", got)
	}

	// PRG RAM enabled (prgBank bit4 = 0): round-trip at $6000.
	m.WritePRG(0x6000, 0x5A)
	if got := m.ReadPRG(0x6000); got != 0x5A {
		t.Errorf("PRG RAM = %#02x, want 0x5A", got)
	}
	// Disable PRG RAM via prgBank bit 4: reads return 0, writes ignored.
	mmc1Serial(m, 0xE000, 0x10)
	m.WritePRG(0x6000, 0x99)
	if got := m.ReadPRG(0x6000); got != 0 {
		t.Errorf("disabled PRG RAM read = %#02x, want 0", got)
	}
}

func TestMapper1ResetAndState(t *testing.T) {
	m := NewMapper1(makeData(2, 2))

	// A write with bit 7 set resets the shift register and forces prgMode 3.
	m.WritePRG(0x8000, 0x01) // partial serial write (1 bit)
	m.WritePRG(0x8000, 0x80) // reset
	if m.shiftCount != 0 || m.prgMode != 3 {
		t.Errorf("after reset: shiftCount=%d prgMode=%d, want 0/3", m.shiftCount, m.prgMode)
	}

	m.Step()
	m.ClearIRQ()
	if m.IsIRQPending() {
		t.Error("MMC1 has no IRQ")
	}
	m.WriteCHR(0x0001, 0x42) // CHR RAM write (no CHR ROM banking conflict here)

	mmc1Serial(m, 0xE000, 0x01)
	var buf bytes.Buffer
	if err := m.SaveState(&buf); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	m2 := NewMapper1(makeData(2, 2))
	if err := m2.LoadState(&buf); err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if m2.prgBank != m.prgBank || m2.control != m.control {
		t.Errorf("state mismatch after load")
	}
}

// --- trivial no-ops, debug helpers, and state round-trips for the rest ---

func TestMapperNoOpsAndState(t *testing.T) {
	data := makeData(8, 8)

	// mapper0 (NROM): Step / ClearIRQ are no-ops; IsIRQPending false.
	m0 := NewMapper0(data)
	m0.Step()
	m0.ClearIRQ()
	if m0.IsIRQPending() {
		t.Error("NROM has no IRQ")
	}

	// mapper2 (UxROM): Step/IRQ no-ops, GetCurrentPRGBank, state round-trip.
	m2 := NewMapper2(data)
	m2.WritePRG(0x8000, 0x03)
	_ = m2.GetCurrentPRGBank()
	if m2.IsIRQPending() {
		t.Error("UxROM should never have a pending IRQ")
	}
	m2.Step()
	m2.ClearIRQ()
	roundTrip(t, "mapper2", m2.SaveState, m2.LoadState)

	// mapper3 (CNROM): Step/ClearIRQ no-ops, state round-trip.
	m3 := NewMapper3(data)
	m3.Step()
	m3.ClearIRQ()
	_ = m3.IsIRQPending()
	roundTrip(t, "mapper3", m3.SaveState, m3.LoadState)

	// mapper70: WriteCHR / Step / ClearIRQ.
	m70 := NewMapper70(data)
	m70.WriteCHR(0x0000, 0x11)
	m70.Step()
	m70.ClearIRQ()
	_ = m70.IsIRQPending()

	// mapper69 (FME-7): WriteCHR / Step / ClearIRQ / IRQCapable / state.
	m69 := NewMapper69(data)
	m69.WriteCHR(0x0000, 0x22)
	m69.Step()
	m69.ClearIRQ()
	m69.IRQCapable()
	_ = m69.IsIRQPending()
	roundTrip(t, "mapper69", m69.SaveState, m69.LoadState)

	// mapper5 (MMC5): trivial members + state round-trip.
	m5 := NewMapper5(data)
	m5.Step()
	m5.ClearIRQ()
	m5.IRQCapable()
	m5.DecodesExpansion()
	m5.NotifyA12(0x1000, true)
	m5.WriteCHR(0x0000, 0x33)
	m5.WritePRG(0x6000, 0x44) // exercises prgRAMUnlocked on the RAM path
	_ = m5.ReadPRG(0x6000)
	_ = m5.IsIRQPending()
	roundTrip(t, "mapper5", m5.SaveState, m5.LoadState)

	// mapper4 (MMC3): debug helpers + IRQCapable + ClearIRQ.
	m4 := NewMapper4(data)
	_ = m4.GetBankSelect()
	_ = m4.GetCurrentPRGBanks()
	_ = m4.GetDebugInfo()
	m4.IRQCapable()
	m4.ClearIRQ()
	// DumpCHRRAM / TriggerDump only act when CHR RAM is present.
	ramCart := makeData(8, 0)
	ramCart.CHRROM = nil
	ramCart.CHRRAM = make([]uint8, 32768)
	m4ram := NewMapper4(ramCart)
	m4ram.DumpCHRRAM()
	m4ram.TriggerDump()
}

// roundTrip exercises a mapper's SaveState then LoadState through a buffer.
func roundTrip(t *testing.T, name string, save func(io.Writer) error, load func(io.Reader) error) {
	t.Helper()
	var buf bytes.Buffer
	if err := save(&buf); err != nil {
		t.Fatalf("%s SaveState: %v", name, err)
	}
	if err := load(&buf); err != nil {
		t.Fatalf("%s LoadState: %v", name, err)
	}
}
