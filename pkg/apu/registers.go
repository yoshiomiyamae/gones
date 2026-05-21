package apu

// writePulse handles pulse channel register writes
func (a *APU) writePulse(pulse *PulseChannel, reg uint16, value uint8) {
	switch reg {
	case 0: // $4000/$4004 - Duty, envelope, volume
		pulse.DutyCycle = (value >> 6) & 0x03
		pulse.Length.Halt = (value & 0x20) != 0
		pulse.Envelope.Loop = (value & 0x20) != 0
		pulse.Envelope.Constant = (value & 0x10) != 0
		pulse.Volume = value & 0x0F
		pulse.Envelope.Volume = value & 0x0F
		
	case 1: // $4001/$4005 - Sweep
		pulse.Sweep.Enabled = (value & 0x80) != 0
		pulse.Sweep.Period = (value >> 4) & 0x07
		pulse.Sweep.Negate = (value & 0x08) != 0
		pulse.Sweep.Shift = value & 0x07
		pulse.Sweep.Reload = true
		
	case 2: // $4002/$4006 - Timer low
		pulse.TimerValue = (pulse.TimerValue & 0xFF00) | uint16(value)
		
	case 3: // $4003/$4007 - Length counter, timer high
		pulse.TimerValue = (pulse.TimerValue & 0x00FF) | ((uint16(value) & 0x07) << 8)
		if pulse.Enabled {
			pulse.Length.Value = lengthTable[(value>>3)&0x1F]
		}
		pulse.Envelope.Start = true
		pulse.Sequence = 0
		// Debug: Log timer value updates
		// logger.LogInfo("Pulse timer set: TimerValue=%d, frequency=%.2f Hz", 
		//	pulse.TimerValue, 1789773.0/(16.0*float64(pulse.TimerValue+1)))
	}
}

// writeTriangle handles triangle channel register writes
func (a *APU) writeTriangle(reg uint16, value uint8) {
	switch reg {
	case 0: // $4008 - Linear counter control and reload
		a.Triangle.Length.Halt = (value & 0x80) != 0
		a.Triangle.LinearReload = value & 0x7F
		
	case 1: // $4009 - Unused
		// Unused register
		
	case 2: // $400A - Timer low
		a.Triangle.TimerValue = (a.Triangle.TimerValue & 0xFF00) | uint16(value)
		
	case 3: // $400B - Length counter, timer high
		a.Triangle.TimerValue = (a.Triangle.TimerValue & 0x00FF) | ((uint16(value) & 0x07) << 8)

		if a.Triangle.Enabled {
			a.Triangle.Length.Value = lengthTable[(value>>3)&0x1F]
		}
		// $400B sets only the reload flag — the control flag is a separate
		// piece of state owned by $4008.
		a.Triangle.LinearReloadFlag = true
	}
}

// writeNoise handles noise channel register writes
func (a *APU) writeNoise(reg uint16, value uint8) {
	switch reg {
	case 0: // $400C - Envelope, volume
		a.Noise.Length.Halt = (value & 0x20) != 0
		a.Noise.Envelope.Loop = (value & 0x20) != 0
		a.Noise.Envelope.Constant = (value & 0x10) != 0
		a.Noise.Volume = value & 0x0F
		a.Noise.Envelope.Volume = value & 0x0F
		
	case 1: // $400D - Unused
		// Unused register
		
	case 2: // $400E - Period, mode
		a.Noise.Mode = (value & 0x80) != 0
		periodIndex := value & 0x0F
		a.Noise.TimerValue = noisePeriods[periodIndex]
		
	case 3: // $400F - Length counter
		if a.Noise.Enabled {
			a.Noise.Length.Value = lengthTable[(value>>3)&0x1F]
		}
		a.Noise.Envelope.Start = true
	}
}

// writeDMC handles DMC channel register writes
func (a *APU) writeDMC(reg uint16, value uint8) {
	switch reg {
	case 0: // $4010 - Rate, loop, IRQ
		a.DMC.IRQEnabled = (value & 0x80) != 0
		// Clearing the IRQ-enable bit also acknowledges any latched DMC IRQ.
		if !a.DMC.IRQEnabled {
			a.DMC.InterruptFlag = false
		}
		a.DMC.Loop = (value & 0x40) != 0
		a.DMC.Rate = value & 0x0F
		// Set DMC timer based on rate
		// dmcRates[a.DMC.Rate] contains the period in CPU cycles
		
	case 1: // $4011 - Load counter
		a.DMC.LoadCounter = value & 0x7F
		
	case 2: // $4012 - Sample address
		a.DMC.SampleAddress = 0xC000 + (uint16(value) * 64)

	case 3: // $4013 - Sample length
		// Just sets the latch; playback is started by a $4015 write
		// with bit 4 set while CurrentLength is 0.
		a.DMC.SampleLength = (uint16(value) * 16) + 1
	}
}

// writeStatus handles status register writes
func (a *APU) writeStatus(value uint8) {
	// Enable/disable channels
	a.Pulse1.Enabled = (value & 0x01) != 0
	a.Pulse2.Enabled = (value & 0x02) != 0
	a.Triangle.Enabled = (value & 0x04) != 0
	a.Noise.Enabled = (value & 0x08) != 0
	a.DMC.Enabled = (value & 0x10) != 0
	// Any $4015 write acknowledges a pending DMC IRQ.
	a.DMC.InterruptFlag = false

	// Clear length counters for disabled channels
	if !a.Pulse1.Enabled {
		a.Pulse1.Length.Value = 0
	}
	if !a.Pulse2.Enabled {
		a.Pulse2.Length.Value = 0
	}
	if !a.Triangle.Enabled {
		a.Triangle.Length.Value = 0
	}
	if !a.Noise.Enabled {
		a.Noise.Length.Value = 0
	}
	// DMC start/stop: clearing bit 4 zeros bytes-remaining (silences
	// after the current byte plays out); setting bit 4 restarts the
	// sample only if it isn't already active. NESdev's APU DMC page —
	// without this initial load CurrentLength stays 0 and the channel
	// only emits the click from any $4011 write.
	if !a.DMC.Enabled {
		a.DMC.CurrentLength = 0
	} else if a.DMC.CurrentLength == 0 {
		a.DMC.CurrentLength = a.DMC.SampleLength
		a.DMC.CurrentAddress = a.DMC.SampleAddress
	}
}

// writeFrameCounter handles frame counter register writes
func (a *APU) writeFrameCounter(value uint8) {
	a.FrameCounter = value
	
	// Reset frame counter
	a.FrameStep = 0
	
	// If 5-step mode is set, step immediately
	if (value & 0x80) != 0 {
		a.stepEnvelopes()
		a.stepLengthCounters()
		a.stepSweeps()
	}
	
	// Clear frame IRQ if inhibit flag is set
	if (value & 0x40) != 0 {
		a.FrameIRQ = false
	}
}

// initializeChannels initializes channel default values
func (a *APU) initializeChannels() {
	// Initialize noise shift register to 1
	a.Noise.ShiftReg = 1
	
	// Initialize envelope generators
	a.Pulse1.Envelope.Volume = 15
	a.Pulse2.Envelope.Volume = 15
	a.Noise.Envelope.Volume = 15
	
	// Initialize length counters
	a.Pulse1.Length.Enabled = true
	a.Pulse2.Length.Enabled = true
	a.Triangle.Length.Enabled = true
	a.Noise.Length.Enabled = true
	
	// Initialize DMC
	a.DMC.BufferEmpty = true
	a.DMC.LoadCounter = 0
	// Start the rate counter at a full period so the sample unit doesn't
	// clock on the very first cycle.
	a.DMC.Timer = dmcRates[a.DMC.Rate&0x0F] - 1
}

// Helper function to get frequency from timer value
func getFrequency(timerValue uint16) float32 {
	if timerValue == 0 {
		return 0
	}
	// NES CPU clock is ~1.789773 MHz
	// APU timer frequency = CPU_FREQ / (16 * (timer + 1))
	cpuFreq := float32(1789773)
	return cpuFreq / (16.0 * float32(timerValue+1))
}

// Helper function to get period from frequency
func getPeriod(frequency float32) uint16 {
	if frequency == 0 {
		return 0
	}
	// Period = (CPU_FREQ / (16 * frequency)) - 1
	cpuFreq := float32(1789773)
	period := (cpuFreq / (16.0 * frequency)) - 1
	if period < 0 {
		return 0
	}
	if period > 0x7FF {
		return 0x7FF
	}
	return uint16(period)
}