package test

import (
	"bytes"
	"os"
	"testing"

	"github.com/yoshiomiyamaegones/pkg/cartridge"
	"github.com/yoshiomiyamaegones/pkg/nes"
)

// benchROMs are representative ROMs that exercise the rendering hot paths.
// BladeBuster is a sprite- and scroll-heavy demo (stresses the sprite CHR
// fetch path); full_palette redraws a full-screen background every frame.
var benchROMs = []struct {
	name string
	path string
}{
	{"BladeBuster", `R:\nes-test-roms-master\other\BladeBuster.nes`},
	{"full_palette", `R:\nes-test-roms-master\full_palette\full_palette.nes`},
	{"scanline", `R:\nes-test-roms-master\scanline\scanline.nes`},
}

// newBenchNES loads a ROM and returns a reset NES, skipping the benchmark if
// the ROM isn't present on this machine.
func newBenchNES(b *testing.B, path string) *nes.NES {
	b.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		b.Skipf("ROM not available: %v", err)
	}
	cart, err := cartridge.LoadFromReader(bytes.NewReader(data))
	if err != nil {
		b.Fatalf("load cartridge: %v", err)
	}
	system := nes.NewNES()
	system.LoadCartridge(cart)
	system.Reset()
	return system
}

// BenchmarkStepFrame measures the cost of emulating one full frame
// (CPU + PPU + APU + mapper) with no SDL/audio/sleep overhead.
func BenchmarkStepFrame(b *testing.B) {
	for _, rom := range benchROMs {
		b.Run(rom.name, func(b *testing.B) {
			system := newBenchNES(b, rom.path)
			// Warm up past the boot/title-init so we measure steady-state
			// rendering rather than the initial black frames.
			for i := 0; i < 120; i++ {
				system.StepFrame()
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				system.StepFrame()
			}
		})
	}
}
