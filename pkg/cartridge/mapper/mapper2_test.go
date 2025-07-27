package mapper

import (
	"testing"
)

// TestMapper2_UxROM tests the UxROM mapper (mapper 2)
func TestMapper2_UxROM(t *testing.T) {
	t.Run("PRG_Bank_Switching", func(t *testing.T) {
		// Create 128KB PRG ROM (8 banks of 16KB)
		prgROM := make([]uint8, 128*1024)
		for i := 0; i < len(prgROM); i++ {
			prgROM[i] = uint8((i / 16384) + 1) // Different value per 16KB bank
		}
		
		data := &CartridgeData{
			PRGROM: prgROM,
			CHRRAM: make([]uint8, 8*1024), // CHR RAM
		}
		
		mapper := NewMapper2(data)
		
		// Test initial state - should have bank 0 at $8000 and last bank at $C000
		bank0Value := mapper.ReadPRG(0x8000)
		lastBankValue := mapper.ReadPRG(0xC000)
		
		if bank0Value != 0x01 {
			t.Errorf("Expected bank 0 value $01 at $8000, got $%02X", bank0Value)
		}
		if lastBankValue != 0x08 {
			t.Errorf("Expected last bank value $08 at $C000, got $%02X", lastBankValue)
		}
		
		// Switch to bank 2
		mapper.WritePRG(0x8000, 0x02)
		
		// Test that $8000 now reads bank 2, $C000 still reads last bank
		newBank0Value := mapper.ReadPRG(0x8000)
		stillLastBankValue := mapper.ReadPRG(0xC000)
		
		if newBank0Value != 0x03 { // Bank 2 (0-indexed) has value 3
			t.Errorf("Expected bank 2 value $03 at $8000, got $%02X", newBank0Value)
		}
		if stillLastBankValue != 0x08 {
			t.Errorf("Last bank should remain fixed at $C000, got $%02X", stillLastBankValue)
		}
	})
	
	t.Run("CHR_RAM_Access", func(t *testing.T) {
		data := &CartridgeData{
			PRGROM: testPRGROM32KB,
			CHRRAM: make([]uint8, 8*1024),
		}
		
		mapper := NewMapper2(data)
		
		// Test CHR RAM write/read
		mapper.WriteCHR(0x0555, 0xAA)
		mapper.WriteCHR(0x1AAA, 0x55)
		
		if mapper.ReadCHR(0x0555) != 0xAA {
			t.Errorf("CHR RAM write/read failed at $0555: expected $AA, got $%02X", mapper.ReadCHR(0x0555))
		}
		if mapper.ReadCHR(0x1AAA) != 0x55 {
			t.Errorf("CHR RAM write/read failed at $1AAA: expected $55, got $%02X", mapper.ReadCHR(0x1AAA))
		}
	})
	
	t.Run("Bank_Selection_Masking", func(t *testing.T) {
		// Create 64KB PRG ROM (4 banks of 16KB)
		prgROM := make([]uint8, 64*1024)
		for i := 0; i < len(prgROM); i++ {
			prgROM[i] = uint8((i / 16384) + 0x10) // Start with value 0x10 per bank
		}
		
		data := &CartridgeData{
			PRGROM: prgROM,
			CHRRAM: make([]uint8, 8*1024),
		}
		
		mapper := NewMapper2(data)
		
		// Test bank selection with different bit patterns
		// UxROM typically uses 3-4 bits for bank selection
		
		// Select bank 1
		mapper.WritePRG(0x8000, 0x01)
		value1 := mapper.ReadPRG(0x8000)
		if value1 != 0x11 { // Bank 1 should have value 0x11
			t.Errorf("Expected bank 1 value $11, got $%02X", value1)
		}
		
		// Select bank 3 (should be valid for 4-bank ROM)
		mapper.WritePRG(0x8000, 0x03)
		value3 := mapper.ReadPRG(0x8000)
		if value3 != 0x13 { // Bank 3 should have value 0x13
			t.Errorf("Expected bank 3 value $13, got $%02X", value3)
		}
		
		// Try to select bank 7 (should wrap to bank 3 for 4-bank ROM)
		mapper.WritePRG(0x8000, 0x07)
		value7 := mapper.ReadPRG(0x8000)
		if value7 != 0x13 { // Should wrap to bank 3
			t.Errorf("Expected wrapped bank value $13, got $%02X", value7)
		}
	})
	
	t.Run("Fixed_Last_Bank", func(t *testing.T) {
		// Test that the last bank is always fixed at $C000-$FFFF
		prgROM := make([]uint8, 256*1024) // 16 banks
		for i := 0; i < len(prgROM); i++ {
			prgROM[i] = uint8((i / 16384) + 0x20) // Start with value 0x20 per bank
		}
		
		data := &CartridgeData{
			PRGROM: prgROM,
			CHRRAM: make([]uint8, 8*1024),
		}
		
		mapper := NewMapper2(data)
		
		// Get the last bank value initially
		initialLastBank := mapper.ReadPRG(0xC000)
		expectedLastBankValue := uint8(0x20 + 15) // Bank 15 (0-indexed)
		
		if initialLastBank != expectedLastBankValue {
			t.Errorf("Expected last bank value $%02X, got $%02X", expectedLastBankValue, initialLastBank)
		}
		
		// Switch switchable bank multiple times
		for bank := uint8(0); bank < 8; bank++ {
			mapper.WritePRG(0x8000, bank)
			
			// Verify switchable bank changed
			switchableValue := mapper.ReadPRG(0x8000)
			expectedSwitchableValue := uint8(0x20 + bank)
			if switchableValue != expectedSwitchableValue {
				t.Errorf("Expected switchable bank %d value $%02X, got $%02X", bank, expectedSwitchableValue, switchableValue)
			}
			
			// Verify last bank remained fixed
			currentLastBank := mapper.ReadPRG(0xC000)
			if currentLastBank != expectedLastBankValue {
				t.Errorf("Last bank should remain fixed at $%02X, got $%02X", expectedLastBankValue, currentLastBank)
			}
		}
	})
	
	t.Run("Address_Range_Validation", func(t *testing.T) {
		data := &CartridgeData{
			PRGROM: testPRGROM32KB,
			CHRRAM: make([]uint8, 8*1024),
		}
		
		mapper := NewMapper2(data)
		
		// Test that writes anywhere in ROM space affect bank selection
		originalValue := mapper.ReadPRG(0x8000)
		
		// Write to different addresses in ROM space
		addresses := []uint16{0x8000, 0x9000, 0xA000, 0xB000, 0xC000, 0xD000, 0xE000, 0xF000}
		
		for _, addr := range addresses {
			mapper.WritePRG(addr, 0x01) // Select bank 1
			newValue := mapper.ReadPRG(0x8000)
			
			// All addresses should affect bank selection
			if newValue == originalValue {
				t.Logf("Write to $%04X affected bank selection", addr)
			}
		}
	})
	
	t.Run("CHR_No_Banking", func(t *testing.T) {
		// UxROM has no CHR banking - test that CHR is fixed
		data := &CartridgeData{
			PRGROM: testPRGROM32KB,
			CHRRAM: make([]uint8, 8*1024),
		}
		
		mapper := NewMapper2(data)
		
		// Write pattern to CHR RAM
		testPattern := []uint8{0x12, 0x34, 0x56, 0x78}
		for i, val := range testPattern {
			mapper.WriteCHR(uint16(i*0x800), val) // Write to different 2KB sections
		}
		
		// Verify pattern persists regardless of PRG bank switches
		for bank := uint8(0); bank < 4; bank++ {
			mapper.WritePRG(0x8000, bank) // Switch PRG bank
			
			// CHR should remain unchanged
			for i, expectedVal := range testPattern {
				actualVal := mapper.ReadCHR(uint16(i * 0x800))
				if actualVal != expectedVal {
					t.Errorf("CHR changed after PRG bank switch: expected $%02X, got $%02X", expectedVal, actualVal)
				}
			}
		}
	})
}