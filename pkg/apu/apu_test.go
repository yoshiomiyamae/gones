package apu

import (
	"testing"
	"math"
)

// createTestAPU creates an APU instance for testing
func createTestAPU() *APU {
	apu := New()
	apu.Reset()
	return apu
}

// Test APU creation and reset
func TestAPUCreation(t *testing.T) {
	apu := createTestAPU()
	
	if apu == nil {
		t.Error("APU should not be nil")
	}
	
	// Check initial state
	if apu.Cycles != 0 {
		t.Errorf("Expected cycles=0, got %d", apu.Cycles)
	}
	if apu.FrameStep != 0 {
		t.Errorf("Expected frame step=0, got %d", apu.FrameStep)
	}
	if apu.FrameIRQ {
		t.Error("Frame IRQ should be false initially")
	}
}

// Test pulse channel register writes
func TestPulseChannelRegisters(t *testing.T) {
	apu := createTestAPU()
	
	// Test Pulse 1 duty cycle and volume
	apu.WriteRegister(0x4000, 0xBF) // Duty=10, Envelope loop, Constant volume, Volume=15
	
	if apu.Pulse1.DutyCycle != 2 {
		t.Errorf("Expected duty cycle=2, got %d", apu.Pulse1.DutyCycle)
	}
	if !apu.Pulse1.Length.Halt {
		t.Error("Length halt should be true")
	}
	if !apu.Pulse1.Envelope.Constant {
		t.Error("Envelope constant should be true")
	}
	if apu.Pulse1.Volume != 15 {
		t.Errorf("Expected volume=15, got %d", apu.Pulse1.Volume)
	}
	
	// Test sweep register
	apu.WriteRegister(0x4001, 0x88) // Enabled, period=0, negate=true, shift=0
	
	if !apu.Pulse1.Sweep.Enabled {
		t.Error("Sweep should be enabled")
	}
	if apu.Pulse1.Sweep.Period != 0 {
		t.Errorf("Expected sweep period=0, got %d", apu.Pulse1.Sweep.Period)
	}
	if !apu.Pulse1.Sweep.Negate {
		t.Error("Sweep negate should be true")
	}
	
	// Test timer
	apu.WriteRegister(0x4002, 0x55) // Timer low
	apu.WriteRegister(0x4003, 0x12) // Length=4, Timer high=2
	
	expectedTimer := uint16(0x255)
	if apu.Pulse1.TimerValue != expectedTimer {
		t.Errorf("Expected timer=%04X, got %04X", expectedTimer, apu.Pulse1.TimerValue)
	}
}

// Test triangle channel registers
func TestTriangleChannelRegisters(t *testing.T) {
	apu := createTestAPU()
	
	// Enable triangle channel first
	apu.WriteRegister(0x4015, 0x04) // Enable triangle
	
	// Test linear counter
	apu.WriteRegister(0x4008, 0x81) // Control flag set, counter=1
	
	if !apu.Triangle.Length.Halt {
		t.Error("Triangle length halt should be true")
	}
	if apu.Triangle.LinearCounter != 0 {
		t.Errorf("Expected linear counter=0, got %d", apu.Triangle.LinearCounter)
	}
	
	// Test timer
	apu.WriteRegister(0x400A, 0xAA) // Timer low
	apu.WriteRegister(0x400B, 0x13) // Length=4, Timer high=3
	
	expectedTimer := uint16(0x3AA)
	if apu.Triangle.TimerValue != expectedTimer {
		t.Errorf("Expected timer=%04X, got %04X", expectedTimer, apu.Triangle.TimerValue)
	}
}

// Test noise channel registers
func TestNoiseChannelRegisters(t *testing.T) {
	apu := createTestAPU()
	
	// Test envelope
	apu.WriteRegister(0x400C, 0x3A) // Loop, Constant, Volume=10
	
	if !apu.Noise.Length.Halt {
		t.Error("Noise length halt should be true")
	}
	if !apu.Noise.Envelope.Constant {
		t.Error("Noise envelope constant should be true")
	}
	if apu.Noise.Volume != 10 {
		t.Errorf("Expected volume=10, got %d", apu.Noise.Volume)
	}
	
	// Test period and mode
	apu.WriteRegister(0x400E, 0x8F) // Mode=1, Period=15
	
	if !apu.Noise.Mode {
		t.Error("Noise mode should be true")
	}
	if apu.Noise.TimerValue != noisePeriods[15] {
		t.Errorf("Expected timer=%d, got %d", noisePeriods[15], apu.Noise.TimerValue)
	}
}

// Test status register
func TestStatusRegister(t *testing.T) {
	apu := createTestAPU()
	
	// Enable all channels
	apu.WriteRegister(0x4015, 0x1F) // Enable all channels
	
	if !apu.Pulse1.Enabled {
		t.Error("Pulse 1 should be enabled")
	}
	if !apu.Pulse2.Enabled {
		t.Error("Pulse 2 should be enabled")
	}
	if !apu.Triangle.Enabled {
		t.Error("Triangle should be enabled")
	}
	if !apu.Noise.Enabled {
		t.Error("Noise should be enabled")
	}
	if !apu.DMC.Enabled {
		t.Error("DMC should be enabled")
	}
	
	// Disable channels
	apu.WriteRegister(0x4015, 0x00)
	
	if apu.Pulse1.Enabled {
		t.Error("Pulse 1 should be disabled")
	}
	if apu.Triangle.Enabled {
		t.Error("Triangle should be disabled")
	}
}

// Test envelope stepping
func TestEnvelopeGenerator(t *testing.T) {
	apu := createTestAPU()
	
	// Set up pulse channel with envelope
	apu.WriteRegister(0x4000, 0x08) // No constant volume, volume=8
	apu.WriteRegister(0x4003, 0x08) // Trigger envelope start
	
	// Envelope should start at 0
	if apu.Pulse1.Envelope.Counter != 0 {
		t.Errorf("Expected envelope counter=0, got %d", apu.Pulse1.Envelope.Counter)
	}
	
	// Step envelope multiple times
	for i := 0; i < 16; i++ {
		apu.stepEnvelope(&apu.Pulse1.Envelope)
	}
	
	// Should be at 14 after one complete cycle
	if apu.Pulse1.Envelope.Counter != 14 {
		t.Errorf("Expected envelope counter=14, got %d", apu.Pulse1.Envelope.Counter)
	}
}

// Test length counter
func TestLengthCounter(t *testing.T) {
	apu := createTestAPU()
	
	// Enable pulse channel and set length
	apu.WriteRegister(0x4015, 0x01) // Enable pulse 1
	apu.WriteRegister(0x4003, 0x08) // Length counter = lengthTable[1] = 254
	
	expectedLength := lengthTable[1]
	if apu.Pulse1.Length.Value != expectedLength {
		t.Errorf("Expected length=%d, got %d", expectedLength, apu.Pulse1.Length.Value)
	}
	
	// Step length counter
	originalValue := apu.Pulse1.Length.Value
	apu.stepLengthCounter(&apu.Pulse1.Length)
	
	if apu.Pulse1.Length.Value != originalValue-1 {
		t.Errorf("Expected length=%d, got %d", originalValue-1, apu.Pulse1.Length.Value)
	}
}

// Test sweep unit
func TestSweepUnit(t *testing.T) {
	apu := createTestAPU()
	
	// Set up pulse channel with sweep
	apu.WriteRegister(0x4001, 0x81) // Enable sweep, period=0, negate=false, shift=1
	apu.WriteRegister(0x4002, 0x00) // Timer low = 0
	apu.WriteRegister(0x4003, 0x01) // Timer high = 1, so timer = 0x100
	
	originalTimer := apu.Pulse1.TimerValue
	
	// Step sweep
	apu.stepSweep(&apu.Pulse1, &apu.Pulse1.Sweep, true)
	
	// Timer should increase (sweep adds)
	if apu.Pulse1.TimerValue <= originalTimer {
		t.Errorf("Expected timer to increase from %d, got %d", originalTimer, apu.Pulse1.TimerValue)
	}
}

// Test frame counter
func TestFrameCounter(t *testing.T) {
	apu := createTestAPU()
	
	// Test 4-step mode
	apu.WriteRegister(0x4017, 0x00) // 4-step mode, no IRQ inhibit
	
	if apu.FrameStep != 0 {
		t.Errorf("Expected frame step=0, got %d", apu.FrameStep)
	}
	
	// Test 5-step mode
	apu.WriteRegister(0x4017, 0x80) // 5-step mode
	
	if apu.FrameStep != 0 {
		t.Errorf("Expected frame step=0 after write, got %d", apu.FrameStep)
	}
}

// Test channel output
func TestChannelOutput(t *testing.T) {
	apu := createTestAPU()
	
	// Enable pulse 1 and set up for output
	apu.WriteRegister(0x4015, 0x01) // Enable pulse 1
	apu.WriteRegister(0x4000, 0x5F) // Duty=01 (25%), Constant volume, max volume
	apu.WriteRegister(0x4002, 0x00) // Timer low
	apu.WriteRegister(0x4003, 0x01) // Timer high, length counter
	
	// Step pulse to advance sequence to position 1 (where duty cycle outputs 1)
	apu.stepPulse(&apu.Pulse1)
	
	// Get output
	output := apu.getPulseOutput(&apu.Pulse1)
	
	// Should have some output
	if output == 0 {
		t.Error("Expected non-zero output from enabled pulse channel")
	}
	
	// Disable channel
	apu.WriteRegister(0x4015, 0x00)
	output = apu.getPulseOutput(&apu.Pulse1)
	
	if output != 0 {
		t.Error("Expected zero output from disabled pulse channel")
	}
}

// Test audio mixing
func TestAudioMixing(t *testing.T) {
	apu := createTestAPU()
	
	// Enable all channels with some output
	apu.WriteRegister(0x4015, 0x1F) // Enable all
	
	// Set up pulse channels
	apu.WriteRegister(0x4000, 0x1F) // Pulse 1: max volume
	apu.WriteRegister(0x4004, 0x1F) // Pulse 2: max volume
	
	// Set up triangle
	apu.WriteRegister(0x4008, 0x81) // Triangle: linear counter
	
	// Set up noise
	apu.WriteRegister(0x400C, 0x1F) // Noise: max volume
	
	// Get mixed output
	sample := apu.mixChannels()
	
	// Should be in valid range [-1.0, 1.0]
	if sample < -1.0 || sample > 1.0 {
		t.Errorf("Mixed sample out of range [-1,1]: %f", sample)
	}
}

// Test frequency calculation helper
func TestFrequencyCalculation(t *testing.T) {
	// Test known frequency
	freq := getFrequency(0x100)
	expectedFreq := float32(1789773) / (16.0 * (0x100 + 1))
	
	if math.Abs(float64(freq - expectedFreq)) > 0.001 {
		t.Errorf("Expected frequency %f, got %f", expectedFreq, freq)
	}
	
	// Test zero timer
	freq = getFrequency(0)
	if freq != 0 {
		t.Errorf("Expected frequency 0 for timer 0, got %f", freq)
	}
}

// Test period calculation helper
func TestPeriodCalculation(t *testing.T) {
	// Test known period
	period := getPeriod(440.0) // A4 note
	
	// Should be reasonable value
	if period == 0 || period > 0x7FF {
		t.Errorf("Period out of range for 440Hz: %d", period)
	}
	
	// Test zero frequency
	period = getPeriod(0)
	if period != 0 {
		t.Errorf("Expected period 0 for frequency 0, got %d", period)
	}
}

// Test APU step function
func TestAPUStep(t *testing.T) {
	apu := createTestAPU()
	
	initialCycles := apu.Cycles
	
	// Step APU
	apu.Step()
	
	// Cycles should increment
	if apu.Cycles != initialCycles + 1 {
		t.Errorf("Expected cycles=%d, got %d", initialCycles + 1, apu.Cycles)
	}
	
	// Output buffer should have sample
	if len(apu.Output) == 0 {
		t.Error("Expected output buffer to have sample after step")
	}
}
