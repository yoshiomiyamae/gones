package test

import (
	"bytes"
	"testing"

	"github.com/yoshiomiyamaegones/pkg/cartridge"
)

// TestCartridgeLoader tests the cartridge loading functionality
func TestCartridgeLoader(t *testing.T) {
	// Create a minimal valid iNES ROM
	rom := createMinimalROM()

	// Load cartridge
	reader := bytes.NewReader(rom)
	cart, err := cartridge.LoadFromReader(reader)
	if err != nil {
		t.Fatalf("Failed to load test ROM: %v", err)
	}

	// Verify header was parsed correctly
	if cart.Header.PRGROMSize != 1 {
		t.Errorf("Expected PRG ROM size = 1, got %d", cart.Header.PRGROMSize)
	}

	if cart.Header.CHRROMSize != 1 {
		t.Errorf("Expected CHR ROM size = 1, got %d", cart.Header.CHRROMSize)
	}

	// Verify ROM data
	if len(cart.PRGROM) != 16384 {
		t.Errorf("Expected PRG ROM length = 16384, got %d", len(cart.PRGROM))
	}

	if len(cart.CHRROM) != 8192 {
		t.Errorf("Expected CHR ROM length = 8192, got %d", len(cart.CHRROM))
	}

	// Test mapper functionality
	if cart.Mapper == nil {
		t.Fatal("Mapper should not be nil")
	}

	// Test reading from PRG ROM
	value := cart.ReadPRG(0x8000)
	if value != 0x42 {
		t.Errorf("Expected first PRG byte = 0x42, got 0x%02X", value)
	}

	// Test reading from CHR ROM
	value = cart.ReadCHR(0x0000)
	if value != 0x55 {
		t.Errorf("Expected first CHR byte = 0x55, got 0x%02X", value)
	}
}

// TestInvalidROM tests loading invalid ROM data
func TestInvalidROM(t *testing.T) {
	// Test invalid magic number
	invalidROM := []byte{0x4E, 0x45, 0x53, 0x00} // "NES\x00" instead of "NES\x1A"
	reader := bytes.NewReader(invalidROM)

	_, err := cartridge.LoadFromReader(reader)
	if err == nil {
		t.Error("Expected error for invalid magic number")
	}

	// Test truncated ROM
	truncatedROM := []byte{0x4E, 0x45, 0x53, 0x1A, 0x01} // Too short
	reader = bytes.NewReader(truncatedROM)

	_, err = cartridge.LoadFromReader(reader)
	if err == nil {
		t.Error("Expected error for truncated ROM")
	}
}

// createMinimalROM creates a minimal valid iNES ROM for testing
func createMinimalROM() []byte {
	rom := make([]byte, 0)

	// iNES header (16 bytes)
	header := []byte{
		0x4E, 0x45, 0x53, 0x1A, // "NES\x1A"
		0x01,                                           // 1 x 16KB PRG ROM
		0x01,                                           // 1 x 8KB CHR ROM
		0x00,                                           // Flags 6: Horizontal mirroring, Mapper 0
		0x00,                                           // Flags 7: Mapper 0
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Padding
	}
	rom = append(rom, header...)

	// PRG ROM (16KB)
	prgROM := make([]byte, 16384)
	prgROM[0] = 0x42 // Test value
	// Fill reset vector (at end of 16KB block)
	prgROM[0x3FFC] = 0x00 // Reset vector low
	prgROM[0x3FFD] = 0x80 // Reset vector high
	rom = append(rom, prgROM...)

	// CHR ROM (8KB)
	chrROM := make([]byte, 8192)
	chrROM[0] = 0x55 // Test value
	rom = append(rom, chrROM...)

	return rom
}

// TestMapperSelection tests mapper selection logic
func TestMapperSelection(t *testing.T) {
	// Test different mapper numbers
	testCases := []struct {
		flags6     uint8
		flags7     uint8
		mapperNum  uint8
		shouldFail bool
	}{
		{0x00, 0x00, 0, false}, // Mapper 0
		{0x10, 0x00, 1, false}, // Mapper 1
		{0x20, 0x00, 2, false}, // Mapper 2
		{0x30, 0x00, 3, false}, // Mapper 3
		{0x40, 0x00, 4, false}, // Mapper 4
		{0x50, 0x00, 5, true},  // Mapper 5 (unsupported)
	}

	for _, tc := range testCases {
		rom := createMinimalROM()
		// Modify mapper flags
		rom[6] = tc.flags6
		rom[7] = tc.flags7

		reader := bytes.NewReader(rom)
		cart, err := cartridge.LoadFromReader(reader)

		if tc.shouldFail {
			if err == nil {
				t.Errorf("Expected error for unsupported mapper %d", tc.mapperNum)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for mapper %d: %v", tc.mapperNum, err)
			}
			if cart == nil {
				t.Errorf("Cart should not be nil for mapper %d", tc.mapperNum)
			}
		}
	}
}

// TestMirroringModes tests mirroring mode detection
func TestMirroringModes(t *testing.T) {
	testCases := []struct {
		flags6    uint8
		mirroring cartridge.MirroringMode
	}{
		{0x00, cartridge.MirroringHorizontal}, // Bit 0 clear
		{0x01, cartridge.MirroringVertical},   // Bit 0 set
		{0x08, cartridge.MirroringFourScreen}, // Bit 3 set (four-screen)
	}

	for _, tc := range testCases {
		rom := createMinimalROM()
		rom[6] = tc.flags6

		reader := bytes.NewReader(rom)
		cart, err := cartridge.LoadFromReader(reader)
		if err != nil {
			t.Fatalf("Failed to load ROM: %v", err)
		}

		if cart.Mirroring != tc.mirroring {
			t.Errorf("Expected mirroring %d, got %d", tc.mirroring, cart.Mirroring)
		}
	}
}
