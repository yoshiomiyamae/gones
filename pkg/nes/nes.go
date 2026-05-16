package nes

import (
	"encoding/binary"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/yoshiomiyamaegones/pkg/apu"
	"github.com/yoshiomiyamaegones/pkg/cartridge"
	"github.com/yoshiomiyamaegones/pkg/cheat"
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
	Cheats    *cheat.Manager

	Cycles uint64
	Frame  uint64

	// pendingNMI is the 1-step deferred NMI flag — see Step() for the why.
	pendingNMI bool
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
	nes.Cheats = cheat.NewManager()

	// Connect components to memory
	nes.Memory.SetPPU(nes.PPU)
	nes.Memory.SetAPU(nes.APU)
	nes.Memory.SetInput(nes.Input)
	nes.Memory.Cheats = nes.Cheats

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
	// Run the CPU instruction first. If a previous PPU catch-up set
	// pendingNMI, we hold off triggering it until AFTER this instruction —
	// this lets a polling LDA $2002 see the vblank flag that fired during
	// the previous step's catch-up, matching real-hardware behavior where
	// vblank flips during the polling instruction's read cycle and NMI is
	// sampled at end-of-instruction. Without this, Dragon Quest III hangs
	// in its vblank-wait loop because the NMI handler steals the flag.
	cpuCycles := n.CPU.Step()

	// The CPU instruction may have written $E000 (MMC3 IRQ ack) which
	// drops the mapper's pending bit. Refresh the PPU's cached mirror so
	// the level-triggered c.IRQ sync below sees the new state — without
	// this, blargg's mmc3_test get_pending CLI fires a phantom IRQ.
	if n.Cartridge != nil {
		n.PPU.MapperIRQ = n.Cartridge.IsIRQPending()
	}

	if n.pendingNMI {
		n.CPU.TriggerNMI()
		n.pendingNMI = false
	}

	for i := 0; i < cpuCycles*3; i++ {
		n.PPU.Step()

		if n.PPU.NMIRequested {
			n.pendingNMI = true
			n.PPU.NMIRequested = false
		}

		// Level-triggered IRQ: track the mapper line every cycle so a
		// mid-instruction mapper-pending rise from the PPU side is
		// visible at the next instruction boundary, and an $E000 ack
		// drops the line immediately rather than firing spuriously.
		n.CPU.IRQ = n.PPU.MapperIRQ
	}

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

// CompanionFile returns a path co-located with romPath, with its extension
// replaced by suffix. Used for sidecar files like SRAM (".sav") and save
// states (".state1" etc.). suffix must include the leading dot.
func CompanionFile(romPath, suffix string) string {
	return strings.TrimSuffix(romPath, filepath.Ext(romPath)) + suffix
}

// Save-state file header constants. Bump StateVersion whenever the on-disk
// layout of any component changes — older files are then rejected at load.
const (
	stateMagic   uint32 = 0x47_4E_53_54 // "GNST"
	StateVersion uint32 = 3              // v3: + NES.pendingNMI byte after cartridge section
)

// SaveState writes a complete emulator snapshot (CPU + PPU + APU + memory +
// cartridge + mapper) to w. Each component handles its own serialization;
// the order here must match LoadState.
func (n *NES) SaveState(w io.Writer) error {
	if err := binary.Write(w, binary.LittleEndian, stateMagic); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, StateVersion); err != nil {
		return err
	}
	if err := n.CPU.SaveState(w); err != nil {
		return fmt.Errorf("cpu: %w", err)
	}
	if err := n.PPU.SaveState(w); err != nil {
		return fmt.Errorf("ppu: %w", err)
	}
	if err := n.APU.SaveState(w); err != nil {
		return fmt.Errorf("apu: %w", err)
	}
	if err := n.Memory.SaveState(w); err != nil {
		return fmt.Errorf("memory: %w", err)
	}
	if n.Cartridge != nil {
		if err := n.Cartridge.SaveState(w); err != nil {
			return fmt.Errorf("cartridge: %w", err)
		}
	}
	// pendingNMI: deferred-by-1-instruction NMI flag (see Step). Must be
	// preserved so a state saved between vblank and NMI handler doesn't lose
	// the pending interrupt.
	var pending uint8
	if n.pendingNMI {
		pending = 1
	}
	return binary.Write(w, binary.LittleEndian, pending)
}

// LoadState restores a snapshot written by SaveState. The cartridge must
// already be the same ROM as when the state was saved — we restore RAM/mapper
// state but never ROM contents.
func (n *NES) LoadState(r io.Reader) error {
	var magic, version uint32
	if err := binary.Read(r, binary.LittleEndian, &magic); err != nil {
		return err
	}
	if magic != stateMagic {
		return fmt.Errorf("not a GoNES state file (magic=%#x)", magic)
	}
	if err := binary.Read(r, binary.LittleEndian, &version); err != nil {
		return err
	}
	if version != StateVersion {
		return fmt.Errorf("state version mismatch: file=%d, want=%d", version, StateVersion)
	}
	if err := n.CPU.LoadState(r); err != nil {
		return fmt.Errorf("cpu: %w", err)
	}
	if err := n.PPU.LoadState(r); err != nil {
		return fmt.Errorf("ppu: %w", err)
	}
	if err := n.APU.LoadState(r); err != nil {
		return fmt.Errorf("apu: %w", err)
	}
	if err := n.Memory.LoadState(r); err != nil {
		return fmt.Errorf("memory: %w", err)
	}
	if n.Cartridge != nil {
		if err := n.Cartridge.LoadState(r); err != nil {
			return fmt.Errorf("cartridge: %w", err)
		}
	}
	var pending uint8
	if err := binary.Read(r, binary.LittleEndian, &pending); err != nil {
		return fmt.Errorf("pendingNMI: %w", err)
	}
	n.pendingNMI = pending != 0
	return nil
}
