// Package gui — frame pacing, FPS counter, and window-title updates.
//
// The pacing strategy is target-time accumulation: each frame's deadline is
// startTime + frameCount*FrameTime, not the previous deadline + FrameTime.
// That way Sleep() overshoot doesn't drift the framerate downward.
package gui

import (
	"fmt"
	"time"

	"github.com/yoshiomiyamaegones/pkg/logger"
)

// Timing constants
const (
	TargetFPS = 60.0988 // NES actual framerate

	// TurboRenderInterval throttles texture uploads while turbo is on: the
	// emulator may produce hundreds of fps, but the user only sees ~60Hz, so
	// rendering more often just burns GPU/copy cycles.
	TurboRenderInterval = 16 * time.Millisecond
)

// NTSC NES frame rate: 60.0988 FPS (more precisely: 1789773 / 29780.5 = 60.0988139...)
// Frame time = 1,000,000,000 / 60.0988139 = 16,639,266.85 ns
var FrameTime = time.Duration(16639267) * time.Nanosecond // 16.639267ms per frame

// waitForNextFrame sleeps until the next frame deadline and logs noticeable
// timing deviations every 60 frames. startTime/frameCount form the
// accumulating-target baseline; frameStart is when this frame's work began
// (used only for the deviation log). In turbo mode the pacing is skipped
// entirely — frames run as fast as they can.
func (g *NESGUI) waitForNextFrame(startTime, frameStart time.Time, frameCount int) {
	targetEndTime := startTime.Add(time.Duration(frameCount) * FrameTime)

	if !g.turbo {
		now := time.Now()
		if now.Before(targetEndTime) {
			time.Sleep(targetEndTime.Sub(now))
		}
	}

	// Debug: Log frame timing every 60 frames (skipped in turbo — frames
	// run as fast as they can by design).
	if frameCount%60 == 0 && !g.turbo {
		actualFrameTime := time.Since(frameStart)
		expectedFrameTime := FrameTime
		deviation := float64(actualFrameTime-expectedFrameTime) / float64(expectedFrameTime) * 100

		// Also check average frame rate
		avgFrameTime := time.Since(startTime) / time.Duration(frameCount)
		avgDeviation := float64(avgFrameTime-expectedFrameTime) / float64(expectedFrameTime) * 100

		if deviation > 5 || deviation < -5 || avgDeviation > 2 || avgDeviation < -2 {
			logger.LogInfo("Frame timing: actual=%.3fms, avg=%.3fms, expected=%.3fms, deviation=%.1f%%, avg_dev=%.1f%%",
				actualFrameTime.Seconds()*1000, avgFrameTime.Seconds()*1000,
				expectedFrameTime.Seconds()*1000, deviation, avgDeviation)
		}
	}
}

// updateFPS calculates the current FPS
func (g *NESGUI) updateFPS() {
	g.fpsCounter++

	// Update FPS every 0.5 seconds for more responsive display
	elapsed := time.Since(g.fpsTimer)
	if elapsed >= 500*time.Millisecond {
		g.currentFPS = float64(g.fpsCounter) / elapsed.Seconds()

		// Debug: Log if FPS is significantly off target (skipped in turbo —
		// the FPS is *expected* to be off target while fast-forwarding).
		if g.fpsCounter%30 == 0 && !g.turbo {
			deviation := (g.currentFPS - TargetFPS) / TargetFPS * 100
			if deviation > 5 || deviation < -5 {
				logger.LogInfo("FPS: %.2f (target: %.2f, deviation: %.1f%%)",
					g.currentFPS, TargetFPS, deviation)
			}
		}

		g.fpsCounter = 0
		g.fpsTimer = time.Now()
	}
}

// updateWindowTitle updates the window title with FPS information
func (g *NESGUI) updateWindowTitle() {
	title := fmt.Sprintf("%s - FPS: %.1f", WindowTitle, g.currentFPS)
	if g.turbo {
		title += " [TURBO]"
	}
	g.window.SetTitle(title)
}
