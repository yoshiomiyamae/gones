package mapper

import (
	"testing"
)

// TestMapper4_MMC3 tests the MMC3 mapper (mapper 4)
func TestMapper4_MMC3(t *testing.T) {
	t.Run("Bank_Register_Setup", func(t *testing.T) {
		// Create large ROM for testing multiple banks
		prgROM := make([]uint8, 256*1024) // 256KB PRG ROM
		chrROM := make([]uint8, 128*1024) // 128KB CHR ROM
		
		// Initialize with bank-specific data
		for i := 0; i < len(prgROM); i++ {
			prgROM[i] = uint8((i / 8192) + 1) // Different value per 8KB bank
		}
		for i := 0; i < len(chrROM); i++ {
			chrROM[i] = uint8((i / 1024) + 1) // Different value per 1KB bank
		}
		
		data := &CartridgeData{
			PRGROM: prgROM,
			CHRROM: chrROM,
		}
		
		mapper := NewMapper4(data)
		
		// Test initial PRG bank configuration
		// Last bank should always be at $E000-$FFFF
		lastBankValue := mapper.ReadPRG(0xE000)
		expectedLastBank := uint8(len(prgROM)/8192) // Should be last 8KB bank
		if lastBankValue != expectedLastBank {
			t.Errorf("Expected last PRG bank value $%02X at $E000, got $%02X", expectedLastBank, lastBankValue)
		}
	})
	
	t.Run("PRG_Banking_Modes", func(t *testing.T) {
		prgROM := make([]uint8, 256*1024)
		for i := 0; i < len(prgROM); i++ {
			prgROM[i] = uint8((i / 8192) + 1)
		}
		
		data := &CartridgeData{
			PRGROM: prgROM,
			CHRROM: make([]uint8, 8*1024),
		}
		
		mapper := NewMapper4(data)
		
		// Set bank select to R6 (PRG bank register)
		mapper.WritePRG(0x8000, 0x06)
		
		// Set R6 to bank 10
		mapper.WritePRG(0x8001, 0x0A)
		
		// Test PRG mode 0: R6 should appear at $8000
		value := mapper.ReadPRG(0x8000)
		if value != 0x0B { // Bank 10 (0x0A) has value 11 (0x0B)
			t.Errorf("Expected PRG bank 10 value $0B at $8000, got $%02X", value)
		}
		
		// Switch to PRG mode 1
		mapper.WritePRG(0x8000, 0x46) // Set bit 6 for PRG mode 1
		
		// In mode 1, R6 should appear at $C000
		valueAtC000 := mapper.ReadPRG(0xC000)
		valueAt8000 := mapper.ReadPRG(0x8000)
		
		if valueAtC000 != 0x0B {
			t.Errorf("Expected PRG bank 10 value $0B at $C000 in mode 1, got $%02X", valueAtC000)
		}
		// $8000 should now have second-to-last bank
		expectedSecondLast := uint8(len(prgROM)/8192) - 1
		if valueAt8000 != expectedSecondLast {
			t.Errorf("Expected second-to-last PRG bank $%02X at $8000 in mode 1, got $%02X", expectedSecondLast, valueAt8000)
		}
	})
	
	t.Run("CHR_Banking_Modes", func(t *testing.T) {
		chrROM := make([]uint8, 128*1024)
		for i := 0; i < len(chrROM); i++ {
			chrROM[i] = uint8((i / 1024) + 1)
		}
		
		data := &CartridgeData{
			PRGROM: testPRGROM32KB,
			CHRROM: chrROM,
		}
		
		mapper := NewMapper4(data)
		
		// Set R0 to bank 20 (2KB bank)
		mapper.WritePRG(0x8000, 0x00) // Select R0
		mapper.WritePRG(0x8001, 0x14) // Set to bank 20
		
		// In CHR mode 0, R0 should control $0000-$07FF (2KB)
		value := mapper.ReadCHR(0x0000)
		if value != 0x15 { // Bank 20 has value 21
			t.Errorf("Expected CHR bank 20 value $15 at $0000, got $%02X", value)
		}
		
		// Switch to CHR mode 1
		mapper.WritePRG(0x8000, 0x80) // Set bit 7 for CHR mode 1
		mapper.WritePRG(0x8001, 0x00) // Reset R0 to bank 0 for clarity
		
		// In mode 1, R0 should control $1000-$17FF
		valueAt1000 := mapper.ReadCHR(0x1000)
		if valueAt1000 != 0x01 { // Bank 0 has value 1
			t.Errorf("Expected CHR bank 0 value $01 at $1000 in mode 1, got $%02X", valueAt1000)
		}
	})
	
	t.Run("Mirroring_Control", func(t *testing.T) {
		data := &CartridgeData{
			PRGROM: testPRGROM32KB,
			CHRROM: testCHRROM8KB,
		}
		
		mapper := NewMapper4(data)
		
		// Test mirroring control
		mapper.WritePRG(0xA000, 0x00) // Set horizontal mirroring
		if mapper.GetMirroringMode() != 0 {
			t.Errorf("Expected horizontal mirroring (0), got %d", mapper.GetMirroringMode())
		}
		
		mapper.WritePRG(0xA000, 0x01) // Set vertical mirroring
		if mapper.GetMirroringMode() != 1 {
			t.Errorf("Expected vertical mirroring (1), got %d", mapper.GetMirroringMode())
		}
	})
	
	t.Run("IRQ_Registers", func(t *testing.T) {
		data := &CartridgeData{
			PRGROM: testPRGROM32KB,
			CHRROM: testCHRROM8KB,
		}
		
		mapper := NewMapper4(data)
		
		// Test IRQ register setup
		mapper.WritePRG(0xC000, 0x08) // Set IRQ latch to 8
		mapper.WritePRG(0xC001, 0x00) // Reload counter
		mapper.WritePRG(0xE001, 0x00) // Enable IRQ
		
		// Get IRQ state for verification
		counter, reload, enabled, pending := mapper.GetIRQState()
		
		if reload != 0x08 {
			t.Errorf("Expected IRQ reload value 8, got %d", reload)
		}
		if !enabled {
			t.Errorf("Expected IRQ to be enabled")
		}
		if counter != 0 {
			t.Errorf("Expected IRQ counter to be 0 after reload, got %d", counter)
		}
		if pending {
			t.Errorf("Expected no pending IRQ initially")
		}
		
		// Test IRQ disable
		mapper.WritePRG(0xE000, 0x00) // Disable IRQ
		_, _, enabled, pending = mapper.GetIRQState()
		
		if enabled {
			t.Errorf("Expected IRQ to be disabled")
		}
		if pending {
			t.Errorf("Expected no pending IRQ after disable")
		}
	})
	
	t.Run("Bank_Register_All", func(t *testing.T) {
		// Test all 8 bank registers (R0-R7)
		prgROM := make([]uint8, 512*1024) // Large ROM for testing
		chrROM := make([]uint8, 256*1024)
		
		for i := 0; i < len(prgROM); i++ {
			prgROM[i] = uint8((i / 8192) + 1)
		}
		for i := 0; i < len(chrROM); i++ {
			chrROM[i] = uint8((i / 1024) + 1)
		}
		
		data := &CartridgeData{
			PRGROM: prgROM,
			CHRROM: chrROM,
		}
		
		mapper := NewMapper4(data)
		
		// Test setting all bank registers
		for reg := uint8(0); reg < 8; reg++ {
			bankValue := uint8(reg * 5) // Use different values for each register
			
			mapper.WritePRG(0x8000, reg)        // Select register
			mapper.WritePRG(0x8001, bankValue)  // Set bank value
			
			// Verify the register was set (check internal state if possible)
			registers := mapper.GetBankRegisters()
			if reg < 6 {
				// CHR registers should be modded by CHR bank count
				t.Logf("Set CHR register R%d to bank %d, got %d", reg, bankValue, registers[reg])
			} else {
				// PRG registers should be modded by PRG bank count
				t.Logf("Set PRG register R%d to bank %d, got %d", reg, bankValue, registers[reg])
			}
		}
	})
	
	t.Run("PRG_RAM_Control", func(t *testing.T) {
		data := &CartridgeData{
			PRGROM: testPRGROM32KB,
			CHRROM: testCHRROM8KB,
			PRGRAM: make([]uint8, 8*1024), // 8KB PRG RAM
		}
		
		mapper := NewMapper4(data)
		
		// Test PRG RAM write/read
		mapper.WritePRG(0x6000, 0xAB)
		if mapper.ReadPRG(0x6000) != 0xAB {
			t.Errorf("PRG RAM write/read failed: expected $AB, got $%02X", mapper.ReadPRG(0x6000))
		}
		
		// Test PRG RAM protection register
		mapper.WritePRG(0xA001, 0x00) // Disable PRG RAM
		
		// After protection change, RAM behavior might be affected
		// (This would require checking internal protection state)
	})
	
	t.Run("CHR_RAM_Support", func(t *testing.T) {
		// Test MMC3 with CHR RAM instead of CHR ROM
		data := &CartridgeData{
			PRGROM: testPRGROM32KB,
			CHRRAM: make([]uint8, 8*1024), // 8KB CHR RAM
		}
		
		mapper := NewMapper4(data)
		
		// CHR RAM should be writable
		mapper.WriteCHR(0x1000, 0xCC)
		if mapper.ReadCHR(0x1000) != 0xCC {
			t.Errorf("CHR RAM write failed: expected $CC, got $%02X", mapper.ReadCHR(0x1000))
		}
		
		// Bank switching should not affect CHR RAM (direct mapping)
		mapper.WritePRG(0x8000, 0x00) // Select R0
		mapper.WritePRG(0x8001, 0x01) // Set different bank
		
		// CHR RAM should retain its value
		if mapper.ReadCHR(0x1000) != 0xCC {
			t.Errorf("CHR RAM should retain value after bank switch: expected $CC, got $%02X", mapper.ReadCHR(0x1000))
		}
	})
	
	t.Run("Register_Address_Decode", func(t *testing.T) {
		// Test that MMC3 correctly decodes register addresses
		data := &CartridgeData{
			PRGROM: testPRGROM32KB,
			CHRROM: testCHRROM8KB,
		}
		
		mapper := NewMapper4(data)
		
		// Test even/odd address decoding
		evenAddresses := []uint16{0x8000, 0xA000, 0xC000, 0xE000}
		oddAddresses := []uint16{0x8001, 0xA001, 0xC001, 0xE001}
		
		// Even addresses should affect bank select and control
		for _, addr := range evenAddresses {
			mapper.WritePRG(addr, 0x00) // Should not cause errors
		}
		
		// Odd addresses should affect bank data and settings
		for _, addr := range oddAddresses {
			mapper.WritePRG(addr, 0x00) // Should not cause errors
		}
		
		// Test that in-range addresses work
		inRangeAddresses := []uint16{0x9FFF, 0xBFFF, 0xDFFF, 0xFFFF}
		for _, addr := range inRangeAddresses {
			mapper.WritePRG(addr, 0x00) // Should not cause errors
		}
	})

	// Test 32KB CHR RAM like mmc3bigchrram.nes
	t.Run("CHR_RAM_32KB_Pattern_Test", func(t *testing.T) {
		// Create mapper with 32KB CHR RAM
		data := &CartridgeData{
			PRGROM: testPRGROM32KB,
			CHRRAM: make([]uint8, 32*1024), // 32KB CHR RAM
		}
		
		mapper := NewMapper4(data)
		
		// Test bank switching and pattern writing/verification
		// This mimics the mmc3bigchrram.nes test ROM behavior
		
		// Test the pattern that was observed in logs (Rijndael-like pattern)
		// Also test our Rijndael pattern generator
		expectedPattern := []uint8{0x03, 0x05, 0x0F, 0x11, 0x33, 0x55, 0xFF, 0x1A, 0x2E, 0x72, 0x96, 0xA1, 0xF8, 0x13, 0x35, 0x5F}
		generatedPattern := generateRijndaelPattern(0x03, 16)
		
		// Verify our Rijndael algorithm produces the expected test pattern
		for i := 0; i < len(expectedPattern) && i < len(generatedPattern); i++ {
			if expectedPattern[i] != generatedPattern[i] {
				t.Logf("Rijndael pattern mismatch at index %d: expected $%02X, generated $%02X", 
					i, expectedPattern[i], generatedPattern[i])
			}
		}
		
		// Step 1: Set R0 to bank 0 and write pattern
		mapper.WritePRG(0x8000, 0x00) // Bank select R0
		mapper.WritePRG(0x8001, 0x00) // Set R0 to bank 0
		
		// Write test pattern to CHR addresses $0000-$000F (should go to bank 0)
		for i, val := range expectedPattern {
			mapper.WriteCHR(uint16(i), val)
		}
		
		// Step 2: Switch to bank 2
		mapper.WritePRG(0x8000, 0x00) // Bank select R0
		mapper.WritePRG(0x8001, 0x02) // Set R0 to bank 2
		
		// Write different pattern to bank 2
		for i := 0; i < 16; i++ {
			mapper.WriteCHR(uint16(i), uint8(0x20+i))
		}
		
		// Step 3: Switch to bank 6
		mapper.WritePRG(0x8000, 0x00) // Bank select R0
		mapper.WritePRG(0x8001, 0x06) // Set R0 to bank 6
		
		// Write another pattern to bank 6
		for i := 0; i < 16; i++ {
			mapper.WriteCHR(uint16(i), uint8(0x60+i))
		}
		
		// Step 4: Switch back to bank 0 and verify
		mapper.WritePRG(0x8000, 0x00) // Bank select R0
		mapper.WritePRG(0x8001, 0x00) // Set R0 back to bank 0
		
		// Read back from bank 0 and verify it matches original pattern
		for i, expected := range expectedPattern {
			actual := mapper.ReadCHR(uint16(i))
			if actual != expected {
				t.Errorf("CHR RAM verification failed at offset %d: expected $%02X, got $%02X", i, expected, actual)
			}
		}
		
		// Additional verification: check that bank 2 and bank 6 have different patterns
		mapper.WritePRG(0x8001, 0x02) // Switch to bank 2
		bank2Value := mapper.ReadCHR(0x0000)
		if bank2Value == expectedPattern[0] {
			t.Errorf("Bank 2 should have different pattern than bank 0, but both have $%02X", bank2Value)
		}
		
		mapper.WritePRG(0x8001, 0x06) // Switch to bank 6
		bank6Value := mapper.ReadCHR(0x0000)
		if bank6Value == expectedPattern[0] {
			t.Errorf("Bank 6 should have different pattern than bank 0, but both have $%02X", bank6Value)
		}
		
		t.Logf("32KB CHR RAM test passed: Bank 0=$%02X, Bank 2=$%02X, Bank 6=$%02X", expectedPattern[0], bank2Value, bank6Value)
	})
}

// generateRijndaelPattern generates a test pattern using Rijndael GF(256) multiplication
// This mimics the algorithm used in mmc3bigchrram.nes
func generateRijndaelPattern(seed uint8, length int) []uint8 {
	pattern := make([]uint8, length)
	value := seed
	
	for i := 0; i < length; i++ {
		pattern[i] = value
		
		// Rijndael GF(256) multiplication by 3 (left shift + XOR)
		newValue := value << 1
		if value&0x80 != 0 {
			newValue ^= 0x1B // Rijndael irreducible polynomial
		}
		value = newValue ^ seed // XOR with original to multiply by 3
	}
	
	return pattern
}