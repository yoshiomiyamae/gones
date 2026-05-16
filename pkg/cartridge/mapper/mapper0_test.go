package mapper

import (
	"testing"
)

// TestMapper0_NROM tests the NROM mapper (mapper 0)
func TestMapper0_NROM(t *testing.T) {
	t.Run("NROM-128_16KB_PRG", func(t *testing.T) {
		// Test NROM-128 with 16KB PRG ROM
		data := &CartridgeData{
			PRGROM: testPRGROM16KB,
			CHRROM: testCHRROM8KB,
		}
		
		mapper := NewMapper0(data)
		
		// Test PRG ROM reading - should mirror at $C000
		value1 := mapper.ReadPRG(0x8000)
		value2 := mapper.ReadPRG(0xC000)
		if value1 != value2 {
			t.Errorf("NROM-128 mirroring failed: $8000=%02X, $C000=%02X", value1, value2)
		}
		
		// Test specific addresses
		if mapper.ReadPRG(0x8001) != 0x01 {
			t.Errorf("Expected $01 at $8001, got $%02X", mapper.ReadPRG(0x8001))
		}
		
		// Test CHR ROM reading
		if mapper.ReadCHR(0x0000) != 0x00 {
			t.Errorf("Expected $00 at CHR $0000, got $%02X", mapper.ReadCHR(0x0000))
		}
		if mapper.ReadCHR(0x0001) != 0x01 {
			t.Errorf("Expected $01 at CHR $0001, got $%02X", mapper.ReadCHR(0x0001))
		}
	})
	
	t.Run("NROM-256_32KB_PRG", func(t *testing.T) {
		// Test NROM-256 with 32KB PRG ROM
		data := &CartridgeData{
			PRGROM: testPRGROM32KB,
			CHRROM: testCHRROM8KB,
		}
		
		mapper := NewMapper0(data)
		
		// Test PRG ROM reading - no mirroring (32KB ROM fills entire space)
		value1 := mapper.ReadPRG(0x8000)
		value2 := mapper.ReadPRG(0xC000)
		// For 32KB ROM, addresses should map to different data
		// $8000 maps to offset 0x0000, $C000 maps to offset 0x4000
		expected1 := testPRGROM32KB[0x0000]
		expected2 := testPRGROM32KB[0x4000]
		
		if value1 != expected1 {
			t.Errorf("Expected $%02X at $8000, got $%02X", expected1, value1)
		}
		if value2 != expected2 {
			t.Errorf("Expected $%02X at $C000, got $%02X", expected2, value2)
		}
		
		// Test full address range
		if mapper.ReadPRG(0x8000) != 0x00 {
			t.Errorf("Expected $00 at $8000, got $%02X", mapper.ReadPRG(0x8000))
		}
		if mapper.ReadPRG(0xFFFF) != 0xFF {
			t.Errorf("Expected $FF at $FFFF, got $%02X", mapper.ReadPRG(0xFFFF))
		}
	})
	
	t.Run("CHR_RAM_Support", func(t *testing.T) {
		// Test CHR RAM support
		data := &CartridgeData{
			PRGROM: testPRGROM16KB,
			CHRRAM: make([]uint8, 8*1024),
		}
		
		mapper := NewMapper0(data)
		
		// Test CHR RAM write/read
		mapper.WriteCHR(0x1000, 0xAB)
		if mapper.ReadCHR(0x1000) != 0xAB {
			t.Errorf("CHR RAM write/read failed: expected $AB, got $%02X", mapper.ReadCHR(0x1000))
		}
	})
	
	t.Run("PRG_RAM_Support", func(t *testing.T) {
		// Test PRG RAM support (Family Basic variant)
		data := &CartridgeData{
			PRGROM: testPRGROM16KB,
			CHRROM: testCHRROM8KB,
			PRGRAM: make([]uint8, 2*1024), // 2KB PRG RAM
		}
		
		mapper := NewMapper0(data)
		
		// Test PRG RAM write/read
		mapper.WritePRG(0x6000, 0xCD)
		if mapper.ReadPRG(0x6000) != 0xCD {
			t.Errorf("PRG RAM write/read failed: expected $CD, got $%02X", mapper.ReadPRG(0x6000))
		}
		
		// Test ROM area is read-only
		originalValue := mapper.ReadPRG(0x8000)
		mapper.WritePRG(0x8000, 0xFF)
		newValue := mapper.ReadPRG(0x8000)
		if originalValue != newValue {
			t.Errorf("ROM should be read-only: was $%02X, now $%02X", originalValue, newValue)
		}
	})
	
	t.Run("IRQ_Unsupported", func(t *testing.T) {
		// Test that NROM doesn't support IRQ
		data := &CartridgeData{
			PRGROM: testPRGROM16KB,
			CHRROM: testCHRROM8KB,
		}
		
		mapper := NewMapper0(data)
		
		// IRQ should always be false
		if mapper.IsIRQPending() {
			t.Errorf("NROM should not support IRQ")
		}
		
		// Clear IRQ should do nothing (no panic)
		mapper.ClearIRQ()
		
		// Step should do nothing (no panic)
		mapper.Step()
	})
}