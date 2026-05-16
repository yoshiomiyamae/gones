// Package gui — audio device initialisation and per-frame sample queueing.
//
// The audio path is mono APU samples → SDL output queue. SDL may negotiate a
// different format/channel count than requested (especially on Windows
// WASAPI), so initAudio records what it actually got and queueAudio fans
// each mono sample out to every channel of one output frame.
package gui

import (
	"fmt"
	"math"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/yoshiomiyamaegones/pkg/logger"
)

// Audio constants
const (
	AudioSampleRate = 44100
	AudioBufferSize = 1024             // Standard buffer size
	AudioChannels   = 1                // Mono
	AudioFormat     = sdl.AUDIO_F32LSB // 32-bit float, little-endian
)

// initAudio initializes SDL audio device and callback
func (g *NESGUI) initAudio() error {
	// List available audio drivers for debugging
	numDrivers := sdl.GetNumAudioDrivers()
	logger.LogInfo("Available audio drivers (%d):", numDrivers)
	for i := 0; i < numDrivers; i++ {
		driverName := sdl.GetAudioDriver(i)
		logger.LogInfo("  %d: %s", i, driverName)
	}

	currentDriver := sdl.GetCurrentAudioDriver()
	logger.LogInfo("Current audio driver: %s", currentDriver)

	// Define audio specification with callback
	want := &sdl.AudioSpec{
		Freq:     AudioSampleRate,
		Format:   AudioFormat,
		Channels: AudioChannels,
		Samples:  AudioBufferSize,
	}

	logger.LogInfo("Requesting audio format: %dHz, %d channels, format 0x%x, buffer %d",
		want.Freq, want.Channels, want.Format, want.Samples)

	// Open audio device
	var have sdl.AudioSpec
	device, err := sdl.OpenAudioDevice("", false, want, &have, sdl.AUDIO_ALLOW_ANY_CHANGE)
	if err != nil {
		// Try with 16-bit format for better Windows compatibility
		logger.LogInfo("Retrying with 16-bit audio format...")
		want.Format = sdl.AUDIO_S16LSB
		device, err = sdl.OpenAudioDevice("", false, want, &have, sdl.AUDIO_ALLOW_ANY_CHANGE)
		if err != nil {
			return fmt.Errorf("failed to open audio device: %v", err)
		}
	}

	g.audioDevice = device
	g.audioSpec = &have

	logger.LogInfo("Audio initialized: %dHz, %d channels, format 0x%x, buffer size %d",
		have.Freq, have.Channels, have.Format, have.Samples)

	// IMPORTANT: Check if actual sample rate differs from requested
	if have.Freq != AudioSampleRate {
		logger.LogInfo("WARNING: Requested %d Hz but got %d Hz - audio pitch will be wrong!",
			AudioSampleRate, have.Freq)
	}

	// Start audio playback
	sdl.PauseAudioDevice(device, false)

	return nil
}

// queueAudio queues APU audio samples to SDL.
//
// The APU is a mono source but the audio device may have been opened with more
// channels (WASAPI on Windows often refuses mono and gives us stereo). Each
// APU sample is duplicated to every channel of one output frame; without this
// SDL would interpret consecutive mono samples as L/R pairs of a stereo frame,
// playing audio at twice the intended rate (one octave high) with severe
// inter-sample noise from the L/R mismatch.
func (g *NESGUI) queueAudio() {
	if g.audioDevice == 0 {
		return
	}

	// Turbo mode is allowed to queue audio normally. Samples generated
	// faster than realtime play back at the same elevated rate (chipmunk
	// pitch, occasional buffer-full drops), which the user opted into by
	// hitting Tab — accepting "garbled but audible" over silence. The
	// maxBytes guard below still caps queue growth so SDL doesn't grow
	// unbounded during long turbo bursts.

	apuOutput := g.nes.APU.Output
	if len(apuOutput) == 0 {
		return
	}
	defer func() { g.nes.APU.Output = g.nes.APU.Output[:0] }()

	var bytesPerSample int
	switch g.audioSpec.Format {
	case sdl.AUDIO_F32LSB:
		bytesPerSample = 4
	case sdl.AUDIO_S16LSB:
		bytesPerSample = 2
	default:
		return
	}
	channels := int(g.audioSpec.Channels)
	bytesPerFrame := bytesPerSample * channels

	// Cap queued audio at ~2 device buffers' worth to keep latency bounded
	// while still tolerating short emulation hiccups. Uses the *actual* buffer
	// size SDL gave us (the requested AudioBufferSize is often downgraded).
	maxBytes := uint32(int(g.audioSpec.Samples) * bytesPerFrame * 2)
	if sdl.GetQueuedAudioSize(g.audioDevice) >= maxBytes {
		return
	}

	needed := len(apuOutput) * bytesPerFrame
	if cap(g.audioBuf) < needed {
		g.audioBuf = make([]byte, needed)
	} else {
		g.audioBuf = g.audioBuf[:needed]
	}

	// 2× gain brings the APU's post-HPF peak (~±0.5) up near full scale,
	// matching the recorder so what you hear is what you record.
	switch g.audioSpec.Format {
	case sdl.AUDIO_F32LSB:
		for i, sample := range apuOutput {
			v := sample * 2.0
			if v > 1.0 {
				v = 1.0
			} else if v < -1.0 {
				v = -1.0
			}
			bits := math.Float32bits(v)
			base := i * bytesPerFrame
			for ch := 0; ch < channels; ch++ {
				off := base + ch*4
				g.audioBuf[off+0] = byte(bits)
				g.audioBuf[off+1] = byte(bits >> 8)
				g.audioBuf[off+2] = byte(bits >> 16)
				g.audioBuf[off+3] = byte(bits >> 24)
			}
		}
	case sdl.AUDIO_S16LSB:
		for i, sample := range apuOutput {
			sample *= 2.0
			if sample > 1.0 {
				sample = 1.0
			} else if sample < -1.0 {
				sample = -1.0
			}
			intSample := int16(sample * 32767)
			base := i * bytesPerFrame
			for ch := 0; ch < channels; ch++ {
				off := base + ch*2
				g.audioBuf[off+0] = byte(intSample)
				g.audioBuf[off+1] = byte(intSample >> 8)
			}
		}
	}

	sdl.QueueAudio(g.audioDevice, g.audioBuf)
}
