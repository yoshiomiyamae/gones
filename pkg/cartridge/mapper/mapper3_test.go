package mapper

import (
	"testing"
)

// TestMapper3_CNROM tests the CNROM mapper (mapper 3)
func TestMapper3_CNROM(t *testing.T) {
	t.Run("CHR_Bank_Switching", func(t *testing.T) {
		data := &CartridgeData{
			PRGROM: testPRGROM32KB,
			CHRROM: testCHRROM32KB, // 32KB CHR ROM (4 banks of 8KB)
		}
		
		// Initialize CHR ROM with bank-specific data
		for i := 0; i < len(data.CHRROM); i++ {
			data.CHRROM[i] = uint8((i / 8192) + 1) // Different value per 8KB bank
		}
		
		mapper := NewMapper3(data)
		
		// Test initial bank (should be bank 0)
		initialValue := mapper.ReadCHR(0x0000)
		if initialValue != 0x01 {
			t.Errorf("Expected CHR bank 0 value $01, got $%02X", initialValue)
		}
		
		// Switch to CHR bank 2
		mapper.WritePRG(0x8000, 0x02)
		
		// Test that CHR reading now returns bank 2 data
		newValue := mapper.ReadCHR(0x0000)
		if newValue != 0x03 { // Bank 2 has value 3
			t.Errorf("Expected CHR bank 2 value $03, got $%02X", newValue)
		}
		
		// Test different addresses within the bank
		midBankValue := mapper.ReadCHR(0x1000)
		if midBankValue != 0x03 {
			t.Errorf("Expected same bank value $03 at $1000, got $%02X", midBankValue)
		}
	})
	
	t.Run("PRG_ROM_Fixed_32KB", func(t *testing.T) {
		data := &CartridgeData{
			PRGROM: testPRGROM32KB,
			CHRROM: testCHRROM32KB,
		}
		
		mapper := NewMapper3(data)
		
		// PRG ROM should be 32KB fixed - no banking or mirroring
		value1 := mapper.ReadPRG(0x8000)
		value2 := mapper.ReadPRG(0xFFFF)
		
		if value1 != 0x00 {
			t.Errorf("Expected $00 at $8000, got $%02X", value1)
		}
		if value2 != 0xFF {
			t.Errorf("Expected $FF at $FFFF, got $%02X", value2)
		}
		
		// Test that full 32KB range is accessible without mirroring
		midValue := mapper.ReadPRG(0xC000) // Should be different from 0x8000 in 32KB ROM
		if midValue == value1 && len(data.PRGROM) == 32768 {
			// If it's the same, check if this is intentional mirroring or actual 32KB data
			data.PRGROM[0x4000] = 0xAA // Set different value at 16KB offset
			newMidValue := mapper.ReadPRG(0xC000)
			if newMidValue != 0xAA {
				t.Errorf("32KB PRG ROM not properly mapped: expected $AA at $C000, got $%02X", newMidValue)
			}
		}
		
		// Writing to PRG area should only affect CHR banking, not PRG reading
		originalValue := mapper.ReadPRG(0x9000)
		mapper.WritePRG(0x9000, 0xFF) // This should switch CHR bank, not affect PRG
		newValue := mapper.ReadPRG(0x9000)
		
		if originalValue != newValue {
			t.Errorf("PRG ROM should be unaffected by writes: was $%02X, now $%02X", originalValue, newValue)
		}
	})
	
	t.Run("Bank_Select_Masking", func(t *testing.T) {
		// Test with different CHR ROM sizes to verify bank masking
		chrROM16KB := make([]uint8, 16*1024) // 2 banks
		for i := 0; i < len(chrROM16KB); i++ {
			chrROM16KB[i] = uint8((i / 8192) + 0x10)
		}
		
		data := &CartridgeData{
			PRGROM: testPRGROM32KB,
			CHRROM: chrROM16KB,
		}
		
		mapper := NewMapper3(data)
		
		// Select bank 1 (valid)
		mapper.WritePRG(0x8000, 0x01)
		value1 := mapper.ReadCHR(0x0000)
		if value1 != 0x11 { // Bank 1 value
			t.Errorf("Expected bank 1 value $11, got $%02X", value1)
		}
		
		// Select bank 3 (should wrap to bank 1 for 2-bank ROM)
		mapper.WritePRG(0x8000, 0x03)
		value3 := mapper.ReadCHR(0x0000)
		if value3 != 0x11 { // Should wrap to bank 1
			t.Errorf("Expected wrapped bank value $11, got $%02X", value3)
		}
		
		// Select bank 0 again
		mapper.WritePRG(0x8000, 0x00)
		value0 := mapper.ReadCHR(0x0000)
		if value0 != 0x10 { // Bank 0 value
			t.Errorf("Expected bank 0 value $10, got $%02X", value0)
		}
	})
	
	t.Run("Multiple_Write_Addresses", func(t *testing.T) {
		data := &CartridgeData{
			PRGROM: testPRGROM32KB,
			CHRROM: testCHRROM32KB,
		}
		
		// Initialize CHR ROM with bank-specific data
		for i := 0; i < len(data.CHRROM); i++ {
			data.CHRROM[i] = uint8((i / 8192) + 0x20)
		}
		
		mapper := NewMapper3(data)
		
		// Test that writes to any address in ROM space affect CHR bank selection
		testAddresses := []uint16{0x8000, 0x9000, 0xA000, 0xB000, 0xC000, 0xD000, 0xE000, 0xF000}
		
		for i, addr := range testAddresses {
			bankNum := uint8(i % 4) // Cycle through available banks
			mapper.WritePRG(addr, bankNum)
			
			expectedValue := uint8(0x20 + bankNum)
			actualValue := mapper.ReadCHR(0x0000)
			
			if actualValue != expectedValue {
				t.Errorf("Write to $%04X failed: expected $%02X, got $%02X", addr, expectedValue, actualValue)
			}
		}
	})
	
	t.Run("CHR_ROM_vs_RAM", func(t *testing.T) {
		// Test with CHR ROM (read-only)
		dataROM := &CartridgeData{
			PRGROM: testPRGROM32KB,
			CHRROM: testCHRROM32KB,
		}
		
		mapperROM := NewMapper3(dataROM)
		
		originalValue := mapperROM.ReadCHR(0x1000)
		mapperROM.WriteCHR(0x1000, 0xFF) // Should be ignored
		newValue := mapperROM.ReadCHR(0x1000)
		
		if originalValue != newValue {
			t.Errorf("CHR ROM should be read-only: was $%02X, now $%02X", originalValue, newValue)
		}
		
		// Test with CHR RAM (writable)
		dataRAM := &CartridgeData{
			PRGROM: testPRGROM32KB,
			CHRRAM: make([]uint8, 8*1024), // Single 8KB CHR RAM
		}
		
		mapperRAM := NewMapper3(dataRAM)
		
		// CHR RAM should be writable but not banked
		mapperRAM.WriteCHR(0x1000, 0xAA)
		if mapperRAM.ReadCHR(0x1000) != 0xAA {
			t.Errorf("CHR RAM write failed: expected $AA, got $%02X", mapperRAM.ReadCHR(0x1000))
		}
		
		// Bank switching should not affect CHR RAM
		mapperRAM.WritePRG(0x8000, 0x01) // Try to switch bank
		if mapperRAM.ReadCHR(0x1000) != 0xAA {
			t.Errorf("CHR RAM should not be affected by bank switching")
		}
	})
	
	t.Run("Bus_Conflicts", func(t *testing.T) {
		// Test AND-type bus conflicts (submapper 2)
		data := &CartridgeData{
			PRGROM: testPRGROM32KB,
			CHRROM: testCHRROM32KB,
		}
		
		// Set specific PRG ROM pattern for testing
		data.PRGROM[0x0000] = 0x03 // At $8000: allows banks 0-1
		data.PRGROM[0x1000] = 0x02 // At $9000: allows banks 0,2
		data.PRGROM[0x2000] = 0x01 // At $A000: allows bank 0 only
		
		// Initialize CHR ROM with bank-specific data
		for i := 0; i < len(data.CHRROM); i++ {
			data.CHRROM[i] = uint8((i / 8192) + 0x40)
		}
		
		mapper := NewMapper3(data)
		mapper.SetBusConflictMode(2) // AND-type conflicts
		
		// Test bus conflict at $8000: write 0x03, PRG=0x03, effective=0x03&0x03=0x03->bank 3
		mapper.WritePRG(0x8000, 0x03)
		if mapper.GetCurrentCHRBank() != 0x03 {
			t.Errorf("Expected bank 3 with no conflict, got %d", mapper.GetCurrentCHRBank())
		}
		
		// Test bus conflict at $9000: write 0x03, PRG=0x02, effective=0x03&0x02=0x02->bank 2
		mapper.WritePRG(0x9000, 0x03)
		if mapper.GetCurrentCHRBank() != 0x02 {
			t.Errorf("Expected bank 2 from bus conflict, got %d", mapper.GetCurrentCHRBank())
		}
		
		// Test bus conflict at $A000: write 0x03, PRG=0x01, effective=0x03&0x01=0x01->bank 1
		mapper.WritePRG(0xA000, 0x03)
		if mapper.GetCurrentCHRBank() != 0x01 {
			t.Errorf("Expected bank 1 from bus conflict, got %d", mapper.GetCurrentCHRBank())
		}
		
		// Test no bus conflicts (submapper 1)
		mapper.SetBusConflictMode(1)
		mapper.WritePRG(0xA000, 0x03) // PRG=0x01, but no conflict, so bank=0x03&0x03=3
		if mapper.GetCurrentCHRBank() != 0x03 {
			t.Errorf("Expected bank 3 with no conflicts, got %d", mapper.GetCurrentCHRBank())
		}
	})
	
	t.Run("Full_Address_Range", func(t *testing.T) {
		data := &CartridgeData{
			PRGROM: testPRGROM32KB,
			CHRROM: testCHRROM32KB,
		}
		
		// Initialize CHR ROM with address-based pattern
		for i := 0; i < len(data.CHRROM); i++ {
			data.CHRROM[i] = uint8(i & 0xFF)
		}
		
		mapper := NewMapper3(data)
		
		// Test reading across full CHR address range in different banks
		for bank := uint8(0); bank < 4; bank++ {
			mapper.WritePRG(0x8000, bank)
			
			// Test key addresses in each bank
			testAddresses := []uint16{0x0000, 0x0800, 0x1000, 0x1800, 0x1FFF}
			
			for _, addr := range testAddresses {
				value := mapper.ReadCHR(addr)
				expectedValue := uint8((uint32(bank)*8192 + uint32(addr)) & 0xFF)
				
				if value != expectedValue {
					t.Errorf("Bank %d addr $%04X: expected $%02X, got $%02X", bank, addr, expectedValue, value)
				}
			}
		}
	})
}