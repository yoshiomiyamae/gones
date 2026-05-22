package ppu

import (
	"testing"

	"github.com/yoshiomiyamaegones/pkg/memory"
)

func TestGetFramebufferRGBA(t *testing.T) {
	p := New(memory.New())
	p.FrameBuffer[0] = 0xFF112233 // ARGB
	rgba := p.GetFramebuffer()
	if len(rgba) != ScreenWidth*ScreenHeight*4 {
		t.Fatalf("len = %d, want %d", len(rgba), ScreenWidth*ScreenHeight*4)
	}
	// 0xAARRGGBB -> R,G,B,A byte order.
	if rgba[0] != 0x11 || rgba[1] != 0x22 || rgba[2] != 0x33 || rgba[3] != 0xFF {
		t.Errorf("pixel0 RGBA = %02X%02X%02X%02X, want 11 22 33 FF",
			rgba[0], rgba[1], rgba[2], rgba[3])
	}
}

func TestMapperIRQAccessors(t *testing.T) {
	p := New(memory.New())
	if p.IsMapperIRQPending() {
		t.Error("fresh PPU should report no mapper IRQ")
	}
	p.MapperIRQ = true
	if !p.IsMapperIRQPending() {
		t.Error("IsMapperIRQPending should reflect MapperIRQ")
	}
	// ClearMapperIRQ with no cartridge just drops the cached flag.
	p.ClearMapperIRQ()
	if p.IsMapperIRQPending() {
		t.Error("ClearMapperIRQ should clear the flag")
	}
}
