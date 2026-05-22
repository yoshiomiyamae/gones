package main

import (
	"bytes"
	"encoding/binary"
	"math"
	"math/cmplx"
	"os"
	"path/filepath"
	"testing"
)

// writeWAV writes a minimal 16-bit PCM WAV file (mono/stereo) for tests.
func writeWAV(t *testing.T, path string, channels uint16, sampleRate uint32, samples []int16) {
	t.Helper()
	var buf bytes.Buffer
	dataSize := uint32(len(samples) * 2)
	blockAlign := channels * 2
	byteRate := sampleRate * uint32(blockAlign)

	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(36+dataSize))
	buf.WriteString("WAVE")

	buf.WriteString("fmt ")
	binary.Write(&buf, binary.LittleEndian, uint32(16))
	binary.Write(&buf, binary.LittleEndian, uint16(1)) // PCM
	binary.Write(&buf, binary.LittleEndian, channels)
	binary.Write(&buf, binary.LittleEndian, sampleRate)
	binary.Write(&buf, binary.LittleEndian, byteRate)
	binary.Write(&buf, binary.LittleEndian, blockAlign)
	binary.Write(&buf, binary.LittleEndian, uint16(16)) // bits per sample

	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, dataSize)
	for _, s := range samples {
		binary.Write(&buf, binary.LittleEndian, s)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("write wav: %v", err)
	}
}

func TestLoadWAVMono(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mono.wav")
	samples := []int16{0, 16384, -16384, 32767, -32768}
	writeWAV(t, path, 1, 44100, samples)

	w, err := loadWAV(path)
	if err != nil {
		t.Fatalf("loadWAV: %v", err)
	}
	if w.channels != 1 || w.sampleRate != 44100 || w.bitsPer != 16 {
		t.Errorf("header = ch%d/%dHz/%dbit, want 1/44100/16", w.channels, w.sampleRate, w.bitsPer)
	}
	if len(w.samples) != len(samples) {
		t.Fatalf("got %d samples, want %d", len(w.samples), len(samples))
	}
	// 16384/32768 = 0.5 normalized.
	if math.Abs(w.samples[1]-0.5) > 1e-6 {
		t.Errorf("sample[1] = %f, want 0.5", w.samples[1])
	}
}

func TestLoadWAVStereoAveragedToMono(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stereo.wav")
	// Two frames: L/R = (32767,-32767) averages ~0; (16384,16384) -> 0.5.
	writeWAV(t, path, 2, 48000, []int16{32767, -32767, 16384, 16384})

	w, err := loadWAV(path)
	if err != nil {
		t.Fatalf("loadWAV: %v", err)
	}
	if w.channels != 2 || w.sampleRate != 48000 {
		t.Errorf("header = ch%d/%dHz, want 2/48000", w.channels, w.sampleRate)
	}
	if len(w.samples) != 2 {
		t.Fatalf("got %d frames, want 2", len(w.samples))
	}
	if math.Abs(w.samples[0]) > 1e-3 {
		t.Errorf("frame0 average = %f, want ~0", w.samples[0])
	}
	if math.Abs(w.samples[1]-0.5) > 1e-6 {
		t.Errorf("frame1 average = %f, want 0.5", w.samples[1])
	}
}

func TestLoadWAVErrors(t *testing.T) {
	dir := t.TempDir()

	// Missing file.
	if _, err := loadWAV(filepath.Join(dir, "nope.wav")); err == nil {
		t.Error("missing file should error")
	}

	// Not a RIFF/WAVE container.
	bad := filepath.Join(dir, "bad.wav")
	os.WriteFile(bad, []byte("NOTAWAVEFILE...."), 0o644)
	if _, err := loadWAV(bad); err == nil {
		t.Error("non-RIFF file should error")
	}

	// 8-bit PCM is unsupported.
	eight := filepath.Join(dir, "eight.wav")
	var buf bytes.Buffer
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(36))
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	binary.Write(&buf, binary.LittleEndian, uint32(16))
	binary.Write(&buf, binary.LittleEndian, uint16(1))
	binary.Write(&buf, binary.LittleEndian, uint16(1))
	binary.Write(&buf, binary.LittleEndian, uint32(44100))
	binary.Write(&buf, binary.LittleEndian, uint32(44100))
	binary.Write(&buf, binary.LittleEndian, uint16(1))
	binary.Write(&buf, binary.LittleEndian, uint16(8)) // 8-bit
	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, uint32(0))
	os.WriteFile(eight, buf.Bytes(), 0o644)
	if _, err := loadWAV(eight); err == nil {
		t.Error("8-bit PCM should error")
	}
}

func TestAnalyzeFullPath(t *testing.T) {
	// >= fftSize (32768) samples so analyze runs the spectrum-band branch.
	const n = 40000
	samples := make([]int16, n)
	for i := range samples {
		samples[i] = int16(20000 * math.Sin(2*math.Pi*440*float64(i)/44100))
	}
	path := filepath.Join(t.TempDir(), "tone.wav")
	writeWAV(t, path, 1, 44100, samples)

	// Silence analyze's stdout to keep test output clean. Restore via defer so
	// a panic in analyze() can't leave the process-global os.Stdout dangling.
	old := os.Stdout
	devnull, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	os.Stdout = devnull
	defer func() {
		os.Stdout = old
		if devnull != nil {
			devnull.Close()
		}
	}()
	if err := analyze(path); err != nil {
		t.Fatalf("analyze: %v", err)
	}

	// analyze should propagate loadWAV errors too.
	if err := analyze(filepath.Join(t.TempDir(), "missing.wav")); err == nil {
		t.Error("analyze on missing file should error")
	}
}

func TestFFTImpulseAndTone(t *testing.T) {
	// FFT of a unit impulse is flat magnitude 1 across all bins.
	n := 64
	x := make([]complex128, n)
	x[0] = 1
	fft(x)
	for i, v := range x {
		if math.Abs(cmplx.Abs(v)-1) > 1e-9 {
			t.Fatalf("impulse bin %d magnitude = %f, want 1", i, cmplx.Abs(v))
		}
	}

	// FFT of cos(2*pi*k*i/n) has energy concentrated in bins k and n-k.
	const k = 4
	y := make([]complex128, n)
	for i := range y {
		y[i] = complex(math.Cos(2*math.Pi*k*float64(i)/float64(n)), 0)
	}
	fft(y)
	peak, peakBin := 0.0, 0
	for i := 0; i < n/2; i++ {
		if m := cmplx.Abs(y[i]); m > peak {
			peak, peakBin = m, i
		}
	}
	if peakBin != k {
		t.Errorf("tone peak bin = %d, want %d", peakBin, k)
	}
}
