// Package gui — save/load state slots, screenshots, and cheat-file loading.
package gui

import (
	"fmt"
	"os"
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/yoshiomiyamaegones/pkg/cheat"
	"github.com/yoshiomiyamaegones/pkg/logger"
	"github.com/yoshiomiyamaegones/pkg/nes"
)

// loadCheats reads <romPath>.cht if present and feeds the entries to the
// NES cheat manager. Missing files are silent (cheats are opt-in); parse
// errors per-line are logged but don't block other cheats from loading.
func (g *NESGUI) loadCheats() {
	if g.romPath == "" {
		return
	}
	path := nes.CompanionFile(g.romPath, ".cht")
	f, err := os.Open(path)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.LogError("Cheats: open %s: %v", path, err)
		}
		return
	}
	defer f.Close()
	cheats, err := cheat.LoadFile(f)
	if err != nil {
		logger.LogError("Cheats: %v", err)
	}
	for _, c := range cheats {
		g.nes.Cheats.Add(c)
	}
	logger.LogInfo("Cheats: loaded %d from %s", len(cheats), path)
}

// stateSlotPath returns the .stateN path next to the ROM.
func (g *NESGUI) stateSlotPath(slot int) string {
	return nes.CompanionFile(g.romPath, fmt.Sprintf(".state%d", slot))
}

func (g *NESGUI) saveStateSlot(slot int) {
	if g.romPath == "" {
		logger.LogError("Save state: no ROM path configured")
		return
	}
	path := g.stateSlotPath(slot)
	f, err := os.Create(path)
	if err != nil {
		logger.LogError("Save state slot %d: %v", slot, err)
		return
	}
	defer f.Close()
	if err := g.nes.SaveState(f); err != nil {
		logger.LogError("Save state slot %d: %v", slot, err)
		return
	}
	logger.LogInfo("Saved state to slot %d: %s", slot, path)
}

func (g *NESGUI) loadStateSlot(slot int) {
	if g.romPath == "" {
		logger.LogError("Load state: no ROM path configured")
		return
	}
	path := g.stateSlotPath(slot)
	f, err := os.Open(path)
	if err != nil {
		logger.LogError("Load state slot %d: %v", slot, err)
		return
	}
	defer f.Close()
	if err := g.nes.LoadState(f); err != nil {
		logger.LogError("Load state slot %d: %v", slot, err)
		return
	}
	logger.LogInfo("Loaded state from slot %d: %s", slot, path)
}

// saveScreenshot saves the current screen to a file
func (g *NESGUI) saveScreenshot() {
	filename := fmt.Sprintf("screenshot_%03d.png", g.screenshotNum)
	g.screenshotNum++
	g.saveScreenshotWithName(filename)
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
