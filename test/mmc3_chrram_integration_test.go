package test

import (
	"testing"

	"github.com/yoshiomiyamaegones/pkg/cartridge"
	"github.com/yoshiomiyamaegones/pkg/cartridge/mapper"
	"github.com/yoshiomiyamaegones/pkg/nes"
)

// TestMMC3_CHR_RAM_Integration tests the actual CPU+PPU+MMC3 integration
// This mimics the mmc3bigchrram.nes test ROM behavior exactly
func TestMMC3_CHR_RAM_Integration(t *testing.T) {
	// Create a test cartridge with 32KB CHR RAM (like mmc3bigchrram.nes)
	prgROM := make([]uint8, 32*1024) // 32KB PRG ROM
	chrRAM := make([]uint8, 32*1024) // 32KB CHR RAM

	// Set up a simple PRG ROM that can perform the test
	// This simulates the test ROM's logic
	testCode := []uint8{
		// Test program starts at $8000
		0xA9, 0x00, // LDA #$00
		0x8D, 0x06, 0x20, // STA $2006 (PPUADDR high)
		0xA9, 0x00, // LDA #$00
		0x8D, 0x06, 0x20, // STA $2006 (PPUADDR low) - now pointing to $0000

		// Write Rijndael pattern to CHR addresses $0000-$000F
		0xA9, 0x03, // LDA #$03
		0x8D, 0x07, 0x20, // STA $2007 (PPUDATA)
		0xA9, 0x05, // LDA #$05
		0x8D, 0x07, 0x20, // STA $2007
		0xA9, 0x0F, // LDA #$0F
		0x8D, 0x07, 0x20, // STA $2007
		0xA9, 0x11, // LDA #$11
		0x8D, 0x07, 0x20, // STA $2007
		0xA9, 0x33, // LDA #$33
		0x8D, 0x07, 0x20, // STA $2007
		0xA9, 0x55, // LDA #$55
		0x8D, 0x07, 0x20, // STA $2007
		0xA9, 0xFF, // LDA #$FF
		0x8D, 0x07, 0x20, // STA $2007
		0xA9, 0x1A, // LDA #$1A
		0x8D, 0x07, 0x20, // STA $2007
		0xA9, 0x2E, // LDA #$2E
		0x8D, 0x07, 0x20, // STA $2007
		0xA9, 0x72, // LDA #$72
		0x8D, 0x07, 0x20, // STA $2007
		0xA9, 0x96, // LDA #$96
		0x8D, 0x07, 0x20, // STA $2007
		0xA9, 0xA1, // LDA #$A1
		0x8D, 0x07, 0x20, // STA $2007
		0xA9, 0xF8, // LDA #$F8
		0x8D, 0x07, 0x20, // STA $2007
		0xA9, 0x13, // LDA #$13
		0x8D, 0x07, 0x20, // STA $2007
		0xA9, 0x35, // LDA #$35
		0x8D, 0x07, 0x20, // STA $2007
		0xA9, 0x5F, // LDA #$5F
		0x8D, 0x07, 0x20, // STA $2007

		// Switch to bank 2 using MMC3 registers
		0xA9, 0x00, // LDA #$00 (select R0)
		0x8D, 0x00, 0x80, // STA $8000 (bank select)
		0xA9, 0x02, // LDA #$02 (bank 2)
		0x8D, 0x01, 0x80, // STA $8001 (bank data)

		// Write different pattern to bank 2
		0xA9, 0x00, // LDA #$00
		0x8D, 0x06, 0x20, // STA $2006 (PPUADDR high)
		0xA9, 0x00, // LDA #$00
		0x8D, 0x06, 0x20, // STA $2006 (PPUADDR low)
		0xA9, 0x20, // LDA #$20 (different pattern)
		0x8D, 0x07, 0x20, // STA $2007
		0xA9, 0x21, // LDA #$21
		0x8D, 0x07, 0x20, // STA $2007
		0xA9, 0x22, // LDA #$22
		0x8D, 0x07, 0x20, // STA $2007
		0xA9, 0x23, // LDA #$23
		0x8D, 0x07, 0x20, // STA $2007

		// Switch to bank 6
		0xA9, 0x00, // LDA #$00
		0x8D, 0x00, 0x80, // STA $8000
		0xA9, 0x06, // LDA #$06 (bank 6)
		0x8D, 0x01, 0x80, // STA $8001

		// Write pattern to bank 6
		0xA9, 0x00, // LDA #$00
		0x8D, 0x06, 0x20, // STA $2006
		0xA9, 0x00, // LDA #$00
		0x8D, 0x06, 0x20, // STA $2006
		0xA9, 0x60, // LDA #$60 (bank 6 pattern)
		0x8D, 0x07, 0x20, // STA $2007
		0xA9, 0x61, // LDA #$61
		0x8D, 0x07, 0x20, // STA $2007
		0xA9, 0x62, // LDA #$62
		0x8D, 0x07, 0x20, // STA $2007
		0xA9, 0x63, // LDA #$63
		0x8D, 0x07, 0x20, // STA $2007

		// Switch back to bank 0
		0xA9, 0x00, // LDA #$00
		0x8D, 0x00, 0x80, // STA $8000
		0xA9, 0x00, // LDA #$00 (bank 0)
		0x8D, 0x01, 0x80, // STA $8001

		// Read back from bank 0 and verify
		0xA9, 0x00, // LDA #$00
		0x8D, 0x06, 0x20, // STA $2006
		0xA9, 0x00, // LDA #$00
		0x8D, 0x06, 0x20, // STA $2006

		// The test ROM would read from $2007 here and compare values
		// We'll simulate this in our test code

		0x4C, 0x00, 0x80, // JMP $8000 (infinite loop)
	}

	// Copy test code to PRG ROM
	copy(prgROM, testCode)

	// Set reset vector to point to our test code
	prgROM[0x7FFC] = 0x00 // Reset vector low
	prgROM[0x7FFD] = 0x80 // Reset vector high

	cartData := &mapper.CartridgeData{
		PRGROM: prgROM,
		CHRRAM: chrRAM,
	}

	// Create cartridge with MMC3 mapper
	cart := &cartridge.Cartridge{
		PRGROM: prgROM,
		CHRRAM: chrRAM,
		Mapper: mapper.NewMapper4(cartData),
	}

	// Create NES system
	nesSystem := nes.NewNES()
	nesSystem.LoadCartridge(cart)
	nesSystem.Reset()

	// Execute the test program step by step and verify CHR RAM operations

	// Step 1: Execute the initial pattern write to bank 0
	// Run enough cycles to execute the pattern writing
	for i := 0; i < 1000; i++ {
		nesSystem.Step()
	}

	// Check that bank 0 has the expected pattern
	mapper4 := cart.Mapper.(*mapper.Mapper4)

	// Verify bank 0 has the Rijndael pattern
	expectedPattern := []uint8{0x03, 0x05, 0x0F, 0x11, 0x33, 0x55, 0xFF, 0x1A, 0x2E, 0x72, 0x96, 0xA1, 0xF8, 0x13, 0x35, 0x5F}

	// Switch to bank 0 and read
	mapper4.WritePRG(0x8000, 0x00)
	mapper4.WritePRG(0x8001, 0x00)

	for i, expected := range expectedPattern {
		actual := mapper4.ReadCHR(uint16(i))
		if actual != expected {
			t.Errorf("Bank 0 pattern mismatch at offset %d: expected $%02X, got $%02X", i, expected, actual)
		}
	}

	// Continue execution to test bank switching
	for i := 0; i < 2000; i++ {
		nesSystem.Step()
	}

	// Check bank 2 has different pattern
	mapper4.WritePRG(0x8000, 0x00)
	mapper4.WritePRG(0x8001, 0x02)

	bank2Value := mapper4.ReadCHR(0x0000)
	t.Logf("Bank 2 value at offset 0: $%02X (expected $20)", bank2Value)

	// Check first few bytes of bank 2
	for i := 0; i < 4; i++ {
		val := mapper4.ReadCHR(uint16(i))
		t.Logf("Bank 2 offset %d: $%02X", i, val)
	}

	// Check bank 6 has different pattern
	mapper4.WritePRG(0x8000, 0x00)
	mapper4.WritePRG(0x8001, 0x06)

	bank6Value := mapper4.ReadCHR(0x0000)
	t.Logf("Bank 6 value at offset 0: $%02X (expected $60)", bank6Value)

	// Check first few bytes of bank 6
	for i := 0; i < 4; i++ {
		val := mapper4.ReadCHR(uint16(i))
		t.Logf("Bank 6 offset %d: $%02X", i, val)
	}

	// Final test: switch back to bank 0 and verify pattern is preserved
	mapper4.WritePRG(0x8000, 0x00)
	mapper4.WritePRG(0x8001, 0x00)

	for i, expected := range expectedPattern {
		actual := mapper4.ReadCHR(uint16(i))
		if actual != expected {
			t.Errorf("Bank 0 pattern not preserved after bank switching at offset %d: expected $%02X, got $%02X", i, expected, actual)
		}
	}

	t.Logf("Integration test completed: Bank 0=$%02X, Bank 2=$%02X, Bank 6=$%02X",
		expectedPattern[0], bank2Value, bank6Value)
}

// TestMMC3_Direct_CHR_Write tests direct CHR RAM writing without CPU execution
func TestMMC3_Direct_CHR_Write(t *testing.T) {
	// Create cartridge with 32KB CHR RAM
	cartData := &mapper.CartridgeData{
		PRGROM: make([]uint8, 32*1024),
		CHRRAM: make([]uint8, 32*1024),
	}

	cart := &cartridge.Cartridge{
		PRGROM: make([]uint8, 32*1024),
		CHRRAM: make([]uint8, 32*1024),
		Mapper: mapper.NewMapper4(cartData),
	}

	nesSystem := nes.NewNES()
	nesSystem.LoadCartridge(cart)

	mapper4 := cart.Mapper.(*mapper.Mapper4)
	mem := nesSystem.Memory

	// Test 1: Write to bank 0 via PPU registers
	t.Log("=== Test 1: Write to bank 0 ===")

	// Ensure we're on bank 0
	mapper4.WritePRG(0x8000, 0x00) // Select R0
	mapper4.WritePRG(0x8001, 0x00) // Set R0 to bank 0

	// Set PPUADDR to $0000
	mem.Write(0x2006, 0x00)
	mem.Write(0x2006, 0x00)

	// Write test pattern
	testPattern := []uint8{0x03, 0x05, 0x0F, 0x11}
	for i, value := range testPattern {
		mem.Write(0x2007, value)
		t.Logf("Wrote $%02X to PPU at step %d", value, i)
	}

	// Read back via direct CHR access
	for i, expected := range testPattern {
		actual := mapper4.ReadCHR(uint16(i))
		t.Logf("Bank 0 offset %d: wrote $%02X, read $%02X", i, expected, actual)
		if actual != expected {
			t.Errorf("Bank 0 mismatch at offset %d: expected $%02X, got $%02X", i, expected, actual)
		}
	}

	// Test 2: Switch to bank 2 and write
	t.Log("=== Test 2: Write to bank 2 ===")

	mapper4.WritePRG(0x8000, 0x00) // Select R0
	mapper4.WritePRG(0x8001, 0x02) // Set R0 to bank 2

	// Set PPUADDR to $0000
	mem.Write(0x2006, 0x00)
	mem.Write(0x2006, 0x00)

	// Write different pattern to bank 2
	bank2Pattern := []uint8{0x20, 0x21, 0x22, 0x23}
	for i, value := range bank2Pattern {
		mem.Write(0x2007, value)
		t.Logf("Wrote $%02X to bank 2 at step %d", value, i)
	}

	// Read back from bank 2
	for i, expected := range bank2Pattern {
		actual := mapper4.ReadCHR(uint16(i))
		t.Logf("Bank 2 offset %d: wrote $%02X, read $%02X", i, expected, actual)
		if actual != expected {
			t.Errorf("Bank 2 mismatch at offset %d: expected $%02X, got $%02X", i, expected, actual)
		}
	}

	// Test 3: Switch back to bank 0 and verify
	t.Log("=== Test 3: Verify bank 0 preserved ===")

	mapper4.WritePRG(0x8000, 0x00) // Select R0
	mapper4.WritePRG(0x8001, 0x00) // Set R0 to bank 0

	// Read bank 0 again
	for i, expected := range testPattern {
		actual := mapper4.ReadCHR(uint16(i))
		t.Logf("Bank 0 preserved check offset %d: expected $%02X, read $%02X", i, expected, actual)
		if actual != expected {
			t.Errorf("Bank 0 not preserved at offset %d: expected $%02X, got $%02X", i, expected, actual)
		}
	}

	t.Log("Direct CHR write test completed")
}

// TestMMC3_PPU_Integration tests PPU register access through CPU memory mapping
func TestMMC3_PPU_Integration(t *testing.T) {
	// Create minimal cartridge
	cartData := &mapper.CartridgeData{
		PRGROM: make([]uint8, 32*1024),
		CHRRAM: make([]uint8, 32*1024),
	}

	cart := &cartridge.Cartridge{
		PRGROM: make([]uint8, 32*1024),
		CHRRAM: make([]uint8, 32*1024),
		Mapper: mapper.NewMapper4(cartData),
	}

	// Create NES system to get properly wired components
	nesSystem := nes.NewNES()
	nesSystem.LoadCartridge(cart)

	mem := nesSystem.Memory

	// Test PPUADDR/PPUDATA sequence

	// Set PPUADDR to $0000
	mem.Write(0x2006, 0x00) // High byte
	mem.Write(0x2006, 0x00) // Low byte

	// Write test pattern via PPUDATA
	testPattern := []uint8{0x03, 0x05, 0x0F, 0x11}
	for _, value := range testPattern {
		mem.Write(0x2007, value)
	}

	// Reset PPUADDR to $0000
	mem.Write(0x2006, 0x00)
	mem.Write(0x2006, 0x00)

	// Read back via PPUDATA
	for i, expected := range testPattern {
		actual := mem.Read(0x2007)
		if actual != expected {
			t.Errorf("PPU integration test failed at index %d: expected $%02X, got $%02X", i, expected, actual)
		}
	}

	// Test bank switching affects CHR reads
	mapper4 := cart.Mapper.(*mapper.Mapper4)

	// Switch to bank 2
	mapper4.WritePRG(0x8000, 0x00)
	mapper4.WritePRG(0x8001, 0x02)

	// Reset PPUADDR
	mem.Write(0x2006, 0x00)
	mem.Write(0x2006, 0x00)

	// Write different pattern to bank 2
	mem.Write(0x2007, 0x20)
	mem.Write(0x2007, 0x21)

	// Switch back to bank 0
	mapper4.WritePRG(0x8000, 0x00)
	mapper4.WritePRG(0x8001, 0x00)

	// Reset PPUADDR
	mem.Write(0x2006, 0x00)
	mem.Write(0x2006, 0x00)

	// Should still read original pattern from bank 0
	actual := mem.Read(0x2007)
	if actual != testPattern[0] {
		t.Errorf("Bank 0 data lost after bank switch: expected $%02X, got $%02X", testPattern[0], actual)
	}

	t.Logf("PPU integration test passed")
}
