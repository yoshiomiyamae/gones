package gui

import (
	"encoding/binary"
	"fmt"
	"os"
	"time"

	"github.com/yoshiomiyamaegones/pkg/logger"
	"github.com/yoshiomiyamaegones/pkg/nes"
)

// wavRecorder streams APU samples to a mono 16-bit PCM WAV file. The RIFF
// header is written with placeholder sizes up front; Close seeks back and
// patches them so the file remains valid no matter when the user stops.
//
// Samples are written with the same 2× gain queueAudio applies for SDL —
// what you hear equals what's recorded. If FilterEnabled is off, the raw
// mixer is unipolar 0..1 and the recording shows the DC bias (correct,
// but unusual for a WAV).
type wavRecorder struct {
	f          *os.File
	dataBytes  uint32
	sampleRate uint32
	path       string
	buf        []byte // reused across WriteSamples calls — ~88KB/s of GC otherwise
}

const wavHeaderSize = 44

func newWAVRecorder(path string, sampleRate uint32) (*wavRecorder, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	r := &wavRecorder{f: f, sampleRate: sampleRate, path: path}
	if err := r.writeHeader(); err != nil {
		f.Close()
		os.Remove(path)
		return nil, err
	}
	return r, nil
}

func (r *wavRecorder) writeHeader() error {
	var hdr [wavHeaderSize]byte
	copy(hdr[0:], "RIFF")
	// hdr[4:8] = file size - 8 (patched on Close)
	copy(hdr[8:], "WAVE")
	copy(hdr[12:], "fmt ")
	binary.LittleEndian.PutUint32(hdr[16:], 16)             // fmt chunk size
	binary.LittleEndian.PutUint16(hdr[20:], 1)              // PCM
	binary.LittleEndian.PutUint16(hdr[22:], 1)              // channels (mono)
	binary.LittleEndian.PutUint32(hdr[24:], r.sampleRate)   // sample rate
	binary.LittleEndian.PutUint32(hdr[28:], r.sampleRate*2) // byte rate (mono * 16-bit)
	binary.LittleEndian.PutUint16(hdr[32:], 2)              // block align
	binary.LittleEndian.PutUint16(hdr[34:], 16)             // bits per sample
	copy(hdr[36:], "data")
	// hdr[40:44] = data size (patched on Close)
	_, err := r.f.Write(hdr[:])
	return err
}

// WriteSamples writes APU samples as signed 16-bit PCM with the same 2×
// gain queueAudio uses. The mixer's HPF already centers the signal, so no
// DC removal is needed here.
func (r *wavRecorder) WriteSamples(samples []float32) error {
	if len(samples) == 0 {
		return nil
	}
	need := len(samples) * 2
	if cap(r.buf) < need {
		r.buf = make([]byte, need)
	} else {
		r.buf = r.buf[:need]
	}
	for i, s := range samples {
		v := s * 2.0
		if v > 1.0 {
			v = 1.0
		} else if v < -1.0 {
			v = -1.0
		}
		iv := int16(v * 32767)
		r.buf[i*2] = byte(iv)
		r.buf[i*2+1] = byte(iv >> 8)
	}
	n, err := r.f.Write(r.buf)
	r.dataBytes += uint32(n)
	return err
}

// Close patches the two size fields in the RIFF header and closes the file.
func (r *wavRecorder) Close() error {
	defer r.f.Close()
	if _, err := r.f.Seek(4, 0); err != nil {
		return err
	}
	if err := binary.Write(r.f, binary.LittleEndian, uint32(36)+r.dataBytes); err != nil {
		return err
	}
	if _, err := r.f.Seek(40, 0); err != nil {
		return err
	}
	return binary.Write(r.f, binary.LittleEndian, r.dataBytes)
}

// recordingPath builds the WAV path: alongside the ROM with a timestamp
// suffix so successive recordings don't clobber each other. Falls back to
// the working directory when no ROM is loaded.
func (g *NESGUI) recordingPath() string {
	ts := time.Now().Format("20060102_150405")
	if g.romPath == "" {
		return fmt.Sprintf("gones-%s.wav", ts)
	}
	return nes.CompanionFile(g.romPath, "."+ts+".wav")
}

// toggleRecording starts a new recording or finalizes the current one.
// Ctrl+E is the bound hotkey; F-keys are reserved for save states.
func (g *NESGUI) toggleRecording() {
	if g.recorder != nil {
		path := g.recorder.path
		bytes := g.recorder.dataBytes
		if err := g.recorder.Close(); err != nil {
			logger.LogError("Recording: close failed: %v", err)
		} else {
			seconds := float64(bytes) / float64(g.recorder.sampleRate*2)
			logger.LogInfo("Recording stopped: %s (%.2fs)", path, seconds)
		}
		g.recorder = nil
		return
	}
	path := g.recordingPath()
	rec, err := newWAVRecorder(path, AudioSampleRate)
	if err != nil {
		logger.LogError("Recording: open failed: %v", err)
		return
	}
	g.recorder = rec
	logger.LogInfo("Recording started: %s", path)
}
