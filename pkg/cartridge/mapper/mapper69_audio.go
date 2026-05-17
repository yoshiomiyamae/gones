package mapper

// FME-7 / Sunsoft 5B expansion audio: three square-wave channels and a
// shared envelope generator driven by the YM2149-style register file
// at the back of the bus map.
//
//	$C000-$DFFF write  selects which internal register the next
//	                   $E000-$FFFF write targets
//	$E000-$FFFF write  writes data to the selected register
//
// Register layout (subset relevant to NES titles):
//
//	R0  / R1   channel A period (12 bits split low/high)
//	R2  / R3   channel B period
//	R4  / R5   channel C period
//	R7         mixer enable: bit 0/1/2 = A/B/C tone enable (active-low
//	           on real silicon — 1 means *disabled*)
//	R8/R9/R10  channel A/B/C volume — bit 4 = "use envelope", bits
//	           0-3 = fixed 4-bit volume
//	R11 / R12  envelope period (low + high)
//	R13        envelope shape (CONTINUE/ATTACK/ALTERNATE/HOLD)
//
// R6 (noise period) is the only register still unimplemented; Gimmick!
// doesn't use it. Tone channels tick once per CPU cycle and toggle
// every period × 16 cycles (YM2149's /16 divider folded into the
// counter math). The envelope clocks at master / 256 / period.
type fme7Audio struct {
	regSelect uint8
	regs      [16]uint8

	channels [3]fme7Channel
	envelope fme7Envelope
}

// fme7Channel is one of the three square-wave tone generators.
type fme7Channel struct {
	// period is the 12-bit register value as last written; an effective
	// divisor of zero would never tick, so when period == 0 we treat
	// it as 1 (matching every YM2149-derived chip and Sunsoft 5B docs).
	period      uint16
	counter     uint16
	output      uint8 // 0 or 1 — the current square-wave level
	enable      bool  // mixer bit (post-invert from R7)
	volume      uint8 // 4-bit, 0..15
	useEnvelope bool  // R8/R9/R10 bit 4: true → take volume from the envelope unit
}

// fme7Envelope is the single shared envelope generator. R11/R12 set
// the 16-bit period; R13 sets the shape. Output is a 4-bit value in
// 0..15 that channels can use as their volume.
type fme7Envelope struct {
	period  uint16 // R11 low + R12 high
	counter uint32
	step    uint8 // 0..31 within the current envelope cycle
	shape   uint8 // R13: CONTINUE / ATTACK / ALTERNATE / HOLD bits
	output  uint8 // 0..15
	holding bool  // true once a non-continue envelope settles
}

// fme7VolumeTable is the YM2149 logarithmic 4-bit DAC normalised so
// volume 15 → 1.0. Successive entries are ~1/sqrt(2) (≈ 3 dB) apart,
// matching the standard "3 dB per step" curve cited by the 5B docs.
var fme7VolumeTable = [16]float32{
	0.0000, 0.0078, 0.0110, 0.0156, 0.0221, 0.0312, 0.0442, 0.0625,
	0.0884, 0.1250, 0.1768, 0.2500, 0.3536, 0.5000, 0.7071, 1.0000,
}

// writeAudioSelect handles $C000-$DFFF writes (latches the register
// number for the next data write).
func (a *fme7Audio) writeAudioSelect(value uint8) {
	a.regSelect = value & 0x0F
}

// writeAudioData handles $E000-$FFFF writes (data for the latched
// register).
func (a *fme7Audio) writeAudioData(value uint8) {
	a.regs[a.regSelect] = value
	switch a.regSelect {
	case 0, 2, 4:
		ch := a.regSelect / 2
		a.channels[ch].period = (a.channels[ch].period & 0x0F00) | uint16(value)
	case 1, 3, 5:
		ch := (a.regSelect - 1) / 2
		a.channels[ch].period = (a.channels[ch].period & 0x00FF) | (uint16(value&0x0F) << 8)
	case 7:
		// R7 bit 0/1/2 = tone-A/B/C disable (active-low). Bits 3-5 are
		// noise enables, ignored here. Bits 6-7 are I/O direction, also
		// unused.
		a.channels[0].enable = value&0x01 == 0
		a.channels[1].enable = value&0x02 == 0
		a.channels[2].enable = value&0x04 == 0
	case 8, 9, 10:
		ch := a.regSelect - 8
		// Bit 4 = 1: use the shared envelope generator's output as
		// volume. Bit 4 = 0: bits 0-3 are a fixed 4-bit volume.
		a.channels[ch].useEnvelope = value&0x10 != 0
		a.channels[ch].volume = value & 0x0F
	case 11:
		a.envelope.period = (a.envelope.period & 0xFF00) | uint16(value)
	case 12:
		a.envelope.period = (a.envelope.period & 0x00FF) | (uint16(value) << 8)
	case 13:
		// Writing R13 resets the envelope cycle and arms the shape.
		// Per AY-3-8910/YM2149: ATTACK bit (bit 2) sets the initial
		// direction; without it the envelope counts down from 15.
		a.envelope.shape = value & 0x0F
		a.envelope.counter = 0
		a.envelope.step = 0
		a.envelope.holding = false
		if value&0x04 != 0 {
			a.envelope.output = 0
		} else {
			a.envelope.output = 15
		}
	}
}

// tick advances the three tone-generator phase counters and the
// shared envelope unit by `cycles` CPU cycles. The 5B / YM2149
// divides the input clock by 16 before counting against `period`.
func (a *fme7Audio) tick(cycles int) {
	for ch := range a.channels {
		a.channels[ch].tick(cycles)
	}
	a.envelope.tick(cycles)
}

func (c *fme7Channel) tick(cycles int) {
	period := c.period
	if period == 0 {
		period = 1
	}
	// One toggle per (period * 16) CPU cycles. Maintain a counter in
	// "CPU cycles" and flip output whenever it crosses period*16.
	step := uint16(16)
	for i := 0; i < cycles; i++ {
		c.counter++
		if c.counter >= period*step {
			c.counter = 0
			c.output ^= 1
		}
	}
}

// sample returns the mixed analog level of the three channels in the
// APU's 0..1 floating range. The per-channel volumes land roughly in
// the same loudness ballpark as a 2A03 pulse at half volume; without
// the scale the FME-7 mix sat at ~1/4 the 2A03 level and the bass
// channel disappeared into the noise floor.
func (a *fme7Audio) sample() float32 {
	var sum float32
	for ch := range a.channels {
		c := &a.channels[ch]
		if !c.enable || c.output == 0 {
			continue
		}
		vol := c.volume
		if c.useEnvelope {
			vol = a.envelope.output
		}
		sum += fme7VolumeTable[vol]
	}
	// Three channels at peak ≈ 3.0; scaled so a single channel at full
	// volume ≈ 0.33 — roughly comparable to one 2A03 pulse at duty 50%.
	return sum * 0.33
}

// tick advances the envelope's internal counter and steps through the
// 32-position waveform when the configured period elapses. The shape
// register's four bits (CONTINUE / ATTACK / ALTERNATE / HOLD) decide
// what happens after the first 16-step ramp completes — see the
// AY-3-8910 envelope-shape table.
//
// The 5B clocks the envelope counter at clock/256 (effectively /16
// from the tone divider and /16 from an internal counter); we collapse
// that into "every (period * 256) CPU cycles, advance one step".
func (e *fme7Envelope) tick(cycles int) {
	period := e.period
	if period == 0 {
		period = 1
	}
	div := uint32(period) * 256
	for i := 0; i < cycles; i++ {
		e.counter++
		if e.counter < div {
			continue
		}
		e.counter = 0
		if e.holding {
			continue
		}
		e.advance()
	}
}

// advance moves the envelope one step through the 32-position table
// and applies the shape semantics. Shape bits: 0=HOLD, 1=ALTERNATE,
// 2=ATTACK, 3=CONTINUE.
func (e *fme7Envelope) advance() {
	attack := e.shape&0x04 != 0
	alt := e.shape&0x02 != 0
	hold := e.shape&0x01 != 0
	cont := e.shape&0x08 != 0

	e.step++
	if e.step < 16 {
		// First ramp.
		if attack {
			e.output = e.step
		} else {
			e.output = 15 - e.step
		}
		return
	}

	// First ramp complete (step == 16). The continue bit decides
	// whether the envelope keeps moving; without it the chip drops to
	// 0 and holds.
	if !cont {
		e.output = 0
		e.holding = true
		return
	}

	switch {
	case hold:
		// CONTINUE+HOLD: stay at the final value of the first ramp.
		if alt {
			// CONT+ATTACK+HOLD+ALT or CONT+HOLD+ALT: invert final.
			if attack {
				e.output = 0
			} else {
				e.output = 15
			}
		} else {
			if attack {
				e.output = 15
			} else {
				e.output = 0
			}
		}
		e.holding = true
	case alt:
		// CONTINUE+ALT (no hold): triangular waveform. Steps 0-15 run
		// the first ramp (handled above), steps 16-31 run the reverse
		// ramp; at step 32 we restart from 0 so the *first* ramp
		// fires again. Resetting to 16 here would lock us into a
		// one-directional sawtooth after the initial cycle.
		if e.step >= 32 {
			e.step = 0
			if attack {
				e.output = 0
			} else {
				e.output = 15
			}
			return
		}
		sub := e.step - 16
		if attack {
			e.output = 15 - sub
		} else {
			e.output = sub
		}
	default:
		// CONTINUE without HOLD or ALT: sawtooth — restart from 0/15
		// every 16 steps.
		e.step = 0
		if attack {
			e.output = 0
		} else {
			e.output = 15
		}
	}
}
