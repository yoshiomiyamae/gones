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

// loadNES loads the iNES ROM at romPath, builds a fresh NES system around
// it, and resets the CPU. Shared by every test that wants to run a ROM
// from disk without bringing in the GUI; the blargg-style status-polling
// loop lives in runBlarggTest, but tests that read raw framebuffer state
// (e.g. scanline_test) use loadNES directly.
func loadNES(t *testing.T, romPath string) *nes.NES {
	t.Helper()
	data, err := os.ReadFile(romPath)
	if err != nil {
		t.Fatalf("read %s: %v", romPath, err)
	}
	cart, err := cartridge.LoadFromReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("load cartridge %s: %v", romPath, err)
	}
	sys := nes.NewNES()
	sys.LoadCartridge(cart)
	sys.Reset()
	return sys
}

// runBlarggTest runs a blargg-style test ROM (status at $6000, ASCII text at
// $6004+). Returns the final status byte, the printed text, and the number of
// frames executed.
func runBlarggTest(t *testing.T, romPath string, maxFrames int) (status uint8, text string, frames int) {
	t.Helper()
	system := loadNES(t, romPath)

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

// blarggCase describes one ROM in a blargg-style suite. When expectedToPass
// is false the test is treated as a known-limitation skip with skipReason
// printed; otherwise a missing "passed" or non-zero status fails the test.
type blarggCase struct {
	name           string
	maxFrames      int
	expectedToPass bool
	skipReason     string
}

// runBlarggSuite runs a slice of blargg ROM cases under t.Run. Shared by
// the MMC3, PPU-blargg, and PPU-vbl-nmi suites — each has the same shape
// (stat → runBlarggTest → status==0 + "passed" check, with optional
// expected-fail rows that get t.Skip'd). When baseDir is empty, each
// case's `name` is treated as the full ROM path.
func runBlarggSuite(t *testing.T, baseDir string, cases []blarggCase) {
	t.Helper()
	if baseDir != "" {
		if _, err := os.Stat(baseDir); err != nil {
			t.Skipf("ROM directory not available: %v", err)
		}
	}
	for _, c := range cases {
		c := c
		t.Run(filepath.Base(c.name), func(t *testing.T) {
			romPath := c.name
			if baseDir != "" {
				romPath = filepath.Join(baseDir, c.name)
			}
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
			if passed {
				t.Logf("known-limitation test unexpectedly passed (%s) — promote it", c.skipReason)
			} else {
				t.Logf("known limitation: %s; status=$%02X", c.skipReason, status)
			}
			t.Skip(c.skipReason)
		})
	}
}

// TestMMC3BlarggSuite runs each of the 6 MMC3 test ROMs. 6-MMC6 tests the
// MMC6 / NEC-MMC3 variant where reload-to-0 from natural 0 does NOT fire
// IRQ — directly conflicts with Sharp MMC3 (which every other test and
// the vast majority of commercial MMC3 ROMs depend on). We pick Sharp and
// skip 6.
func TestMMC3BlarggSuite(t *testing.T) {
	runBlarggSuite(t, blarggMMC3TestDir, []blarggCase{
		{name: "1-clocking.nes", maxFrames: 600, expectedToPass: true},
		{name: "2-details.nes", maxFrames: 1200, expectedToPass: true},
		{name: "3-A12_clocking.nes", maxFrames: 600, expectedToPass: true},
		{name: "4-scanline_timing.nes", maxFrames: 1800, expectedToPass: true},
		{name: "5-MMC3.nes", maxFrames: 1800, expectedToPass: true},
		{name: "6-MMC6.nes", maxFrames: 1200, expectedToPass: false,
			skipReason: "tests MMC6/NEC-MMC3 variant; we implement Sharp MMC3 (needed by test 5)"},
	})
}
