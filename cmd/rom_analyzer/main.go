package main

import (
	"fmt"
	"log"
	"os"

	"github.com/yoshiomiyamaegones/pkg/cartridge"
	"github.com/yoshiomiyamaegones/pkg/cartridge/mapper"
	"github.com/yoshiomiyamaegones/pkg/logger"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: rom_analyzer <rom_file>")
		os.Exit(1)
	}

	romFile := os.Args[1]

	// Load cartridge
	file, err := os.Open(romFile)
	if err != nil {
		log.Fatalf("Failed to open ROM file: %v", err)
	}
	defer file.Close()

	cart, err := cartridge.LoadFromReader(file)
	if err != nil {
		log.Fatalf("Failed to load ROM: %v", err)
	}

	// Display ROM information
	logger.LogInfo("=== ROM Analysis ===\n")
	logger.LogInfo("File: %s\n", romFile)
	logger.LogInfo("\n=== Header Information ===\n")
	logger.LogInfo("Magic: %s (0x%02X%02X%02X%02X)\n",
		string(cart.Header.Magic[:]), cart.Header.Magic[0], cart.Header.Magic[1], cart.Header.Magic[2], cart.Header.Magic[3])
	logger.LogInfo("PRG ROM Size: %d units (%d KB)\n", cart.Header.PRGROMSize, int(cart.Header.PRGROMSize)*16)
	logger.LogInfo("CHR ROM Size: %d units (%d KB)\n", cart.Header.CHRROMSize, int(cart.Header.CHRROMSize)*8)
	logger.LogInfo("Flags6: 0x%02X\n", cart.Header.Flags6)
	logger.LogInfo("Flags7: 0x%02X\n", cart.Header.Flags7)
	logger.LogInfo("Flags8: 0x%02X\n", cart.Header.Flags8)
	logger.LogInfo("Flags9: 0x%02X\n", cart.Header.Flags9)
	logger.LogInfo("Flags10: 0x%02X\n", cart.Header.Flags10)

	// Mapper information
	mapperNumber := (cart.Header.Flags6 >> 4) | (cart.Header.Flags7 & 0xF0)
	logger.LogInfo("\n=== Mapper Information ===\n")
	logger.LogInfo("Mapper Number: %d\n", mapperNumber)

	// Additional flags
	logger.LogInfo("\n=== ROM Configuration ===\n")
	logger.LogInfo("Trainer Present: %v\n", cart.Header.Flags6&0x04 != 0)
	logger.LogInfo("Battery Backed: %v\n", cart.Header.Flags6&0x02 != 0)
	logger.LogInfo("Four Screen VRAM: %v\n", cart.Header.Flags6&0x08 != 0)

	if cart.Header.Flags6&0x08 != 0 {
		logger.LogInfo("Mirroring: Four Screen\n")
	} else if cart.Header.Flags6&0x01 != 0 {
		logger.LogInfo("Mirroring: Vertical\n")
	} else {
		logger.LogInfo("Mirroring: Horizontal\n")
	}

	// Memory sizes
	logger.LogInfo("\n=== Memory Configuration ===\n")
	logger.LogInfo("PRG ROM: %d bytes (0x%04X)\n", len(cart.PRGROM), len(cart.PRGROM))
	if len(cart.CHRROM) > 0 {
		logger.LogInfo("CHR ROM: %d bytes (0x%04X)\n", len(cart.CHRROM), len(cart.CHRROM))
	}
	if len(cart.CHRRAM) > 0 {
		logger.LogInfo("CHR RAM: %d bytes (0x%04X)\n", len(cart.CHRRAM), len(cart.CHRRAM))
	}
	if len(cart.PRGRAM) > 0 {
		logger.LogInfo("PRG RAM: %d bytes (0x%04X)\n", len(cart.PRGRAM), len(cart.PRGRAM))
	}

	// Mapper specific information
	if mapperNumber == 4 {
		logger.LogInfo("\n=== MMC3 (Mapper 4) Specific Information ===\n")
		if mapper4, ok := cart.Mapper.(*mapper.Mapper4); ok {
			// Get current bank configuration
			banks := mapper4.GetCurrentPRGBanks()
			logger.LogInfo("Initial PRG Bank Configuration:\n")
			logger.LogInfo("  $8000-$9FFF: Bank %d\n", banks[0])
			logger.LogInfo("  $A000-$BFFF: Bank %d\n", banks[1])
			logger.LogInfo("  $C000-$DFFF: Bank %d (fixed)\n", banks[2])
			logger.LogInfo("  $E000-$FFFF: Bank %d (fixed)\n", banks[3])

			logger.LogInfo("Bank Counts:\n")
			prgBankCount := len(cart.PRGROM) / 8192
			logger.LogInfo("  PRG Banks (8KB each): %d\n", prgBankCount)

			if len(cart.CHRROM) > 0 {
				chrBankCount := len(cart.CHRROM) / 1024
				logger.LogInfo("  CHR Banks (1KB each): %d\n", chrBankCount)
			} else {
				chrBankCount := len(cart.CHRRAM) / 1024
				logger.LogInfo("  CHR RAM Banks (1KB each): %d\n", chrBankCount)
			}
		}
	}

	logger.LogInfo("\n=== Raw Header Dump ===\n")
	logger.LogInfo("00 01 02 03 04 05 06 07 08 09 0A 0B 0C 0D 0E 0F\n")
	headerBytes := []uint8{
		cart.Header.Magic[0], cart.Header.Magic[1], cart.Header.Magic[2], cart.Header.Magic[3],
		cart.Header.PRGROMSize, cart.Header.CHRROMSize, cart.Header.Flags6, cart.Header.Flags7,
		cart.Header.Flags8, cart.Header.Flags9, cart.Header.Flags10,
		cart.Header.Padding[0], cart.Header.Padding[1], cart.Header.Padding[2], cart.Header.Padding[3], cart.Header.Padding[4],
	}
	for i, b := range headerBytes {
		logger.LogInfo("%02X ", b)
		if (i+1)%16 == 0 {
			logger.LogInfo("\n")
		}
	}
	if len(headerBytes)%16 != 0 {
		logger.LogInfo("\n")
	}
}
