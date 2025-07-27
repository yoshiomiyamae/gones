package apu

// MemoryReader interface for DMC to read from memory
type MemoryReader interface {
	Read(address uint16) uint8
}

// APU represents the Audio Processing Unit
type APU struct {
	// Pulse channels
	Pulse1 PulseChannel
	Pulse2 PulseChannel

	// Triangle channel
	Triangle TriangleChannel

	// Noise channel
	Noise NoiseChannel

	// DMC channel
	DMC DMCChannel

	// Frame counter
	FrameCounter uint8
	FrameStep    int
	FrameIRQ     bool

	// Cycle counter
	Cycles uint64

	// Output buffer
	Output []float32

	// Memory interface for DMC
	Memory MemoryReader
}

// PulseChannel represents a pulse wave channel
type PulseChannel struct {
	Enabled    bool
	DutyCycle  uint8
	Volume     uint8
	Sweep      SweepUnit
	Length     LengthCounter
	Envelope   EnvelopeGenerator
	Timer      uint16
	TimerValue uint16
	Sequence   uint8
}

// TriangleChannel represents the triangle wave channel
type TriangleChannel struct {
	Enabled       bool
	LinearCounter uint8
	LinearReload  uint8
	LinearControl bool // Control flag (halt length counter / reload linear counter)
	Length        LengthCounter
	Timer         uint16
	TimerValue    uint16
	Sequence      uint8
}

// NoiseChannel represents the noise channel
type NoiseChannel struct {
	Enabled    bool
	Volume     uint8
	Length     LengthCounter
	Envelope   EnvelopeGenerator
	Timer      uint16
	TimerValue uint16
	ShiftReg   uint16
	Mode       bool
}

// DMCChannel represents the Delta Modulation Channel
type DMCChannel struct {
	Enabled        bool
	IRQEnabled     bool
	Loop           bool
	Rate           uint8
	LoadCounter    uint8
	SampleAddress  uint16
	SampleLength   uint16
	CurrentAddress uint16
	CurrentLength  uint16
	Buffer         uint8
	ShiftReg       uint8
	BitsRemaining  uint8
	Silence        bool
	SampleBuffer   uint8
	BufferEmpty    bool
}

// SweepUnit represents a sweep unit
type SweepUnit struct {
	Enabled bool
	Period  uint8
	Negate  bool
	Shift   uint8
	Reload  bool
	Counter uint8
}

// LengthCounter represents a length counter
type LengthCounter struct {
	Enabled bool
	Value   uint8
	Halt    bool
}

// EnvelopeGenerator represents an envelope generator
type EnvelopeGenerator struct {
	Start    bool
	Loop     bool
	Constant bool
	Volume   uint8
	Counter  uint8
	Divider  uint8
}

// Length counter lookup table
var lengthTable = [32]uint8{
	10, 254, 20, 2, 40, 4, 80, 6, 160, 8, 60, 10, 14, 12, 26, 14,
	12, 16, 24, 18, 48, 20, 96, 22, 192, 24, 72, 26, 16, 28, 32, 30,
}

// New creates a new APU instance
func New() *APU {
	apu := &APU{
		Output: make([]float32, 0, 4096),
	}
	apu.initializeChannels()
	return apu
}

// SetMemory sets the memory interface for DMC
func (a *APU) SetMemory(mem MemoryReader) {
	a.Memory = mem
}

// Reset resets the APU to initial state
func (a *APU) Reset() {
	a.Pulse1 = PulseChannel{}
	a.Pulse2 = PulseChannel{}
	a.Triangle = TriangleChannel{}
	a.Noise = NoiseChannel{}
	a.DMC = DMCChannel{}
	a.FrameCounter = 0
	a.FrameStep = 0
	a.FrameIRQ = false
	a.Cycles = 0
	a.initializeChannels()
}

// Step executes one APU cycle
func (a *APU) Step() {
	a.Cycles++

	// Frame counter runs at 240Hz (CPU speed / 7457.5)
	// Use more accurate timing with fractional accumulation
	if a.Cycles%7458 == 0 {
		a.stepFrameCounter()
	}

	// Step audio channels
	a.stepPulse(&a.Pulse1)
	a.stepPulse(&a.Pulse2)
	// Triangle channel steps at 1/4 rate - every 4th APU cycle (2 octaves lower)
	a.stepTriangle()
	a.stepNoise()
	a.stepDMC()

	// Generate audio sample - keep it simple
	if a.Cycles%10 == 0 {
		sample := a.mixChannels()
		a.Output = append(a.Output, sample)

		// Prevent buffer from growing too large
		if len(a.Output) > 2048 {
			// Keep only the most recent samples
			copy(a.Output, a.Output[len(a.Output)-1024:])
			a.Output = a.Output[:1024]
		}
	}
}

// stepFrameCounter steps the frame counter
func (a *APU) stepFrameCounter() {
	// 5-step mode (bit 7 set)
	if (a.FrameCounter & 0x80) != 0 {
		switch a.FrameStep {
		case 0, 2:
			a.stepEnvelopes()
			a.stepLinearCounter()
		case 1, 3:
			a.stepEnvelopes()
			a.stepLinearCounter()
			a.stepLengthCounters()
			a.stepSweeps()
		case 4:
			// Do nothing on step 4 in 5-step mode
		}
		a.FrameStep = (a.FrameStep + 1) % 5
	} else {
		// 4-step mode (default)
		switch a.FrameStep {
		case 0, 2:
			a.stepEnvelopes()
			a.stepLinearCounter()
		case 1, 3:
			a.stepEnvelopes()
			a.stepLinearCounter()
			a.stepLengthCounters()
			a.stepSweeps()
			if a.FrameStep == 3 && (a.FrameCounter&0x40) == 0 {
				a.FrameIRQ = true
			}
		}
		a.FrameStep = (a.FrameStep + 1) % 4
	}
}

// stepEnvelopes steps all envelope generators
func (a *APU) stepEnvelopes() {
	a.stepEnvelope(&a.Pulse1.Envelope)
	a.stepEnvelope(&a.Pulse2.Envelope)
	a.stepEnvelope(&a.Noise.Envelope)
}

// stepLengthCounters steps all length counters
func (a *APU) stepLengthCounters() {
	a.stepLengthCounter(&a.Pulse1.Length)
	a.stepLengthCounter(&a.Pulse2.Length)
	a.stepLengthCounter(&a.Triangle.Length)
	a.stepLengthCounter(&a.Noise.Length)
}

// stepSweeps steps all sweep units
func (a *APU) stepSweeps() {
	a.stepSweep(&a.Pulse1, &a.Pulse1.Sweep, true)
	a.stepSweep(&a.Pulse2, &a.Pulse2.Sweep, false)
}

// Channel stepping and mixing functions are implemented in channels.go

// ReadRegister reads from APU register
func (a *APU) ReadRegister(addr uint16) uint8 {
	switch addr {
	case 0x4015: // Status
		status := uint8(0)
		if a.Pulse1.Length.Value > 0 {
			status |= 0x01
		}
		if a.Pulse2.Length.Value > 0 {
			status |= 0x02
		}
		if a.Triangle.Length.Value > 0 {
			status |= 0x04
		}
		if a.Noise.Length.Value > 0 {
			status |= 0x08
		}
		if a.DMC.CurrentLength > 0 {
			status |= 0x10
		}
		if a.FrameIRQ {
			status |= 0x40
		}
		if a.DMC.IRQEnabled && a.DMC.CurrentLength == 0 {
			status |= 0x80
		}

		// Reading status register clears frame IRQ
		a.FrameIRQ = false

		return status
	}
	return 0
}

// WriteRegister writes to APU register
func (a *APU) WriteRegister(addr uint16, value uint8) {
	switch addr {
	case 0x4000, 0x4001, 0x4002, 0x4003: // Pulse 1
		a.writePulse(&a.Pulse1, addr-0x4000, value)
	case 0x4004, 0x4005, 0x4006, 0x4007: // Pulse 2
		a.writePulse(&a.Pulse2, addr-0x4004, value)
	case 0x4008, 0x4009, 0x400A, 0x400B: // Triangle
		a.writeTriangle(addr-0x4008, value)
	case 0x400C, 0x400D, 0x400E, 0x400F: // Noise
		a.writeNoise(addr-0x400C, value)
	case 0x4010, 0x4011, 0x4012, 0x4013: // DMC
		a.writeDMC(addr-0x4010, value)
	case 0x4015: // Status
		a.writeStatus(value)
	case 0x4017: // Frame counter
		a.writeFrameCounter(value)
	}
}

// Register write functions are implemented in registers.go
