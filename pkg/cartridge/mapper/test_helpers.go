package mapper

// Test data for various mapper tests
var (
	// 16KB PRG ROM data (NROM-128)
	testPRGROM16KB = make([]uint8, 16*1024)
	// 32KB PRG ROM data (NROM-256)
	testPRGROM32KB = make([]uint8, 32*1024)
	// 8KB CHR ROM data
	testCHRROM8KB = make([]uint8, 8*1024)
	// 32KB CHR ROM data for CNROM
	testCHRROM32KB = make([]uint8, 32*1024)
)

func init() {
	// Initialize test PRG ROM with pattern for testing
	for i := range testPRGROM16KB {
		testPRGROM16KB[i] = uint8(i & 0xFF)
	}
	for i := range testPRGROM32KB {
		testPRGROM32KB[i] = uint8(i & 0xFF)
	}
	
	// Initialize test CHR ROM with pattern
	for i := range testCHRROM8KB {
		testCHRROM8KB[i] = uint8(i & 0xFF)
	}
	for i := range testCHRROM32KB {
		testCHRROM32KB[i] = uint8(i & 0xFF)
	}
	
	// Set up reset vector at $FFFC-$FFFD for PRG ROM
	if len(testPRGROM16KB) >= 0x4000 {
		testPRGROM16KB[0x3FFC] = 0x00 // Reset vector low
		testPRGROM16KB[0x3FFD] = 0x80 // Reset vector high ($8000)
	}
	if len(testPRGROM32KB) >= 0x8000 {
		testPRGROM32KB[0x7FFC] = 0x00 // Reset vector low
		testPRGROM32KB[0x7FFD] = 0x80 // Reset vector high ($8000)
	}
}