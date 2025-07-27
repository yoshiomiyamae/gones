package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/yoshiomiyamaegones/pkg/cartridge"
	"github.com/yoshiomiyamaegones/pkg/gui"
	"github.com/yoshiomiyamaegones/pkg/logger"
	"github.com/yoshiomiyamaegones/pkg/nes"
)

// Global debug flag
var DebugMode bool

func main() {
	// Define command line flags
	var (
		logLevel   = flag.String("log-level", "info", "Log level (off, error, warn, info, debug, trace)")
		logFile    = flag.String("log-file", "", "Log file path (empty for stdout)")
		cpuLog     = flag.Bool("cpu-log", false, "Enable CPU instruction logging")
		ppuLog     = flag.Bool("ppu-log", false, "Enable PPU logging")
		apuLog     = flag.Bool("apu-log", false, "Enable APU logging")
		mapperLog  = flag.Bool("mapper-log", false, "Enable mapper logging")
		headless   = flag.Bool("headless", false, "Run in headless mode for testing")
		testFrames = flag.Int("test-frames", 600, "Number of frames to run in headless mode")
		debugMode  = flag.Bool("debug", false, "Enable extra debug output (reduces performance)")
	)

	flag.Usage = func() {
		fmt.Printf("Usage: %s [options] <rom_file>\n\n", os.Args[0])
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println("\nControls:")
		fmt.Println("  Z - A button")
		fmt.Println("  X - B button")
		fmt.Println("  A - Select")
		fmt.Println("  S - Start")
		fmt.Println("  Arrow keys - D-pad")
		fmt.Println("  ESC - Quit")
	}

	flag.Parse()

	// Check if ROM file is provided
	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	romFile := flag.Arg(0)

	// Initialize logger
	level := logger.GetLogLevelFromString(*logLevel)
	err := logger.Initialize(level, *logFile)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Close()

	// Configure component logging
	logger.SetCPULogging(*cpuLog)
	logger.SetPPULogging(*ppuLog)
	logger.SetAPULogging(*apuLog)
	logger.SetMapperLogging(*mapperLog)

	// Set global debug mode
	DebugMode = *debugMode

	logger.LogInfo("GoNES Emulator starting...")
	logger.LogInfo("Log level: %s", *logLevel)
	if *logFile != "" {
		logger.LogInfo("Logging to file: %s", *logFile)
	}

	// Check if file exists
	if _, err := os.Stat(romFile); os.IsNotExist(err) {
		log.Fatalf("ROM file not found: %s", romFile)
	}

	// Load cartridge
	file, err := os.Open(romFile)
	if err != nil {
		log.Fatalf("Failed to open ROM file: %v", err)
	}
	defer file.Close()

	cart, err := cartridge.LoadFromReader(file)
	if err != nil {
		logger.LogError("Failed to load ROM: %v", err)
		log.Fatalf("Failed to load ROM: %v", err)
	}

	mapperNumber := (cart.Header.Flags6 >> 4) | (cart.Header.Flags7 & 0xF0)

	logger.LogInfo("Loaded ROM: %s", filepath.Base(romFile))
	logger.LogInfo("Mapper: %d", mapperNumber)
	logger.LogInfo("PRG ROM: %d KB", len(cart.PRGROM)/1024)
	if len(cart.CHRROM) > 0 {
		logger.LogInfo("CHR ROM: %d KB", len(cart.CHRROM)/1024)
	} else {
		logger.LogInfo("CHR RAM: %d KB", len(cart.CHRRAM)/1024)
	}

	// Create NES system
	logger.LogInfo("Creating NES system...")
	nesSystem := nes.NewNES()
	nesSystem.LoadCartridge(cart)
	nesSystem.Reset()
	logger.LogInfo("NES system initialized")

	if *headless {
		// Run in headless mode
		runHeadless(nesSystem, *testFrames)
	} else {
		// Create and run GUI
		logger.LogInfo("Creating GUI...")
		nesGUI, err := gui.NewNESGUI(nesSystem)
		if err != nil {
			logger.LogError("Failed to create GUI: %v", err)
			log.Fatalf("Failed to create GUI: %v", err)
		}
		defer nesGUI.Destroy()

		logger.LogInfo("Starting emulator...")
		// Run the emulator
		nesGUI.Run()
		logger.LogInfo("Emulator stopped")
	}
}

func runHeadless(nesSystem *nes.NES, maxFrames int) {
	logger.LogInfo("Starting headless mode for %d frames", maxFrames)

	startTime := time.Now()

	for frame := 0; frame < maxFrames; frame++ {
		// Run one frame
		nesSystem.StepFrame()
	}

	elapsed := time.Since(startTime)
	logger.LogInfo("Headless execution completed in %v", elapsed)

	// Final frame analysis
	frameBuffer := nesSystem.GetDisplayFramebufferRaw()
	analyzeFrameBuffer(frameBuffer, maxFrames-1)
}

func saveFrameBuffer(frameBuffer []uint32, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		logger.LogError("Error creating file %s: %v", filename, err)
		return
	}
	defer file.Close()

	// Convert uint32 to bytes and write
	for _, pixel := range frameBuffer {
		file.Write([]byte{
			byte(pixel >> 24), // A
			byte(pixel >> 16), // R
			byte(pixel >> 8),  // G
			byte(pixel),       // B
		})
	}

	logger.LogInfo("Frame buffer saved: %s (%d bytes)", filename, len(frameBuffer)*4)
}

func analyzeFrameBuffer(frameBuffer []uint32, frame int) {
	pixelCounts := make(map[uint32]int)
	totalPixels := len(frameBuffer)

	// Count unique pixel values
	for _, pixel := range frameBuffer {
		pixelCounts[pixel]++
	}

	logger.LogInfo("Frame %d analysis:", frame)
	logger.LogInfo("  Total pixels: %d", totalPixels)
	logger.LogInfo("  Unique colors: %d", len(pixelCounts))

	// Show most common colors
	for color, count := range pixelCounts {
		percentage := float64(count) / float64(totalPixels) * 100
		if percentage > 1.0 { // Only show colors that make up >1% of the image
			logger.LogInfo("  Color 0x%08X: %d pixels (%.1f%%)", color, count, percentage)
		}
	}

	// Check for non-background pixels
	nonBgCount := 0
	for color, count := range pixelCounts {
		if color != 0xFF050505 { // Not the typical background color
			nonBgCount += count
		}
	}

	if nonBgCount > 0 {
		logger.LogInfo("  Non-background pixels: %d (%.1f%%)",
			nonBgCount, float64(nonBgCount)/float64(totalPixels)*100)
	} else {
		logger.LogInfo("  All pixels are background color")
	}
}

func countNonBackgroundPixels(frameBuffer []uint32) int {
	count := 0
	bgColor := uint32(0xFF050505)    // Typical background color
	blackColor := uint32(0xFF000000) // Black color
	zeroColor := uint32(0x00000000)  // Uninitialized

	for _, pixel := range frameBuffer {
		// Count as meaningful if it's not background, black, or zero
		if pixel != bgColor && pixel != blackColor && pixel != zeroColor {
			count++
		}
	}
	return count
}
