package mapper

import "testing"

// TestFME7AudioRegisters verifies the register-write protocol shapes
// the channel state correctly: low+high tone bytes form the 12-bit
// period, R7 enables/disables channels via active-low bits, and R8-RA
// set volume.
func TestFME7AudioRegisters(t *testing.T) {
	var a fme7Audio

	// Channel A period = $123 (low=$23, high=$01).
	a.writeAudioSelect(0)
	a.writeAudioData(0x23)
	a.writeAudioSelect(1)
	a.writeAudioData(0x01)
	if a.channels[0].period != 0x123 {
		t.Errorf("channel A period: want $123 got $%X", a.channels[0].period)
	}

	// R7: enable A and C, disable B. Active-low → bits 0,2 clear, bit 1 set.
	a.writeAudioSelect(7)
	a.writeAudioData(0x02)
	if !a.channels[0].enable || a.channels[1].enable || !a.channels[2].enable {
		t.Errorf("R7 enable: A=%v B=%v C=%v, want true,false,true",
			a.channels[0].enable, a.channels[1].enable, a.channels[2].enable)
	}

	// Channel B volume = 12, channel C volume = 7.
	a.writeAudioSelect(9)
	a.writeAudioData(0x0C)
	a.writeAudioSelect(0xA)
	a.writeAudioData(0x07)
	if a.channels[1].volume != 12 || a.channels[2].volume != 7 {
		t.Errorf("volumes: B=%d C=%d, want 12,7", a.channels[1].volume, a.channels[2].volume)
	}
}

// TestFME7AudioGeneratesSquareWave confirms that ticking a channel
// produces a 0/1 toggle at the expected period (period * 16 CPU
// cycles between toggles).
func TestFME7AudioGeneratesSquareWave(t *testing.T) {
	var a fme7Audio
	// Channel A, period 10, volume 15, A enabled.
	a.writeAudioSelect(0)
	a.writeAudioData(10)
	a.writeAudioSelect(1)
	a.writeAudioData(0)
	a.writeAudioSelect(7)
	a.writeAudioData(0x06) // A enabled (bit 0 clear), B/C disabled
	a.writeAudioSelect(8)
	a.writeAudioData(15)

	// Tick less than period*16 = 160 cycles — output should still be 0.
	a.tick(159)
	if a.channels[0].output != 0 {
		t.Errorf("output toggled too early: want 0, got %d", a.channels[0].output)
	}
	// One more cycle crosses the threshold.
	a.tick(1)
	if a.channels[0].output != 1 {
		t.Errorf("output didn't toggle at period*16: got %d", a.channels[0].output)
	}

	// Sample should now be non-zero (volume 15 mapped through the LUT,
	// scaled by 0.25 mixer factor).
	if a.sample() <= 0 {
		t.Errorf("sample() with A on full-volume high: got %f, want > 0", a.sample())
	}
}

// TestFME7AudioDisabledChannelsMute verifies the R7 mixer mask
// silences a channel even when its tone generator is producing a 1.
func TestFME7AudioDisabledChannelsMute(t *testing.T) {
	var a fme7Audio
	a.writeAudioSelect(8)
	a.writeAudioData(15)
	a.channels[0].output = 1
	a.channels[0].enable = false
	if a.sample() != 0 {
		t.Errorf("disabled channel still mixed: got %f", a.sample())
	}
}
