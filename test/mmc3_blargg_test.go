package test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yoshiomiyamaegones/pkg/cartridge"
	"github.com/yoshiomiyamaegones/pkg/nes"
)

// blarggMMC3TestDir is the directory containing Shay Green's mmc3_test ROMs.
// These tests write their status to $6000 (a-la blargg test runner protocol).
const blarggMMC3TestDir = `R:\nes-test-roms-master\mmc3_test`

// runBlarggTest runs a blargg-style test ROM (status at $6000, ASCII text at
// $6004+). Returns the final status byte, the printed text, and the number of
// frames executed.
func runBlarggTest(t *testing.T, romPath string, maxFrames int) (status uint8, text string, frames int) {
	t.Helper()

	data, err := os.ReadFile(romPath)
	if err != nil {
		t.Fatalf("read %s: %v", romPath, err)
	}
	cart, err := cartridge.LoadFromReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("load cartridge %s: %v", romPath, err)
	}

	system := nes.NewNES()
	system.LoadCartridge(cart)
	system.Reset()

	const (
		statusAddr = 0x6000
		sigAddr    = 0x6001
		textAddr   = 0x6004
		runFlag    = 0x80
		resetFlag  = 0x81
	)

	signatureSeen := false
	for frames = 0; frames < maxFrames; frames++ {
		system.StepFrame()

		s0 := system.Memory.Read(sigAddr)
		s1 := system.Memory.Read(sigAddr + 1)
		s2 := system.Memory.Read(sigAddr + 2)
		if s0 == 0xDE && s1 == 0xB0 && s2 == 0x61 {
			signatureSeen = true
		}

		status = system.Memory.Read(statusAddr)
		if !signatureSeen {
			continue
		}
		if status == resetFlag {
			system.Reset()
			continue
		}
		if status < runFlag && status != 0x00 {
			break
		}
		if status == 0x00 && frames > 60 {
			break
		}
	}

	var b strings.Builder
	for i := 0; i < 0x2000; i++ {
		c := system.Memory.Read(uint16(textAddr + i))
		if c == 0 {
			break
		}
		if c >= 0x20 && c < 0x7F {
			b.WriteByte(c)
		} else if c == '\n' {
			b.WriteByte('\n')
		}
	}
	text = strings.TrimSpace(b.String())
	return status, text, frames
}

// TestMMC3BlarggSuite runs each of the 6 MMC3 test ROMs. A ROM passes when
// the status byte at $6000 reaches 0 and the printed text contains "passed".
//
// Notes on coverage:
//   - 1-clocking, 2-details, 3-A12_clocking, 4-scanline_timing, 5-MMC3:
//     core MMC3 IRQ counter, A12 rising-edge clocking, and cycle-accurate
//     raster IRQ timing — all pass.
//   - 6-MMC6: tests a MMC6 / NEC-MMC3 variant where reloading the counter
//     to 0 from a "natural" 0 state does NOT fire IRQ. We implement the
//     Sharp MMC3 behaviour (reload-to-0 always fires) because every other
//     test in this suite — and the vast majority of commercial MMC3 ROMs —
//     depend on the Sharp semantics. The two variants are mutually
//     exclusive; we deliberately fail 6 in favour of passing 5.
func TestMMC3BlarggSuite(t *testing.T) {
	if _, err := os.Stat(blarggMMC3TestDir); err != nil {
		t.Skipf("mmc3_test ROM directory not available: %v", err)
	}

	type tc struct {
		name           string
		maxFrames      int
		expectedToPass bool
		skipReason     string
	}
	cases := []tc{
		{name: "1-clocking.nes", maxFrames: 600, expectedToPass: true},
		{name: "2-details.nes", maxFrames: 1200, expectedToPass: true},
		{name: "3-A12_clocking.nes", maxFrames: 600, expectedToPass: true},
		{name: "4-scanline_timing.nes", maxFrames: 1800, expectedToPass: true},
		{name: "5-MMC3.nes", maxFrames: 1800, expectedToPass: true},
		{name: "6-MMC6.nes", maxFrames: 1200, expectedToPass: false,
			skipReason: "tests MMC6/NEC-MMC3 variant; we implement Sharp MMC3 (needed by test 5)"},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			romPath := filepath.Join(blarggMMC3TestDir, c.name)
			if _, err := os.Stat(romPath); err != nil {
				t.Skipf("ROM missing: %v", err)
			}
			status, text, frames := runBlarggTest(t, romPath, c.maxFrames)
			t.Logf("frames=%d status=$%02X output=%q", frames, status, text)
			passed := status == 0x00 && strings.Contains(strings.ToLower(text), "passed")
			if c.expectedToPass {
				if !passed {
					t.Fatalf("expected pass but failed: status=$%02X output=%q", status, text)
				}
				return
			}
			// Known limitation: log result but don't fail the suite.
			if passed {
				t.Logf("known-limitation test unexpectedly passed (%s) — promote it", c.skipReason)
			} else {
				t.Logf("known limitation: %s; status=$%02X", c.skipReason, status)
			}
			t.Skip(c.skipReason)
		})
	}
}
