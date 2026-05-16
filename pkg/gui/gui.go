// Package gui implements the SDL2-backed front-end for the NES emulator.
//
// The main type is NESGUI, which owns the SDL window/renderer/texture,
// audio device, input manager, and per-frame state (timing, FPS counter,
// turbo flag, save-state paths, optional WAV recorder). Functionality is
// split across files by concern:
//
//   - gui.go      window/renderer/texture lifecycle and the main Run loop
//   - audio.go    SDL audio init and per-frame sample queueing
//   - timing.go   frame pacing, FPS counter, window-title updates
//   - hotkeys.go  emulator-level key dispatch (Esc/Tab/Fn/1-6 + modifiers)
//   - state.go    save/load state slots, screenshots, cheat-file loading
//   - input.go    InputManager — keyboard/joystick/gamepad → NES buttons
//   - recorder.go wavRecorder + toggleRecording (Ctrl+E audio capture)
package gui

import (
	"runtime"
	"time"
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/yoshiomiyamaegones/pkg/logger"
	"github.com/yoshiomiyamaegones/pkg/nes"
	"github.com/yoshiomiyamaegones/pkg/ppu"
)

// Window constants
const (
	WindowScale  = 3
	WindowWidth  = ppu.ScreenWidth * WindowScale
	WindowHeight = ppu.ScreenHeight * WindowScale
	WindowTitle  = "GoNES - Nintendo Entertainment System Emulator"
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
	audioBuf    []byte // grown on demand; reused across queueAudio calls

	// Timing. lastRenderTime gates the turbo throttle; the frame-counter
	// baseline is reset whenever turbo turns off so the limiter doesn't try
	// to "catch up" by running the next several frames with zero sleep.
	lastFrameTime  time.Time
	nextFrameTime  time.Time
	lastRenderTime time.Time

	// FPS tracking
	fpsCounter int
	fpsTimer   time.Time
	currentFPS float64
	showFPS    bool

	// Turbo mode: skip frame limiter (and audio output) to fast-forward.
	// Toggled with Tab.
	turbo bool

	// Texture upload buffer. PPU.FrameBuffer is embedded in a struct that
	// holds other Go pointers, which cgo rejects when handed to SDL. A
	// make()'d slice has no such issue; copy() is a fast memmove.
	textureBuf []uint32

	// Input management
	inputManager *InputManager

	// ROM file path; used to derive save-state slot paths (<rom>.state1..10)
	// and recording paths (<rom>.<timestamp>.wav).
	romPath string

	// WAV recorder for Ctrl+E audio capture. nil when not recording.
	recorder *wavRecorder
}

// NewNESGUI creates a new NES GUI. romPath is used to derive save-state file
// names (<romPath-without-ext>.stateN); pass "" to disable save-state I/O.
func NewNESGUI(nesSystem *nes.NES, romPath string) (*NESGUI, error) {
	// Lock main thread for SDL
	runtime.LockOSThread()

	// Initialize SDL with video, audio, joystick, and gamecontroller support
	if err := sdl.Init(sdl.INIT_VIDEO | sdl.INIT_AUDIO | sdl.INIT_JOYSTICK | sdl.INIT_GAMECONTROLLER); err != nil {
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

	// ARGB8888 matches the PPU's 0xAARRGGBB uint32 layout byte-for-byte on
	// little-endian, so the framebuffer copies straight to the texture with
	// no per-pixel conversion.
	texture, err := renderer.CreateTexture(
		sdl.PIXELFORMAT_ARGB8888,
		sdl.TEXTUREACCESS_STREAMING,
		ppu.ScreenWidth,
		ppu.ScreenHeight,
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
		textureBuf:    make([]uint32, ppu.ScreenWidth*ppu.ScreenHeight),
		romPath:       romPath,
	}

	// Setup audio device
	if err := gui.initAudio(); err != nil {
		logger.LogError("Failed to initialize audio: %v", err)
		logger.LogError("Audio will be disabled. Check SDL2 audio drivers.")
		// Continue without audio rather than failing completely
	} else {
		logger.LogInfo("Audio initialization successful")
	}

	// Initialize input manager
	gui.inputManager = NewInputManager(nesSystem)
	gui.inputManager.Initialize()

	gui.loadCheats()

	return gui, nil
}

// Destroy cleans up SDL resources
func (g *NESGUI) Destroy() {
	// Finalize any in-progress recording so the WAV header sizes get patched.
	if g.recorder != nil {
		if err := g.recorder.Close(); err != nil {
			logger.LogError("Recording: close failed: %v", err)
		}
		g.recorder = nil
	}

	// Close input devices
	if g.inputManager != nil {
		g.inputManager.Cleanup()
	}

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

// Run starts the main GUI loop.
//
// Each iteration: poll events → step the NES → render (throttled in turbo)
// → wait until the next frame deadline. The deadline is computed from
// startTime + frameCount*FrameTime, so Sleep() overshoot doesn't drift the
// framerate downward. Exiting turbo resets that baseline to avoid a
// catch-up burst of zero-sleep frames.
func (g *NESGUI) Run() {
	frameCount := 0
	startTime := time.Now()
	wasTurbo := false

	for g.running {
		frameStart := time.Now()

		g.handleEvents()
		g.update()
		if !g.turbo || time.Since(g.lastRenderTime) >= TurboRenderInterval {
			g.render()
			g.lastRenderTime = time.Now()
		}

		// When exiting turbo, reset the frame-pacing baseline so the limiter
		// doesn't try to "catch up" by running the next several frames with
		// zero sleep.
		if wasTurbo && !g.turbo {
			frameCount = 0
			startTime = time.Now()
		}
		wasTurbo = g.turbo

		frameCount++
		g.waitForNextFrame(startTime, frameStart, frameCount)

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
			if g.handleHotkey(e) {
				continue
			}
			g.inputManager.HandleEvent(event)
		default:
			g.inputManager.HandleEvent(event)
		}
	}
}

// update runs the NES emulation for one frame
func (g *NESGUI) update() {
	// Track APU cycles before frame
	apuCyclesBefore := g.nes.APU.Cycles

	// Run NES for one frame (approximately 29780 CPU cycles)
	g.nes.StepFrame()

	if g.nes.Frame%60 == 0 && g.nes.Frame > 0 {
		apuCyclesThisFrame := g.nes.APU.Cycles - apuCyclesBefore
		samplesGenerated := len(g.nes.APU.Output)
		// Expected: ~29780 cycles/frame, ~732 samples/frame at 44100 Hz
		logger.LogDebug("Frame %d: APU cycles=%d (expected ~29780), samples=%d (expected ~732)",
			g.nes.Frame, apuCyclesThisFrame, samplesGenerated)
	}

	// Capture samples before queueAudio drains the buffer. Records raw APU
	// output (no 0.5x volume scaling) so the file is directly comparable
	// against other emulators' recordings for analysis.
	if g.recorder != nil && len(g.nes.APU.Output) > 0 {
		if err := g.recorder.WriteSamples(g.nes.APU.Output); err != nil {
			logger.LogError("Recording: write failed: %v", err)
		}
	}

	// Queue audio samples
	g.queueAudio()

	// Update FPS counter
	g.updateFPS()
}

// render draws the current frame to the screen.
func (g *NESGUI) render() {
	copy(g.textureBuf, g.nes.GetDisplayFramebufferRaw())
	g.texture.Update(nil, unsafe.Pointer(&g.textureBuf[0]), ppu.ScreenWidth*4)

	g.renderer.SetDrawColor(0, 0, 0, 255)
	g.renderer.Clear()
	g.renderer.Copy(g.texture, nil, nil)

	// Update window title with FPS if enabled
	if g.showFPS {
		g.updateWindowTitle()
	}

	// Present the rendered frame
	g.renderer.Present()
}
