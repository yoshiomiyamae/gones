package gui

import (
	"fmt"
	"os"
	"runtime"
	"time"
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/yoshiomiyamaegones/pkg/logger"
	"github.com/yoshiomiyamaegones/pkg/nes"
)

const (
	WindowWidth  = 256 * 3 // NES resolution 256x240 scaled 3x
	WindowHeight = 240 * 3
	WindowTitle  = "GoNES - Nintendo Entertainment System Emulator"

	// Audio constants
	AudioSampleRate = 44100
	AudioBufferSize = 1024             // Standard buffer size
	AudioChannels   = 1                // Mono
	AudioFormat     = sdl.AUDIO_F32LSB // 32-bit float, little-endian

	// Timing constants
	TargetFPS = 60.0988 // NES actual framerate
)

var (
	// NTSC NES frame rate: 60.0988 FPS (more precisely: 1789773 / 29780.5 = 60.0988139...)
	// Frame time = 1,000,000,000 / 60.0988139 = 16,639,266.85 ns
	FrameTime = time.Duration(16639267) * time.Nanosecond // 16.639267ms per frame
)

// NESGUI represents the GUI for the NES emulator
type NESGUI struct {
	window        *sdl.Window
	renderer      *sdl.Renderer
	texture       *sdl.Texture
	nes           *nes.NES
	running       bool
	screenshotNum int

	// Audio
	audioDevice sdl.AudioDeviceID
	audioSpec   *sdl.AudioSpec

	// Timing
	lastFrameTime time.Time
	nextFrameTime time.Time

	// FPS tracking
	fpsCounter int
	fpsTimer   time.Time
	currentFPS float64
	showFPS    bool
}

// NewNESGUI creates a new NES GUI
func NewNESGUI(nesSystem *nes.NES) (*NESGUI, error) {
	// Lock main thread for SDL
	runtime.LockOSThread()

	// Initialize SDL
	if err := sdl.Init(sdl.INIT_VIDEO | sdl.INIT_AUDIO); err != nil {
		return nil, err
	}

	// Create window
	window, err := sdl.CreateWindow(
		WindowTitle,
		sdl.WINDOWPOS_UNDEFINED,
		sdl.WINDOWPOS_UNDEFINED,
		WindowWidth,
		WindowHeight,
		sdl.WINDOW_SHOWN,
	)
	if err != nil {
		sdl.Quit()
		return nil, err
	}

	// Create renderer
	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		window.Destroy()
		sdl.Quit()
		return nil, err
	}

	// Set renderer blend mode to none (no color blending)
	renderer.SetDrawBlendMode(sdl.BLENDMODE_NONE)

	// Create texture for NES framebuffer (256x240 pixels, ABGR format)
	texture, err := renderer.CreateTexture(
		sdl.PIXELFORMAT_ABGR8888,
		sdl.TEXTUREACCESS_STREAMING,
		256,
		240,
	)
	if err != nil {
		renderer.Destroy()
		window.Destroy()
		sdl.Quit()
		return nil, err
	}

	// Set texture blend mode to none
	texture.SetBlendMode(sdl.BLENDMODE_NONE)

	// Initialize audio
	gui := &NESGUI{
		window:        window,
		renderer:      renderer,
		texture:       texture,
		nes:           nesSystem,
		running:       true,
		screenshotNum: 0,
		lastFrameTime: time.Now(),
		nextFrameTime: time.Now().Add(FrameTime),
		fpsTimer:      time.Now(),
		showFPS:       true,
	}

	// Setup audio device
	if err := gui.initAudio(); err != nil {
		logger.LogError("Failed to initialize audio: %v", err)
		logger.LogError("Audio will be disabled. Check SDL2 audio drivers.")
		// Continue without audio rather than failing completely
	} else {
		logger.LogInfo("Audio initialization successful")
	}

	return gui, nil
}

// Destroy cleans up SDL resources
func (g *NESGUI) Destroy() {
	// Close audio device
	if g.audioDevice != 0 {
		sdl.CloseAudioDevice(g.audioDevice)
	}

	if g.texture != nil {
		g.texture.Destroy()
	}
	if g.renderer != nil {
		g.renderer.Destroy()
	}
	if g.window != nil {
		g.window.Destroy()
	}
	sdl.Quit()
}

// Run starts the main GUI loop
func (g *NESGUI) Run() {
	frameCount := 0
	startTime := time.Now()
	
	for g.running {
		frameStart := time.Now()
		
		g.handleEvents()
		g.update()
		g.render()

		// Calculate target frame end time based on total elapsed time
		// This compensates for Sleep() inaccuracies
		frameCount++
		targetEndTime := startTime.Add(time.Duration(frameCount) * FrameTime)
		
		now := time.Now()
		if now.Before(targetEndTime) {
			time.Sleep(targetEndTime.Sub(now))
		}
		
		// Debug: Log frame timing every 60 frames
		if frameCount%60 == 0 {
			actualFrameTime := time.Since(frameStart)
			expectedFrameTime := FrameTime
			deviation := float64(actualFrameTime-expectedFrameTime) / float64(expectedFrameTime) * 100
			
			// Also check average frame rate
			avgFrameTime := time.Since(startTime) / time.Duration(frameCount)
			avgDeviation := float64(avgFrameTime-expectedFrameTime) / float64(expectedFrameTime) * 100
			
			if deviation > 5 || deviation < -5 || avgDeviation > 2 || avgDeviation < -2 {
				logger.LogInfo("Frame timing: actual=%.3fms, avg=%.3fms, expected=%.3fms, deviation=%.1f%%, avg_dev=%.1f%%",
					actualFrameTime.Seconds()*1000, avgFrameTime.Seconds()*1000, 
					expectedFrameTime.Seconds()*1000, deviation, avgDeviation)
			}
		}
		
		g.lastFrameTime = time.Now()
	}
}

// handleEvents processes SDL events
func (g *NESGUI) handleEvents() {
	for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
		switch e := event.(type) {
		case *sdl.QuitEvent:
			g.running = false
		case *sdl.KeyboardEvent:
			g.handleKeyboard(e)
		}
	}
}

// handleKeyboard maps keyboard input to NES controller
func (g *NESGUI) handleKeyboard(event *sdl.KeyboardEvent) {
	pressed := event.State == sdl.PRESSED

	// Get input interface from NES system
	input := g.nes.GetInput()

	switch event.Keysym.Sym {
	case sdl.K_z: // A button
		input.SetButton(0, 0, pressed) // Controller 1, A button
	case sdl.K_x: // B button
		input.SetButton(0, 1, pressed) // Controller 1, B button
	case sdl.K_a: // Select
		input.SetButton(0, 2, pressed) // Controller 1, Select
	case sdl.K_s: // Start
		input.SetButton(0, 3, pressed) // Controller 1, Start
	case sdl.K_UP:
		input.SetButton(0, 4, pressed) // Controller 1, Up
	case sdl.K_DOWN:
		input.SetButton(0, 5, pressed) // Controller 1, Down
	case sdl.K_LEFT:
		input.SetButton(0, 6, pressed) // Controller 1, Left
	case sdl.K_RIGHT:
		input.SetButton(0, 7, pressed) // Controller 1, Right
	case sdl.K_ESCAPE:
		g.running = false
	case sdl.K_F12:
		if pressed {
			g.saveScreenshot()
		}
	case sdl.K_F3:
		if pressed {
			g.showFPS = !g.showFPS
		}
	}
}

// update runs the NES emulation for one frame
func (g *NESGUI) update() {
	// Track APU cycles before frame
	apuCyclesBefore := g.nes.APU.Cycles
	
	// Run NES for one frame (approximately 29780 CPU cycles)
	g.nes.StepFrame()
	
	// Debug: Log APU cycles per frame every 60 frames (BEFORE queueAudio clears the buffer)
	if g.nes.Frame%60 == 0 && g.nes.Frame > 0 {
		apuCyclesThisFrame := g.nes.APU.Cycles - apuCyclesBefore
		samplesGenerated := len(g.nes.APU.Output)
		// Expected: ~29780 cycles/frame, ~732 samples/frame at 44100 Hz
		logger.LogInfo("Frame %d: APU cycles=%d (expected ~29780), samples=%d (expected ~732)", 
			g.nes.Frame, apuCyclesThisFrame, samplesGenerated)
	}

	// Queue audio samples
	g.queueAudio()

	// Update FPS counter
	g.updateFPS()
}

// render draws the current frame to the screen
func (g *NESGUI) render() {
	// Get current framebuffer (use same method as headless mode)
	framebuffer := g.nes.GetDisplayFramebuffer()

	frameNum := g.nes.GetFrame()

	// Debug: Count non-background pixels in GUI mode and check for colored pixels
	nonBgPixels := 0
	orangePixels := 0 // FFFF2200 converts to high R, low G, no B
	for i := 0; i < len(framebuffer); i += 4 {
		r := framebuffer[i]
		g := framebuffer[i+1]
		b := framebuffer[i+2]
		a := framebuffer[i+3]

		// Check for orange pixels (FFFF2200 -> R=255, G=34, B=0)
		if r > 200 && g > 20 && g < 50 && b < 10 {
			orangePixels++
		}

		// Check if pixel is not transparent black (0,0,0,0) or dark background (5,5,5,255)
		if !(r == 0 && g == 0 && b == 0 && a == 0) && !(r == 5 && g == 5 && b == 5 && a == 255) {
			nonBgPixels++
		}
	}

	// Use F1 key to toggle between test pattern and NES output
	keys := sdl.GetKeyboardState()
	useTestPattern := keys[sdl.SCANCODE_F1] != 0

	if useTestPattern {
		// Create test pattern: specific colors to debug endianness
		testPattern := make([]uint8, 256*240*4)
		for y := 0; y < 240; y++ {
			for x := 0; x < 256; x++ {
				idx := (y*256 + x) * 4
				switch x / 64 {
				case 0: // Pure black (should match NES background)
					testPattern[idx+0] = 5   // R
					testPattern[idx+1] = 5   // G
					testPattern[idx+2] = 5   // B
					testPattern[idx+3] = 255 // A
				case 1: // Send blue data to get red display
					testPattern[idx+0] = 0   // R
					testPattern[idx+1] = 0   // G
					testPattern[idx+2] = 255 // B
					testPattern[idx+3] = 255 // A
				case 2: // Pure green
					testPattern[idx+0] = 0   // R
					testPattern[idx+1] = 255 // G
					testPattern[idx+2] = 0   // B
					testPattern[idx+3] = 255 // A
				case 3: // Send red data to get blue display
					testPattern[idx+0] = 255 // R
					testPattern[idx+1] = 0   // G
					testPattern[idx+2] = 0   // B
					testPattern[idx+3] = 255 // A
				}
			}
		}
	} else {
		// Normal operation - but first check the actual NES framebuffer content
		if frameNum <= 20 || frameNum%60 == 0 { // Extended debugging to frame 20 + every 60 frames
			// Debug: Check if NES framebuffer has non-background colors
			backgroundPixel := uint32(0xFF050505) // Expected background color
			nonBgCount := 0
			for _, pixel := range g.nes.GetDisplayFramebufferRaw() {
				if pixel != backgroundPixel && pixel != 0 {
					nonBgCount++
				}
			}
		}

		g.texture.Update(nil, unsafe.Pointer(&framebuffer[0]), 256*4) // 4 bytes per pixel (RGBA)
	}

	// Clear renderer
	g.renderer.SetDrawColor(0, 0, 0, 255)
	g.renderer.Clear()

	// Copy texture to renderer (scaled to window size)
	g.renderer.Copy(g.texture, nil, nil)

	// Update window title with FPS if enabled
	if g.showFPS {
		g.updateWindowTitle()
	}

	// Present the rendered frame
	g.renderer.Present()
}

// saveScreenshot saves the current screen to a file
func (g *NESGUI) saveScreenshot() {
	filename := fmt.Sprintf("screenshot_%03d.png", g.screenshotNum)
	g.screenshotNum++
	g.saveScreenshotWithName(filename)
}

// saveFramebufferAsRaw saves framebuffer data as raw RGBA file
func (g *NESGUI) saveFramebufferAsRaw(filename string, data []uint8) {
	file, err := os.Create(filename)
	if err != nil {
		logger.LogError("Failed to create file %s: %v\n", filename, err)
		return
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		logger.LogError("Failed to write to file %s: %v\n", filename, err)
		return
	}

	logger.LogInfo("Raw framebuffer saved: %s (%d bytes)\n", filename, len(data))
}

// saveScreenshotWithName saves the current screen with a specific filename
func (g *NESGUI) saveScreenshotWithName(filename string) {
	// Read pixels from renderer
	w, h, _ := g.renderer.GetOutputSize()
	pixels := make([]byte, w*h*4)
	err := g.renderer.ReadPixels(nil, sdl.PIXELFORMAT_RGBA8888, unsafe.Pointer(&pixels[0]), int(w*4))
	if err != nil {
		logger.LogError("Failed to read pixels: %v\n", err)
		return
	}

	// Save as raw RGBA file
	g.saveFramebufferAsRaw(filename, pixels)
}

// initAudio initializes SDL audio device and callback
func (g *NESGUI) initAudio() error {
	// List available audio drivers for debugging
	numDrivers := sdl.GetNumAudioDrivers()
	logger.LogInfo("Available audio drivers (%d):", numDrivers)
	for i := 0; i < numDrivers; i++ {
		driverName := sdl.GetAudioDriver(i)
		logger.LogInfo("  %d: %s", i, driverName)
	}

	currentDriver := sdl.GetCurrentAudioDriver()
	logger.LogInfo("Current audio driver: %s", currentDriver)

	// Define audio specification with callback
	want := &sdl.AudioSpec{
		Freq:     AudioSampleRate,
		Format:   AudioFormat,
		Channels: AudioChannels,
		Samples:  AudioBufferSize,
	}

	logger.LogInfo("Requesting audio format: %dHz, %d channels, format 0x%x, buffer %d",
		want.Freq, want.Channels, want.Format, want.Samples)

	// Open audio device
	var have sdl.AudioSpec
	device, err := sdl.OpenAudioDevice("", false, want, &have, sdl.AUDIO_ALLOW_ANY_CHANGE)
	if err != nil {
		// Try with 16-bit format for better Windows compatibility
		logger.LogInfo("Retrying with 16-bit audio format...")
		want.Format = sdl.AUDIO_S16LSB
		device, err = sdl.OpenAudioDevice("", false, want, &have, sdl.AUDIO_ALLOW_ANY_CHANGE)
		if err != nil {
			return fmt.Errorf("failed to open audio device: %v", err)
		}
	}

	g.audioDevice = device
	g.audioSpec = &have

	logger.LogInfo("Audio initialized: %dHz, %d channels, format 0x%x, buffer size %d",
		have.Freq, have.Channels, have.Format, have.Samples)
	
	// IMPORTANT: Check if actual sample rate differs from requested
	if have.Freq != AudioSampleRate {
		logger.LogInfo("WARNING: Requested %d Hz but got %d Hz - audio pitch will be wrong!", 
			AudioSampleRate, have.Freq)
	}

	// Start audio playback
	sdl.PauseAudioDevice(device, false)

	return nil
}

// queueAudio queues APU audio samples to SDL - simple approach
func (g *NESGUI) queueAudio() {
	if g.audioDevice == 0 {
		return
	}

	// Get audio output from APU
	apuOutput := g.nes.APU.Output
	if len(apuOutput) == 0 {
		return
	}

	// Simple buffering - don't overthink it
	queuedBytes := sdl.GetQueuedAudioSize(g.audioDevice)
	maxBytes := uint32(AudioBufferSize * 4 * 2) // 2 buffers worth

	if queuedBytes < maxBytes {
		var audioData []byte

		// Convert samples based on the actual audio format
		if g.audioSpec.Format == sdl.AUDIO_F32LSB {
			// 32-bit float format
			audioData = make([]byte, len(apuOutput)*4)
			for i, sample := range apuOutput {
				// Boost volume for better audibility
				sample *= 0.5
				bits := *(*uint32)(unsafe.Pointer(&sample))
				audioData[i*4+0] = byte(bits)
				audioData[i*4+1] = byte(bits >> 8)
				audioData[i*4+2] = byte(bits >> 16)
				audioData[i*4+3] = byte(bits >> 24)
			}
		} else if g.audioSpec.Format == sdl.AUDIO_S16LSB {
			// 16-bit signed integer format (better Windows compatibility)
			audioData = make([]byte, len(apuOutput)*2)
			for i, sample := range apuOutput {
				// Boost volume and convert to 16-bit signed
				sample *= 0.5
				if sample > 1.0 {
					sample = 1.0
				} else if sample < -1.0 {
					sample = -1.0
				}
				intSample := int16(sample * 32767)
				audioData[i*2+0] = byte(intSample)
				audioData[i*2+1] = byte(intSample >> 8)
			}
		}

		// Queue audio data
		if len(audioData) > 0 {
			sdl.QueueAudio(g.audioDevice, audioData)
		}
	}

	// Always clear APU buffer
	g.nes.APU.Output = g.nes.APU.Output[:0]
}

// updateFPS calculates the current FPS
func (g *NESGUI) updateFPS() {
	g.fpsCounter++

	// Update FPS every 0.5 seconds for more responsive display
	elapsed := time.Since(g.fpsTimer)
	if elapsed >= 500*time.Millisecond {
		g.currentFPS = float64(g.fpsCounter) / elapsed.Seconds()
		
		// Debug: Log if FPS is significantly off target
		if g.fpsCounter%30 == 0 {
			deviation := (g.currentFPS - TargetFPS) / TargetFPS * 100
			if deviation > 5 || deviation < -5 {
				logger.LogInfo("FPS: %.2f (target: %.2f, deviation: %.1f%%)", 
					g.currentFPS, TargetFPS, deviation)
			}
		}
		
		g.fpsCounter = 0
		g.fpsTimer = time.Now()
	}
}

// updateWindowTitle updates the window title with FPS information
func (g *NESGUI) updateWindowTitle() {
	title := fmt.Sprintf("%s - FPS: %.1f", WindowTitle, g.currentFPS)
	g.window.SetTitle(title)
}
