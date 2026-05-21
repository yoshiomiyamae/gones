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

	// nmiDelay → pendingNMI → c.NMI is the two-step NMI deferral pipeline.
	// Each Step() advances one stage; nmiDelay is set when the PPU asserts
	// NMIRequested (either inside CPU.Step from an immediate $2000-write
	// path or during PPU catch-up from VBL set). See Step() for why two
	// stages are required to hit the right CPU instruction boundary.
	nmiDelay   bool
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
	// APU's DMC channel reads sample bytes from CPU memory. Without
	// this, the DMC's memory reader skips every fetch and the channel
	// emits only the clicks from $4011 direct writes.
	nes.APU.SetMemory(nes.Memory)

	return nes
}

// LoadCartridge loads a cartridge into the NES
func (n *NES) LoadCartridge(cart *cartridge.Cartridge) {
	n.Cartridge = cart
	n.Memory.SetCartridge(cart)
	n.PPU.SetCartridge(cart)
	n.APU.SetExpansionAudio(cart)
}

// Reset performs a power-on reset of the whole system. For the
// reset-button path (CPU registers and RAM preserved) use SoftReset.
func (n *NES) Reset() {
	n.CPU.Reset()
	n.PPU.Reset()
	n.APU.Reset()
	n.Cycles = 0
	n.Frame = 0
	n.nmiDelay = false
	n.pendingNMI = false
}

// SoftReset models the reset button: only the CPU's I-flag and stack
// pointer change (A,X,Y and RAM are preserved). PPU/APU still get a full
// reset for a usable display/audio state — blargg's cpu_reset suite
// only checks CPU state, so cycle-accurate partial PPU/APU reset isn't
// needed here.
func (n *NES) SoftReset() {
	n.CPU.SoftReset()
	n.PPU.Reset()
	n.APU.Reset()
	n.nmiDelay = false
	n.pendingNMI = false
}

// Step executes one CPU cycle
func (n *NES) Step() {
	cpuCycles := n.CPU.Step()

	// Capture an immediate NMI assertion (set inside CPU.Step by a $2000
	// write enabling NMI while VBL is set) — see the pipeline comment
	// below for why it gets a different deferral than a catch-up-side
	// assertion.
	immediateNMI := n.PPU.ConsumeNMI()

	// The CPU instruction may have written $E000 (MMC3 IRQ ack); refresh
	// the cached mirror so the level-triggered c.IRQ sync below sees it.
	if n.Cartridge != nil {
		n.PPU.MapperIRQ = n.Cartridge.IsIRQPending()
	}

	// NMI delivery pipeline — each stage advances one nes.Step:
	//   Immediate ($2000 write inside CPU.Step): immediateNMI →
	//     pendingNMI → c.NMI. 2-step (nmi_control test 11).
	//   Regular (VBL set in catch-up): NMIRequested → nmiDelay →
	//     pendingNMI → c.NMI. 3-step (nmi_timing).
	if n.pendingNMI {
		n.CPU.TriggerNMI()
		n.pendingNMI = false
	}
	if n.nmiDelay {
		n.pendingNMI = true
		n.nmiDelay = false
	}
	if immediateNMI {
		n.pendingNMI = true
	}

	for i := 0; i < cpuCycles*3; i++ {
		n.PPU.Step()

		if n.PPU.ConsumeNMI() {
			n.nmiDelay = true
		}
	}

	for i := 0; i < cpuCycles; i++ {
		n.APU.Step()
	}

	// CPU-rate mapper timers (FME-7's IRQ counter).
	if n.Cartridge != nil {
		n.Cartridge.TickCPU(cpuCycles)
		n.PPU.MapperIRQ = n.Cartridge.IsIRQPending()
	}

	// Level-triggered IRQ — OR together every line tied to the 6502's IRQ
	// input. APU frame/DMC IRQs and mapper IRQs (MMC3) all share the same
	// physical line; the CPU samples it once per cycle.
	n.CPU.IRQ = n.PPU.MapperIRQ || n.APU.FrameIRQ || n.APU.DMC.InterruptFlag

	// Run the just-completed instruction's end-of-cycle IRQ poll now that
	// the bus has caught up with this instruction's PPU/APU output. An
	// MMC3 IRQ asserted mid-instruction is visible here.
	n.CPU.PollIRQ()

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
	StateVersion uint32 = 4              // v4: + NES.nmiDelay; + PPU vblSuppressed/nmiAssertCountdown/oddFrame
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
	// nmiDelay / pendingNMI: the two-stage NMI deferral pipeline (see
	// Step). Both must be preserved so a state saved mid-pipeline doesn't
	// lose the pending interrupt.
	var flags uint8
	if n.pendingNMI {
		flags |= 1
	}
	if n.nmiDelay {
		flags |= 2
	}
	return binary.Write(w, binary.LittleEndian, flags)
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
	var flags uint8
	if err := binary.Read(r, binary.LittleEndian, &flags); err != nil {
		return fmt.Errorf("NMI pipeline flags: %w", err)
	}
	n.pendingNMI = flags&1 != 0
	n.nmiDelay = flags&2 != 0
	// Sprite size is a PPUCTRL-derived hint the cartridge layer caches
	// (MMC5 uses it to route BG vs sprite CHR fetches). PPU.LoadState
	// restores PPUCTRL but doesn't fire the $2000 write hook, so re-push
	// the bit through the same channel the live $2000 path uses.
	if n.Cartridge != nil {
		n.Cartridge.SetSpriteSize(n.PPU.PPUCTRL&ppu.PPUCTRLSpriteSize != 0)
	}
	return nil
}
