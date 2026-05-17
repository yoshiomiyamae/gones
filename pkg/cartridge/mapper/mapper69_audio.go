package mapper

// FME-7 / Sunsoft 5B expansion audio: three square-wave channels driven
// by the YM2149-style register file at the back of the bus map.
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
//	R8  / R9   channel A/B volume (bits 0-3)
//	R10        channel C volume
//
// Noise and the envelope generator (R6, R11-R13) are not implemented —
// Gimmick! drives only the three tone channels. fme7AudioSampleRate
// matches the APU's mixer; per-channel phase counters tick once per CPU
// cycle and toggle on every full period (the YM2149's classic
// /16-and-toggle is folded into the mixer chain via the volume table).
type fme7Audio struct {
	regSelect uint8
	regs      [16]uint8

	channels [3]fme7Channel
}

// fme7Channel is one of the three square-wave tone generators.
type fme7Channel struct {
	// period is the 12-bit register value as last written; an effective
	// divisor of zero would never tick, so when period == 0 we treat
	// it as 1 (matching every YM2149-derived chip and Sunsoft 5B docs).
	period  uint16
	counter uint16
	output  uint8 // 0 or 1 — the current square-wave level
	enable  bool  // mixer bit (post-invert from R7)
	volume  uint8 // 4-bit, 0..15
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
		// Bit 4 selects envelope vs fixed volume; we don't model the
		// envelope generator, so always treat the low 4 bits as a
		// fixed volume.
		a.channels[ch].volume = value & 0x0F
	}
}

// tick advances the three tone-generator phase counters by `cycles`
// CPU cycles. The 5B / YM2149 divides the input clock by 16 before
// counting against `period`; we batch through the natural counter
// arithmetic here.
func (a *fme7Audio) tick(cycles int) {
	for ch := range a.channels {
		a.channels[ch].tick(cycles)
	}
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
// APU's 0..1 floating range.
func (a *fme7Audio) sample() float32 {
	var sum float32
	for ch := range a.channels {
		c := &a.channels[ch]
		if !c.enable {
			continue
		}
		if c.output == 0 {
			continue
		}
		sum += fme7VolumeTable[c.volume]
	}
	// Three channels at peak ≈ 3.5; bring back into a sensible mixer
	// range so the 2A03 pulses don't disappear underneath.
	return sum * 0.25
}
