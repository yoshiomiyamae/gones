package mapper

import (
	"bytes"
	"testing"
)

// TestMapper70 covers the Bandai mapper's bank register layout:
// .BBB CCCC where BBB is the 16KB PRG bank and CCCC is the 8KB CHR bank.
func TestMapper70(t *testing.T) {
	// 128KB PRG: each bank tagged with its index in byte 0.
	prgROM := make([]uint8, 128*1024)
	for i := 0; i < 8; i++ {
		prgROM[i*16384] = uint8(i + 1)
	}
	// 128KB CHR: tag each 8KB bank's byte 0 too.
	chrROM := make([]uint8, 128*1024)
	for i := 0; i < 16; i++ {
		chrROM[i*8192] = uint8(0x10 + i)
	}

	t.Run("Initial_State", func(t *testing.T) {
		m := NewMapper70(&CartridgeData{PRGROM: prgROM, CHRROM: chrROM})
		if got := m.ReadPRG(0x8000); got != 0x01 {
			t.Errorf("initial $8000: want bank 0 ($01), got $%02X", got)
		}
		if got := m.ReadPRG(0xC000); got != 0x08 {
			t.Errorf("initial $C000: want last bank ($08), got $%02X", got)
		}
		if got := m.ReadCHR(0x0000); got != 0x10 {
			t.Errorf("initial CHR bank 0 byte 0: want $10, got $%02X", got)
		}
	})

	t.Run("PRG_And_CHR_Switch_Together", func(t *testing.T) {
		m := NewMapper70(&CartridgeData{PRGROM: prgROM, CHRROM: chrROM})
		// .BBB CCCC = 0b0101 0011 → PRG bank 5, CHR bank 3.
		m.WritePRG(0x8000, 0x53)
		if got := m.ReadPRG(0x8000); got != 0x06 {
			t.Errorf("$8000 after PRG=5: want $06, got $%02X", got)
		}
		if got := m.ReadPRG(0xC000); got != 0x08 {
			t.Errorf("$C000 should stay fixed at last bank ($08), got $%02X", got)
		}
		if got := m.ReadCHR(0x0000); got != 0x13 {
			t.Errorf("CHR bank 3 byte 0: want $13, got $%02X", got)
		}
	})

	t.Run("Bit7_Ignored", func(t *testing.T) {
		m := NewMapper70(&CartridgeData{PRGROM: prgROM, CHRROM: chrROM})
		// Bit 7 must not affect bank selection on mapper 70.
		m.WritePRG(0x8000, 0x80|0x12) // PRG bank 1, CHR bank 2
		if got := m.ReadPRG(0x8000); got != 0x02 {
			t.Errorf("PRG bank with bit 7 set: want $02, got $%02X", got)
		}
		if got := m.ReadCHR(0x0000); got != 0x12 {
			t.Errorf("CHR bank with bit 7 set: want $12, got $%02X", got)
		}
	})

	t.Run("Writes_Anywhere_In_8000_FFFF_Hit_Register", func(t *testing.T) {
		m := NewMapper70(&CartridgeData{PRGROM: prgROM, CHRROM: chrROM})
		for _, addr := range []uint16{0x8000, 0xA000, 0xC000, 0xE000, 0xFFFF} {
			m.WritePRG(addr, 0x20) // PRG bank 2
			if got := m.ReadPRG(0x8000); got != 0x03 {
				t.Errorf("write to $%04X: want bank 2 visible at $8000 ($03), got $%02X", addr, got)
			}
		}
	})

	t.Run("Save_Load_State", func(t *testing.T) {
		m := NewMapper70(&CartridgeData{PRGROM: prgROM, CHRROM: chrROM})
		m.WritePRG(0x8000, 0x47) // PRG 4, CHR 7
		var buf bytes.Buffer
		if err := m.SaveState(&buf); err != nil {
			t.Fatalf("SaveState: %v", err)
		}
		m2 := NewMapper70(&CartridgeData{PRGROM: prgROM, CHRROM: chrROM})
		if err := m2.LoadState(&buf); err != nil {
			t.Fatalf("LoadState: %v", err)
		}
		if got := m2.ReadPRG(0x8000); got != 0x05 {
			t.Errorf("after round-trip: $8000 want bank 4 ($05), got $%02X", got)
		}
		if got := m2.ReadCHR(0x0000); got != 0x17 {
			t.Errorf("after round-trip: CHR bank 7 byte 0 want $17, got $%02X", got)
		}
	})
}
