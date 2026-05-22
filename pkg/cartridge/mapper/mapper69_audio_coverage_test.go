package mapper

import (
	"math"
	"testing"
)

func TestFME7AudioRegisterFile(t *testing.T) {
	a := &fme7Audio{}

	// Channel A/B period (low+high) written; channel C left at period 0 so
	// the tick path's "period == 0 -> 1" guard is exercised.
	a.writeAudioSelect(0)
	a.writeAudioData(0x34) // A period low
	a.writeAudioSelect(1)
	a.writeAudioData(0x02) // A period high
	a.writeAudioSelect(2)
	a.writeAudioData(0x10) // B period low
	a.writeAudioSelect(3)
	a.writeAudioData(0x00)
	a.writeAudioSelect(4)
	a.writeAudioData(0x00) // C period low = 0
	a.writeAudioSelect(5)
	a.writeAudioData(0x00)

	// Period split must land in the right halves: A = 0x234.
	if a.channels[0].period != 0x234 {
		t.Errorf("channel A period = %#x, want 0x234", a.channels[0].period)
	}

	// R7 mixer is active-low: bit set => channel disabled.
	a.writeAudioSelect(7)
	a.writeAudioData(0x02) // disable B only
	if !a.channels[0].enable || a.channels[1].enable || !a.channels[2].enable {
		t.Errorf("R7=0x02 enables = %v/%v/%v, want true/false/true",
			a.channels[0].enable, a.channels[1].enable, a.channels[2].enable)
	}
	a.writeAudioSelect(7)
	a.writeAudioData(0x00) // re-enable all

	// Volumes: bit 4 selects envelope; bits 0-3 fixed volume.
	a.writeAudioSelect(8)
	a.writeAudioData(0x10) // ch A: useEnvelope
	a.writeAudioSelect(9)
	a.writeAudioData(0x0A) // ch B: fixed vol 10
	a.writeAudioSelect(10)
	a.writeAudioData(0x00) // ch C: fixed vol 0
	if !a.channels[0].useEnvelope || a.channels[1].useEnvelope {
		t.Errorf("useEnvelope = %v/%v, want true/false", a.channels[0].useEnvelope, a.channels[1].useEnvelope)
	}
	if a.channels[1].volume != 0x0A {
		t.Errorf("ch B volume = %#x, want 0x0A", a.channels[1].volume)
	}

	// sample() must mix: ch A via envelope.output, B/C fixed, all output high.
	a.envelope.output = 15
	a.channels[0].output = 1
	a.channels[1].output = 1
	a.channels[2].output = 1
	// table[15] + table[10] + table[0] = 1.0 + 0.1768 + 0.0, scaled by 0.33.
	want := (fme7VolumeTable[15] + fme7VolumeTable[10] + fme7VolumeTable[0]) * 0.33
	if got := a.sample(); math.Abs(float64(got-want)) > 1e-5 {
		t.Errorf("sample() = %f, want %f", got, want)
	}

	// A muted (output 0) or disabled channel contributes nothing.
	a.channels[0].output = 0
	wantMuted := (fme7VolumeTable[10] + fme7VolumeTable[0]) * 0.33
	if got := a.sample(); math.Abs(float64(got-wantMuted)) > 1e-5 {
		t.Errorf("sample() with ch A muted = %f, want %f", got, wantMuted)
	}
}

func TestFME7EnvelopeAttackHold(t *testing.T) {
	// Shape 0x04 = ATTACK, no CONTINUE: ramp up 0..15, then drop to 0 and hold.
	e := &fme7Envelope{shape: 0x04}
	e.output = 0
	for i := 0; i < 5; i++ {
		e.advance()
	}
	if e.output != 5 {
		t.Errorf("attack ramp after 5 steps = %d, want 5", e.output)
	}
	for i := 5; i < 16; i++ {
		e.advance() // reach step 16
	}
	if e.output != 0 || !e.holding {
		t.Errorf("non-continue settle: output=%d holding=%v, want 0/true", e.output, e.holding)
	}
}

func TestFME7EnvelopeSawtooth(t *testing.T) {
	// Shape 0x0C = CONTINUE+ATTACK: repeating up-sawtooth, resets every 16 steps.
	e := &fme7Envelope{shape: 0x0C}
	e.output = 0
	for i := 0; i < 15; i++ {
		e.advance()
	}
	if e.output != 15 {
		t.Errorf("sawtooth peak = %d, want 15", e.output)
	}
	e.advance() // step 16 -> reset to 0
	if e.output != 0 || e.holding {
		t.Errorf("sawtooth wrap: output=%d holding=%v, want 0/false", e.output, e.holding)
	}
}

func TestFME7EnvelopeTriangle(t *testing.T) {
	// Shape 0x0A = CONTINUE+ALTERNATE (no attack): down-ramp then up-ramp,
	// returning to the start — guards the "reset to step 0 not 16" invariant.
	e := &fme7Envelope{shape: 0x0A}
	e.output = 15
	for i := 0; i < 15; i++ {
		e.advance()
	}
	if e.output != 0 {
		t.Errorf("triangle trough after 15 steps = %d, want 0", e.output)
	}
	for i := 15; i < 31; i++ {
		e.advance() // climb the reverse ramp
	}
	if e.output != 15 {
		t.Errorf("triangle peak after 31 steps = %d, want 15", e.output)
	}
}

func TestFME7EnvelopeTickHonorsHolding(t *testing.T) {
	// Driven through tick() (the real call path): a non-continue shape must
	// settle to 0 and STAY there — tick's `if holding { continue }` gate.
	a := &fme7Audio{}
	a.writeAudioSelect(11)
	a.writeAudioData(0x01) // envelope period low = 1 (fast)
	a.writeAudioSelect(13)
	a.writeAudioData(0x04) // ATTACK, no CONTINUE
	a.tick(200000)
	if !a.envelope.holding || a.envelope.output != 0 {
		t.Errorf("after tick: holding=%v output=%d, want true/0", a.envelope.holding, a.envelope.output)
	}
}

func TestFME7EnvelopeShapeArms(t *testing.T) {
	// Coverage sweep over all 16 shapes so every advance() arm runs; the
	// behavioral assertions live in the dedicated tests above.
	for shape := uint8(0); shape <= 0x0F; shape++ {
		e := &fme7Envelope{shape: shape}
		if shape&0x04 == 0 {
			e.output = 15
		}
		for i := 0; i < 40; i++ {
			e.advance()
		}
	}
}
