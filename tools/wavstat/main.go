// wavstat reads a 16-bit PCM WAV file and prints summary statistics:
// duration, RMS, peak, DC offset, and per-octave energy bands. Used to
// compare GoNES audio output against reference emulator recordings.
package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/cmplx"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: wavstat <file.wav> [<file2.wav> ...]")
		os.Exit(1)
	}
	for _, path := range os.Args[1:] {
		if err := analyze(path); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
		}
		fmt.Println()
	}
}

type wavInfo struct {
	channels   uint16
	sampleRate uint32
	bitsPer    uint16
	samples    []float64 // averaged to mono, normalized to -1..1
}

func loadWAV(path string) (*wavInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var riff [12]byte
	if _, err := f.Read(riff[:]); err != nil {
		return nil, err
	}
	if string(riff[0:4]) != "RIFF" || string(riff[8:12]) != "WAVE" {
		return nil, fmt.Errorf("not a RIFF/WAVE file")
	}

	info := &wavInfo{}
	var dataBytes []byte
	for {
		var hdr [8]byte
		if _, err := f.Read(hdr[:]); err != nil {
			break
		}
		chunkID := string(hdr[0:4])
		chunkSize := binary.LittleEndian.Uint32(hdr[4:8])
		buf := make([]byte, chunkSize)
		if _, err := f.Read(buf); err != nil {
			return nil, err
		}
		switch chunkID {
		case "fmt ":
			info.channels = binary.LittleEndian.Uint16(buf[2:4])
			info.sampleRate = binary.LittleEndian.Uint32(buf[4:8])
			info.bitsPer = binary.LittleEndian.Uint16(buf[14:16])
		case "data":
			dataBytes = buf
		}
		if chunkSize%2 == 1 {
			f.Seek(1, 1)
		}
	}

	if info.bitsPer != 16 {
		return nil, fmt.Errorf("only 16-bit PCM supported, got %d", info.bitsPer)
	}
	bytesPerFrame := int(info.channels) * 2
	frameCount := len(dataBytes) / bytesPerFrame
	info.samples = make([]float64, frameCount)
	for i := 0; i < frameCount; i++ {
		var sum int32
		for c := 0; c < int(info.channels); c++ {
			off := i*bytesPerFrame + c*2
			s := int16(binary.LittleEndian.Uint16(dataBytes[off : off+2]))
			sum += int32(s)
		}
		info.samples[i] = float64(sum) / float64(info.channels) / 32768.0
	}
	return info, nil
}

func analyze(path string) error {
	w, err := loadWAV(path)
	if err != nil {
		return err
	}
	dur := float64(len(w.samples)) / float64(w.sampleRate)

	var sumSq, sumAbs, sum, peak float64
	for _, s := range w.samples {
		sumSq += s * s
		sumAbs += math.Abs(s)
		sum += s
		if math.Abs(s) > peak {
			peak = math.Abs(s)
		}
	}
	n := float64(len(w.samples))
	rms := math.Sqrt(sumSq / n)
	mean := sum / n
	meanAbs := sumAbs / n

	fmt.Printf("=== %s ===\n", path)
	fmt.Printf("  channels=%d  sample_rate=%d  bits=%d\n", w.channels, w.sampleRate, w.bitsPer)
	fmt.Printf("  duration=%.2fs  samples=%d\n", dur, len(w.samples))
	fmt.Printf("  peak=%.4f (%.1f dBFS)\n", peak, 20*math.Log10(peak+1e-12))
	fmt.Printf("  rms =%.4f (%.1f dBFS)\n", rms, 20*math.Log10(rms+1e-12))
	fmt.Printf("  mean=%+.4f  meanAbs=%.4f\n", mean, meanAbs)
	fmt.Printf("  crest factor=%.2f  (peak/rms; higher=more dynamic)\n", peak/(rms+1e-12))

	// Per-band RMS via simple FFT on a chunk in the middle. Aligned to a
	// power-of-two window so the math stays clean — the absolute frequencies
	// shift slightly with window size but the *ratios* between bands are
	// what we care about for comparison.
	const fftSize = 32768
	if len(w.samples) >= fftSize {
		mid := len(w.samples)/2 - fftSize/2
		chunk := make([]complex128, fftSize)
		for i := 0; i < fftSize; i++ {
			// Hann window to suppress spectral leakage.
			win := 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(fftSize-1)))
			chunk[i] = complex(w.samples[mid+i]*win, 0)
		}
		fft(chunk)
		mags := make([]float64, fftSize/2)
		for i := range mags {
			mags[i] = cmplx.Abs(chunk[i])
		}

		bands := []struct {
			lo, hi float64
			label  string
		}{
			{20, 80, "sub-bass  20-80"},
			{80, 250, "bass    80-250"},
			{250, 500, "lo-mid 250-500"},
			{500, 2000, "mid    500-2k "},
			{2000, 5000, "hi-mid   2k-5k"},
			{5000, 10000, "presence 5-10k"},
			{10000, 20000, "treble  10-20k"},
		}
		fmt.Println("  spectrum (Hann-windowed FFT of mid 32768 samples):")
		for _, b := range bands {
			lo := int(b.lo * fftSize / float64(w.sampleRate))
			hi := int(b.hi * fftSize / float64(w.sampleRate))
			if hi > len(mags) {
				hi = len(mags)
			}
			var s float64
			for i := lo; i < hi; i++ {
				s += mags[i] * mags[i]
			}
			rms := math.Sqrt(s / float64(hi-lo+1))
			fmt.Printf("    %s Hz: %.2f dB\n", b.label, 20*math.Log10(rms+1e-12))
		}
	}
	return nil
}

// fft is an in-place iterative Cooley-Tukey radix-2. n must be a power of 2.
func fft(x []complex128) {
	n := len(x)
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			x[i], x[j] = x[j], x[i]
		}
	}
	for size := 2; size <= n; size <<= 1 {
		half := size >> 1
		step := -2 * math.Pi / float64(size)
		for i := 0; i < n; i += size {
			for k := 0; k < half; k++ {
				t := cmplx.Rect(1, step*float64(k)) * x[i+k+half]
				x[i+k+half] = x[i+k] - t
				x[i+k] = x[i+k] + t
			}
		}
	}
}
