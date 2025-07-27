package nes

import (
	"github.com/yoshiomiyamaegones/pkg/apu"
	"github.com/yoshiomiyamaegones/pkg/cartridge"
	"github.com/yoshiomiyamaegones/pkg/cpu"
	"github.com/yoshiomiyamaegones/pkg/input"
	"github.com/yoshiomiyamaegones/pkg/memory"
	"github.com/yoshiomiyamaegones/pkg/ppu"
)

// NES represents the Nintendo Entertainment System
type NES struct {
	CPU       *cpu.CPU
	PPU       *ppu.PPU
	APU       *apu.APU
	Memory    *memory.Memory
	Cartridge *cartridge.Cartridge
	Input     *input.Controller

	Cycles uint64
	Frame  uint64
}

// NewNES creates a new NES instance
func NewNES() *NES {
	nes := &NES{}

	// Initialize components
	nes.Memory = memory.New()
	nes.CPU = cpu.New(nes.Memory)
	nes.PPU = ppu.New(nes.Memory)
	nes.APU = apu.New()
	nes.Input = input.New()

	// Connect components to memory
	nes.Memory.SetPPU(nes.PPU)
	nes.Memory.SetAPU(nes.APU)
	nes.Memory.SetInput(nes.Input)

	return nes
}

// LoadCartridge loads a cartridge into the NES
func (n *NES) LoadCartridge(cart *cartridge.Cartridge) {
	n.Cartridge = cart
	n.Memory.SetCartridge(cart)
	n.PPU.SetCartridge(cart)
}

// Reset resets the NES to initial state
func (n *NES) Reset() {
	n.CPU.Reset()
	n.PPU.Reset()
	n.APU.Reset()
	n.Cycles = 0
	n.Frame = 0
}

// Step executes one CPU cycle
func (n *NES) Step() {
	cpuCycles := n.CPU.Step()

	// PPU runs 3 times faster than CPU
	for i := 0; i < cpuCycles*3; i++ {
		n.PPU.Step()

		// Check if PPU requested NMI
		if n.PPU.NMIRequested {
			n.CPU.TriggerNMI()
			n.PPU.NMIRequested = false
		}

		// Check if mapper requested IRQ
		if n.PPU.IsMapperIRQPending() {
			n.CPU.TriggerIRQ()
			n.PPU.ClearMapperIRQ()
		}
	}

	// APU runs at CPU speed
	for i := 0; i < cpuCycles; i++ {
		n.APU.Step()
	}

	n.Cycles += uint64(cpuCycles)
}

// StepFrame executes until frame is complete
func (n *NES) StepFrame() {
	stepCount := 0
	maxSteps := 50000 // Proper limit for normal NES frame processing

	for !n.PPU.FrameComplete {
		n.Step()
		stepCount++

		// Safety check to prevent infinite loops during game freezes
		if stepCount > maxSteps {
			n.PPU.FrameComplete = true
			break
		}
	}

	n.PPU.FrameComplete = false
	// Frame counter is managed by PPU, don't increment here
	n.Frame = n.PPU.Frame
}

// GetInput returns the input controller
func (n *NES) GetInput() *input.Controller {
	return n.Input
}

// GetFramebuffer returns the current framebuffer from PPU
func (n *NES) GetFramebuffer() []uint8 {
	return n.PPU.GetFramebuffer()
}

// GetFrame returns the current frame number
func (n *NES) GetFrame() uint64 {
	return n.Frame
}

// GetFramebufferRaw returns the raw framebuffer as 32-bit integers
func (n *NES) GetFramebufferRaw() []uint32 {
	return n.PPU.FrameBuffer[:]
}

// GetDisplayFramebufferRaw returns the display framebuffer considering persistent rendering
func (n *NES) GetDisplayFramebufferRaw() []uint32 {
	return n.PPU.FrameBuffer[:]
}

// GetDisplayFramebuffer returns the display framebuffer as RGBA bytes considering persistent rendering
func (n *NES) GetDisplayFramebuffer() []uint8 {
	// Get the current frame buffer (disable persistent rendering for proper game flow)
	frameBuffer := n.PPU.FrameBuffer[:]

	// Convert 32-bit framebuffer to RGBA bytes
	rgba := make([]uint8, 256*240*4)

	for i, pixel := range frameBuffer {
		// Extract RGB components from 32-bit pixel (0xAARRGGBB format)
		r := uint8((pixel >> 16) & 0xFF) // Extract R
		g := uint8((pixel >> 8) & 0xFF)  // Extract G
		b := uint8(pixel & 0xFF)         // Extract B
		a := uint8((pixel >> 24) & 0xFF) // Extract A

		// Use RGBA order to match expected format
		rgba[i*4+0] = r
		rgba[i*4+1] = g
		rgba[i*4+2] = b
		rgba[i*4+3] = a
	}

	return rgba
}
