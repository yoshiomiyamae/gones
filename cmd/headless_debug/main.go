package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/yoshiomiyamaegones/pkg/cartridge"
	"github.com/yoshiomiyamaegones/pkg/cartridge/mapper"
	"github.com/yoshiomiyamaegones/pkg/logger"
	"github.com/yoshiomiyamaegones/pkg/nes"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: headless_debug <rom_file> [frames]")
		os.Exit(1)
	}

	romFile := os.Args[1]
	maxFrames := 10
	if len(os.Args) >= 3 {
		fmt.Sscanf(os.Args[2], "%d", &maxFrames)
	}

	// Initialize logger
	err := logger.Initialize(logger.LogLevelDebug, "")
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Close()

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

	mapperNumber := (cart.Header.Flags6 >> 4) | (cart.Header.Flags7 & 0xF0)
	logger.LogInfo("=== Headless Debug Mode ===\n")
	logger.LogInfo("ROM: %s\n", romFile)
	logger.LogInfo("Mapper: %d\n", mapperNumber)
	logger.LogInfo("Max frames to run: %d\n", maxFrames)
	logger.LogInfo("\n")

	// Create NES system
	nesSystem := nes.NewNES()
	nesSystem.LoadCartridge(cart)
	nesSystem.Reset()

	logger.LogInfo("=== Initial State ===\n")
	logger.LogInfo("Frame: %d\n", nesSystem.GetFrame())
	logger.LogInfo("Cycles: %d\n", nesSystem.Cycles)

	// Print initial mapper state if it's Mapper 4
	if mapperNumber == 4 {
		printMapper4State(cart.Mapper, 0)
	}

	logger.LogInfo("\n=== Starting Emulation ===\n")
	startTime := time.Now()

	// Run for specified number of frames
	for i := 0; i < maxFrames; i++ {
		frameStart := time.Now()

		// Don't press START button - just let title screen display
		// if i == 5 {
		//	logger.LogInfo("Pressing START button to begin test\n")
		//	nesSystem.Input.SetButton(0, 3, true) // START button (player 1, button 3)
		// }
		// if i == 6 {
		//	nesSystem.Input.SetButton(0, 3, false)
		// }

		nesSystem.StepFrame()

		frameTime := time.Since(frameStart)

		logger.LogInfo("Frame %d completed in %v\n", nesSystem.GetFrame(), frameTime)
		logger.LogInfo("  Total cycles: %d\n", nesSystem.Cycles)

		// Print PPU register state for first frame
		if i == 0 {
			printPPUState(nesSystem)
		}

		// Print mapper state every few frames for Mapper 4
		if mapperNumber == 4 && (i+1)%3 == 0 {
			printMapper4State(cart.Mapper, nesSystem.GetFrame())
		}

		// Check framebuffer content
		framebuffer := nesSystem.GetFramebuffer()
		nonZeroPixels := 0
		pixelStats := make(map[uint8]int)
		for j := 0; j < len(framebuffer); j++ {
			pixelStats[framebuffer[j]]++
			if framebuffer[j] != 0 {
				nonZeroPixels++
			}
		}
		logger.LogInfo("  Non-zero pixels in framebuffer: %d\n", nonZeroPixels)

		// Print pixel distribution for first frame to see if there's any variation
		if i == 0 {
			logger.LogInfo("  Pixel value distribution: ")
			for value, count := range pixelStats {
				if count > 0 {
					logger.LogInfo("0x%02X:%d ", value, count)
				}
			}
			logger.LogInfo("\n")
		}

		// Save framebuffer for the last frame
		if i == maxFrames-1 {
			logger.LogInfo("  Saving final framebuffer...\n")
			saveFramebuffer(framebuffer, fmt.Sprintf("debug_frame_%d.raw", nesSystem.GetFrame()))
		}

		logger.LogInfo("\n")
	}

	totalTime := time.Since(startTime)
	logger.LogInfo("=== Final Results ===\n")
	logger.LogInfo("Completed %d frames in %v\n", nesSystem.GetFrame(), totalTime)
	logger.LogInfo("Average frame time: %v\n", totalTime/time.Duration(maxFrames))
	logger.LogInfo("Final cycle count: %d\n", nesSystem.Cycles)

	// Final mapper state
	if mapperNumber == 4 {
		logger.LogInfo("\n=== Final Mapper 4 State ===\n")
		printMapper4State(cart.Mapper, nesSystem.GetFrame())
	}
}

func printMapper4State(m mapper.Mapper, frame uint64) {
	if mapper4, ok := m.(*mapper.Mapper4); ok {
		logger.LogInfo("--- Mapper 4 State (Frame %d) ---\n", frame)
		banks := mapper4.GetCurrentPRGBanks()
		logger.LogInfo("  PRG Banks: [%d, %d, %d, %d] ($8000, $A000, $C000, $E000)\n",
			banks[0], banks[1], banks[2], banks[3])

		// Get detailed debug info
		debugInfo := mapper4.GetDebugInfo()
		logger.LogInfo("  Bank Select: 0x%02X\n", debugInfo["bankSelect"])
		bankRegs := debugInfo["bankRegisters"].([8]uint8)
		logger.LogInfo("  Bank Registers: [R0=%d, R1=%d, R2=%d, R3=%d, R4=%d, R5=%d, R6=%d, R7=%d]\n",
			bankRegs[0], bankRegs[1], bankRegs[2], bankRegs[3],
			bankRegs[4], bankRegs[5], bankRegs[6], bankRegs[7])
		logger.LogInfo("  PRG Mode: %d, CHR Mode: %d\n",
			debugInfo["prgMode"], debugInfo["chrMode"])
		logger.LogInfo("  Mirroring: %d (0=Vertical, 1=Horizontal)\n", debugInfo["mirroringMode"])
		logger.LogInfo("  PRG RAM Protect: 0x%02X\n", debugInfo["prgRAMProtect"])
		logger.LogInfo("  IRQ: Counter=%d, Reload=%d, Enabled=%v, Pending=%v\n",
			debugInfo["irqCounter"], debugInfo["irqReloadValue"],
			debugInfo["irqEnabled"], debugInfo["irqPending"])
		logger.LogInfo("  Bank Counts: PRG=%d (8KB), CHR=%d (1KB)\n",
			debugInfo["prgBankCount"], debugInfo["chrBankCount"])
	}
}

func printPPUState(nesSystem *nes.NES) {
	logger.LogInfo("  PPU State:\n")
	logger.LogInfo("    Frame: %d, Scanline: %d, Cycle: %d\n",
		nesSystem.PPU.Frame, nesSystem.PPU.Scanline, nesSystem.PPU.Cycle)
	logger.LogInfo("    PPUCTRL: 0x%02X, PPUMASK: 0x%02X, PPUSTATUS: 0x%02X\n",
		nesSystem.PPU.PPUCTRL, nesSystem.PPU.PPUMASK, nesSystem.PPU.PPUSTATUS)

	// Check if rendering is enabled
	bgEnabled := nesSystem.PPU.PPUMASK&0x08 != 0
	spriteEnabled := nesSystem.PPU.PPUMASK&0x10 != 0
	logger.LogInfo("    Rendering: BG=%v, Sprites=%v\n", bgEnabled, spriteEnabled)

	// Check NMI settings
	nmiEnabled := nesSystem.PPU.PPUCTRL&0x80 != 0
	logger.LogInfo("    NMI Enabled: %v, NMI Requested: %v\n", nmiEnabled, nesSystem.PPU.NMIRequested)
}

func saveFramebuffer(framebuffer []uint8, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		logger.LogError("Error creating framebuffer file: %v\n", err)
		return
	}
	defer file.Close()

	_, err = file.Write(framebuffer)
	if err != nil {
		logger.LogError("Error writing framebuffer: %v\n", err)
		return
	}

	logger.LogInfo("  Framebuffer saved to %s (%d bytes)\n", filename, len(framebuffer))
}
