package test

// frameHistogram returns the per-color pixel counts of fb plus the
// dominant color and its count. Shared between the scanline test
// (which needs the dominant color as a "background" baseline) and
// mapper smoke tests (which use the distribution shape as a
// "is the game actually rendering?" heuristic).
func frameHistogram(fb []uint32) (hist map[uint32]int, dominant uint32, dominantCount int) {
	hist = make(map[uint32]int, 16)
	for _, p := range fb {
		hist[p]++
	}
	for v, n := range hist {
		if n > dominantCount {
			dominant = v
			dominantCount = n
		}
	}
	return
}
