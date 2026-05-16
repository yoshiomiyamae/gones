package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/yoshiomiyamaegones/pkg/cartridge"
	"github.com/yoshiomiyamaegones/pkg/gui"
	"github.com/yoshiomiyamaegones/pkg/logger"
	"github.com/yoshiomiyamaegones/pkg/nes"
)

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
		cpuProfile = flag.String("cpuprofile", "", "Write CPU profile to file (use with -headless for clean run)")
		memProfile = flag.String("memprofile", "", "Write heap profile to file at exit")
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
		fmt.Println("  Tab - Toggle turbo (fast-forward)")
		fmt.Println("  Ctrl+R - Reset NES")
		fmt.Println("  F1-F10 - Save state to slot 1-10")
		fmt.Println("  Ctrl+F1-F10 - Load state from slot 1-10")
		fmt.Println("  F11 - Toggle FPS display")
		fmt.Println("  F12 - Save screenshot")
		fmt.Println("  ESC - Quit")
	}

	flag.Parse()

	// Check if ROM file is provided
	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	romFile := flag.Arg(0)

	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			log.Fatalf("create cpu profile: %v", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatalf("start cpu profile: %v", err)
		}
		defer pprof.StopCPUProfile()
	}
	if *memProfile != "" {
		defer func() {
			f, err := os.Create(*memProfile)
			if err != nil {
				logger.LogError("create mem profile: %v", err)
				return
			}
			defer f.Close()
			runtime.GC()
			if err := pprof.WriteHeapProfile(f); err != nil {
				logger.LogError("write mem profile: %v", err)
			}
		}()
	}

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

	// Battery-backed PRG RAM: load <rom>.sav if it exists, persist on exit.
	savePath := nes.CompanionFile(romFile, ".sav")
	if cart.HasBattery() {
		loadBatterySave(cart, savePath)
		defer saveBatterySave(cart, savePath)
	}

	if *headless {
		// Run in headless mode
		runHeadless(nesSystem, *testFrames)
	} else {
		// Create and run GUI
		logger.LogInfo("Creating GUI...")
		nesGUI, err := gui.NewNESGUI(nesSystem, romFile)
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

func loadBatterySave(cart *cartridge.Cartridge, path string) {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		logger.LogInfo("No save file at %s (fresh save)", path)
		return
	}
	if err != nil {
		logger.LogError("Failed to open save file %s: %v", path, err)
		return
	}
	defer f.Close()
	if err := cart.LoadRAM(f); err != nil {
		logger.LogError("Failed to read save file %s: %v", path, err)
		return
	}
	logger.LogInfo("Loaded save file: %s", path)
}

func saveBatterySave(cart *cartridge.Cartridge, path string) {
	f, err := os.Create(path)
	if err != nil {
		logger.LogError("Failed to create save file %s: %v", path, err)
		return
	}
	defer f.Close()
	if err := cart.SaveRAM(f); err != nil {
		logger.LogError("Failed to write save file %s: %v", path, err)
		return
	}
	logger.LogInfo("Wrote save file: %s", path)
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
