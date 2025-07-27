package apu

// Duty cycle sequences for pulse channels (8 steps each)
var dutyCycles = [4][8]uint8{
	{0, 1, 0, 0, 0, 0, 0, 0}, // 12.5%
	{0, 1, 1, 0, 0, 0, 0, 0}, // 25%
	{0, 1, 1, 1, 1, 0, 0, 0}, // 50%
	{1, 0, 0, 1, 1, 1, 1, 1}, // 25% (negated)
}

// Triangle wave sequence (32 steps)
var triangleSequence = [32]uint8{
	15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0,
	0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
}

// Noise periods for different frequencies
var noisePeriods = [16]uint16{
	4, 8, 16, 32, 64, 96, 128, 160, 202, 254, 380, 508, 762, 1016, 2034, 4068,
}

// DMC rate table (in CPU cycles)
var dmcRates = [16]uint16{
	428, 380, 340, 320, 286, 254, 226, 214, 190, 160, 142, 128, 106, 84, 72, 54,
}

// stepPulse steps a pulse channel
func (a *APU) stepPulse(pulse *PulseChannel) {
	if !pulse.Enabled {
		return
	}

	// Step timer
	if pulse.Timer > 0 {
		pulse.Timer--
	} else {
		pulse.Timer = pulse.TimerValue
		pulse.Sequence = (pulse.Sequence + 1) % 8
	}
}

// stepTriangle steps the triangle channel
func (a *APU) stepTriangle() {
	if !a.Triangle.Enabled {
		return
	}

	// Step timer
	if a.Triangle.Timer > 0 {
		a.Triangle.Timer--
	} else {
		// Triangle channel: CPU_FREQ / (32 * (timer + 1))
		// Pulse channel: CPU_FREQ / (16 * (timer + 1))
		// Triangle should be 1 octave lower than pulse for same timer value
		// But our current implementation makes them equal frequency
		// Remove the frequency correction to match actual NES behavior
		a.Triangle.Timer = a.Triangle.TimerValue
		if a.Triangle.Length.Value > 0 && a.Triangle.LinearCounter > 0 {
			a.Triangle.Sequence = (a.Triangle.Sequence + 1) % 32
		}
	}
}

// stepNoise steps the noise channel
func (a *APU) stepNoise() {
	if !a.Noise.Enabled {
		return
	}

	// Step timer
	if a.Noise.Timer > 0 {
		a.Noise.Timer--
	} else {
		a.Noise.Timer = a.Noise.TimerValue

		// Step LFSR
		bit := uint16(0)
		if a.Noise.Mode {
			// Mode 1: tap bits 0 and 6
			bit = (a.Noise.ShiftReg & 1) ^ ((a.Noise.ShiftReg >> 6) & 1)
		} else {
			// Mode 0: tap bits 0 and 1
			bit = (a.Noise.ShiftReg & 1) ^ ((a.Noise.ShiftReg >> 1) & 1)
		}

		a.Noise.ShiftReg = (a.Noise.ShiftReg >> 1) | (bit << 14)
	}
}

// stepDMC steps the DMC channel
func (a *APU) stepDMC() {
	if !a.DMC.Enabled {
		return
	}

	// DMC timer countdown
	if a.DMC.Rate > 0 {
		// Use rate table for proper timing
		dmcPeriod := dmcRates[a.DMC.Rate&0x0F]
		// Simplified timer implementation - needs proper CPU cycle timing
		if a.Cycles%uint64(dmcPeriod) == 0 {
			a.stepDMCSample()
		}
	}
}

// stepDMCSample processes DMC sample data
func (a *APU) stepDMCSample() {
	// Fill sample buffer if empty and memory available
	if a.DMC.BufferEmpty && a.DMC.CurrentLength > 0 && a.Memory != nil {
		a.DMC.SampleBuffer = a.Memory.Read(a.DMC.CurrentAddress)
		a.DMC.BufferEmpty = false
		a.DMC.CurrentAddress++
		if a.DMC.CurrentAddress > 0xFFFF {
			a.DMC.CurrentAddress = 0x8000 // Wrap to ROM area
		}
		a.DMC.CurrentLength--

		// Check for end of sample
		if a.DMC.CurrentLength == 0 {
			if a.DMC.Loop {
				// Restart sample
				a.DMC.CurrentLength = a.DMC.SampleLength
				a.DMC.CurrentAddress = a.DMC.SampleAddress
			} else if a.DMC.IRQEnabled {
				// Generate IRQ (would need CPU interface)
			}
		}
	}

	// Process bits from buffer
	if a.DMC.BitsRemaining == 0 {
		a.DMC.BitsRemaining = 8
		if !a.DMC.BufferEmpty {
			a.DMC.Buffer = a.DMC.SampleBuffer
			a.DMC.BufferEmpty = true
			a.DMC.Silence = false
		} else {
			a.DMC.Silence = true
		}
	}

	if a.DMC.BitsRemaining > 0 && !a.DMC.Silence {
		a.DMC.BitsRemaining--
		bit := (a.DMC.Buffer >> a.DMC.BitsRemaining) & 1

		// Update output counter
		if bit == 1 && a.DMC.LoadCounter <= 125 {
			a.DMC.LoadCounter += 2
		} else if bit == 0 && a.DMC.LoadCounter >= 2 {
			a.DMC.LoadCounter -= 2
		}
	}
}

// stepEnvelope steps an envelope generator
func (a *APU) stepEnvelope(env *EnvelopeGenerator) {
	if env.Start {
		env.Start = false
		env.Counter = 15
		env.Divider = env.Volume
	} else {
		if env.Divider > 0 {
			env.Divider--
		} else {
			env.Divider = env.Volume
			if env.Counter > 0 {
				env.Counter--
			} else if env.Loop {
				env.Counter = 15
			}
		}
	}
}

// stepLengthCounter steps a length counter
func (a *APU) stepLengthCounter(lc *LengthCounter) {
	if lc.Enabled && !lc.Halt && lc.Value > 0 {
		lc.Value--
	}
}

// stepSweep steps a sweep unit
func (a *APU) stepSweep(pulse *PulseChannel, sweep *SweepUnit, channel1 bool) {
	if sweep.Reload {
		sweep.Counter = sweep.Period
		sweep.Reload = false
		if sweep.Enabled && sweep.Period == 0 {
			// If period is 0, perform sweep immediately
			a.performSweep(pulse, sweep, channel1)
		}
	} else if sweep.Counter > 0 {
		sweep.Counter--
	} else {
		sweep.Counter = sweep.Period
		if sweep.Enabled {
			a.performSweep(pulse, sweep, channel1)
		}
	}
}

// performSweep performs the actual sweep calculation
func (a *APU) performSweep(pulse *PulseChannel, sweep *SweepUnit, channel1 bool) {
	change := pulse.TimerValue >> sweep.Shift
	var targetPeriod uint16

	if sweep.Negate {
		if channel1 {
			// Pulse 1 uses one's complement
			targetPeriod = pulse.TimerValue - change - 1
		} else {
			// Pulse 2 uses two's complement
			targetPeriod = pulse.TimerValue - change
		}
	} else {
		targetPeriod = pulse.TimerValue + change
	}

	// Update timer if sweep is valid
	if targetPeriod >= 8 && targetPeriod <= 0x7FF {
		pulse.TimerValue = targetPeriod
	}
}

// getPulseOutput gets the output value for a pulse channel
func (a *APU) getPulseOutput(pulse *PulseChannel) uint8 {
	if !pulse.Enabled || pulse.Length.Value == 0 {
		return 0
	}

	// Check if timer is too low or high
	if pulse.TimerValue < 8 || pulse.TimerValue > 0x7FF {
		return 0
	}

	// Check if sweep unit would mute the channel
	if a.isSweepMuting(pulse, &pulse.Sweep) {
		return 0
	}

	// Get duty cycle output
	dutyOutput := dutyCycles[pulse.DutyCycle][pulse.Sequence]
	if dutyOutput == 0 {
		return 0
	}

	// Get envelope output
	var volume uint8
	if pulse.Envelope.Constant {
		volume = pulse.Volume
	} else {
		volume = pulse.Envelope.Counter
	}

	return volume
}

// isSweepMuting checks if sweep unit would mute the channel
func (a *APU) isSweepMuting(pulse *PulseChannel, sweep *SweepUnit) bool {
	if !sweep.Enabled {
		return false
	}

	change := pulse.TimerValue >> sweep.Shift
	var targetPeriod uint16

	if sweep.Negate {
		// Calculate target period for negative sweep
		if change <= pulse.TimerValue {
			targetPeriod = pulse.TimerValue - change
		} else {
			return true // Would underflow
		}
	} else {
		// Calculate target period for positive sweep
		targetPeriod = pulse.TimerValue + change
	}

	// Mute if target period is out of valid range
	return targetPeriod < 8 || targetPeriod > 0x7FF
}

// getTriangleOutput gets the output value for the triangle channel
func (a *APU) getTriangleOutput() uint8 {
	if !a.Triangle.Enabled || a.Triangle.Length.Value == 0 || a.Triangle.LinearCounter == 0 {
		return 0
	}

	return triangleSequence[a.Triangle.Sequence]
}

// getNoiseOutput gets the output value for the noise channel
func (a *APU) getNoiseOutput() uint8 {
	if !a.Noise.Enabled || a.Noise.Length.Value == 0 {
		return 0
	}

	// Check if shift register bit 0 is set
	if a.Noise.ShiftReg&1 != 0 {
		return 0
	}

	// Get envelope output
	var volume uint8
	if a.Noise.Envelope.Constant {
		volume = a.Noise.Volume
	} else {
		volume = a.Noise.Envelope.Counter
	}

	return volume
}

// getDMCOutput gets the output value for the DMC channel
func (a *APU) getDMCOutput() uint8 {
	// DMC output is more complex, placeholder for now
	if !a.DMC.Enabled {
		return 0
	}
	return a.DMC.LoadCounter
}

// mixChannels mixes all audio channels using proper NES mixing
func (a *APU) mixChannels() float32 {
	pulse1 := a.getPulseOutput(&a.Pulse1)
	pulse2 := a.getPulseOutput(&a.Pulse2)
	triangle := a.getTriangleOutput()
	noise := a.getNoiseOutput()
	dmc := a.getDMCOutput()

	// Mix pulse channels
	pulseSum := pulse1 + pulse2
	var pulseOut float32
	if pulseSum > 0 {
		pulseOut = 95.52 / ((8128.0 / float32(pulseSum)) + 100.0)
	}

	// Mix TND channels
	tndSum := float32(triangle)/8227.0 + float32(noise)/12241.0 + float32(dmc)/22638.0
	var tndOut float32
	if tndSum > 0 {
		tndOut = 163.67 / (1.0/(tndSum) + 24.329)
	}

	// Combine outputs
	output := pulseOut + tndOut

	// Convert to range [-1.0, 1.0] with proper scaling
	normalizedOutput := output * 2.0

	// Clamp to valid range
	if normalizedOutput > 1.0 {
		normalizedOutput = 1.0
	} else if normalizedOutput < -1.0 {
		normalizedOutput = -1.0
	}

	return normalizedOutput
}

// stepLinearCounter steps the triangle's linear counter
func (a *APU) stepLinearCounter() {
	// Check reload flag
	if a.Triangle.LinearControl {
		a.Triangle.LinearCounter = a.Triangle.LinearReload
	} else if a.Triangle.LinearCounter > 0 {
		a.Triangle.LinearCounter--
	}

	// Clear reload flag if control flag is not set
	if !a.Triangle.Length.Halt {
		a.Triangle.LinearControl = false
	}
}

// frameSequencerStep performs quarter frame and half frame operations
func (a *APU) frameSequencerStep(quarter, half bool) {
	if quarter {
		a.stepEnvelopes()
		a.stepLinearCounter()
	}

	if half {
		a.stepLengthCounters()
		a.stepSweeps()
	}
}
