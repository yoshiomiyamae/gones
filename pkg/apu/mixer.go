package apu

// NESdev wiki "APU Mixer" recommends three cascaded 1-pole filters: HPF
// 90 Hz, HPF 440 Hz, LPF 14 kHz. We skip the 440 Hz HPF — it cuts ~6 dB
// from 80-250 Hz where pulse/triangle basslines live. Coefficients are
// pre-computed for 44100 Hz; revisit if AudioSampleRate ever changes.
const (
	hpf90Alpha     = 0.9873 // RC / (RC + dt) for fc=90Hz, fs=44100
	lpf14kAlpha    = 0.6661 // dt / (RC + dt) for fc=14000Hz, fs=44100
	lpf14kFeedback = 1.0 - lpf14kAlpha
)

// mixChannels mixes all audio channels using proper NES mixing
func (a *APU) mixChannels() float32 {
	pulse1 := a.getPulseOutput(&a.Pulse1)
	pulse2 := a.getPulseOutput(&a.Pulse2)
	triangle := a.getTriangleOutput()
	noise := a.getNoiseOutput()
	dmc := a.getDMCOutput()

	// Apply diagnostic mutes. Channels keep ticking so timing-dependent
	// state (length / linear / envelope) stays in sync — only the audible
	// contribution is suppressed.
	if a.ChannelMute[0] {
		pulse1 = 0
	}
	if a.ChannelMute[1] {
		pulse2 = 0
	}
	if a.ChannelMute[2] {
		triangle = 0
	}
	if a.ChannelMute[3] {
		noise = 0
	}
	if a.ChannelMute[4] {
		dmc = 0
	}

	// NESdev nonlinear mixer (https://www.nesdev.org/wiki/APU_Mixer) — what
	// FCEUX / Mesen / Nestopia use. Output naturally lands in 0..1.0.
	pulseSum := pulse1 + pulse2
	var pulseOut float32
	if pulseSum > 0 {
		pulseOut = 95.88 / (8128.0/float32(pulseSum) + 100.0)
	}

	tndSum := float32(triangle)/8227.0 + float32(noise)/12241.0 + float32(dmc)/22638.0
	var tndOut float32
	if tndSum > 0 {
		tndOut = 159.79 / (1.0/tndSum + 100.0)
	}

	output := pulseOut + tndOut
	if output > 1.0 {
		output = 1.0
	}

	if a.FilterEnabled {
		output = a.applyAnalogFilters(output)
	}
	return output
}

func (a *APU) applyAnalogFilters(x float32) float32 {
	hp := hpf90Alpha * (a.hpfPrevOut + x - a.hpfPrevIn)
	a.hpfPrevIn = x
	a.hpfPrevOut = hp

	lp := lpf14kAlpha*hp + lpf14kFeedback*a.lpfPrevOut
	a.lpfPrevOut = lp
	return lp
}
