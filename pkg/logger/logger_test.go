package logger

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetLogLevelFromString(t *testing.T) {
	cases := map[string]LogLevel{
		"off":     LogLevelOff,
		"error":   LogLevelError,
		"warn":    LogLevelWarn,
		"info":    LogLevelInfo,
		"debug":   LogLevelDebug,
		"trace":   LogLevelTrace,
		"unknown": LogLevelInfo, // default
		"":        LogLevelInfo,
	}
	for in, want := range cases {
		if got := GetLogLevelFromString(in); got != want {
			t.Errorf("GetLogLevelFromString(%q) = %d, want %d", in, got, want)
		}
	}
}

// withBuffer initializes the global logger at the given level and redirects
// its output to a buffer the caller can inspect. Returns the buffer.
func withBuffer(t *testing.T, level LogLevel) *bytes.Buffer {
	t.Helper()
	if err := Initialize(level, ""); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	buf := &bytes.Buffer{}
	globalLogger.writer = buf
	return buf
}

func TestPerSubsystemGates(t *testing.T) {
	withBuffer(t, LogLevelTrace)

	// Initialize defaults: CPU on, PPU/APU/mapper off.
	if !CPUEnabled() {
		t.Error("CPUEnabled should be true after Initialize at Trace")
	}
	if PPUEnabled() {
		t.Error("PPUEnabled should be false until SetPPULogging(true)")
	}
	SetPPULogging(true)
	if !PPUEnabled() {
		t.Error("PPUEnabled should be true after SetPPULogging(true)")
	}
	SetCPULogging(false)
	if CPUEnabled() {
		t.Error("CPUEnabled should be false after SetCPULogging(false)")
	}

	// Level below the per-subsystem threshold disables them regardless of flag.
	withBuffer(t, LogLevelInfo)
	SetCPULogging(true)
	SetPPULogging(true)
	if CPUEnabled() {
		t.Error("CPUEnabled requires level >= Debug")
	}
	if PPUEnabled() {
		t.Error("PPUEnabled requires level >= Trace")
	}
}

func TestLogEmitsWhenEnabled(t *testing.T) {
	buf := withBuffer(t, LogLevelTrace)
	SetCPULogging(true)
	SetPPULogging(true)
	SetAPULogging(true)
	SetMapperLogging(true)

	LogCPU("cpu %d", 1)
	LogPPU("ppu %d", 2)
	LogAPU("apu %d", 3)
	LogMapper("mapper %d", 4)
	LogInfo("info %d", 5)
	LogError("error %d", 6)
	LogDebug("debug %d", 7)

	out := buf.String()
	for _, want := range []string{
		"CPU: cpu 1", "PPU: ppu 2", "APU: apu 3", "MAPPER: mapper 4",
		"INFO: info 5", "ERROR: error 6", "DEBUG: debug 7",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n--- got ---\n%s", want, out)
		}
	}
}

func TestLogSuppressedBelowLevel(t *testing.T) {
	buf := withBuffer(t, LogLevelError)
	// Subsystem flags on, but level too low: these must not emit.
	SetCPULogging(true)
	SetPPULogging(true)
	SetAPULogging(true)
	SetMapperLogging(true)
	LogCPU("nope")
	LogPPU("nope")
	LogAPU("nope")
	LogMapper("nope")
	LogInfo("nope")
	LogDebug("nope")
	// Error is at the active level and should emit.
	LogError("yes")

	out := buf.String()
	if strings.Contains(out, "nope") {
		t.Errorf("sub-threshold logs leaked: %q", out)
	}
	if !strings.Contains(out, "ERROR: yes") {
		t.Errorf("error log missing: %q", out)
	}
}

func TestNilLoggerIsSafe(t *testing.T) {
	// Restore the global so a later test (or -shuffle ordering) doesn't inherit
	// a nil logger.
	prev := globalLogger
	t.Cleanup(func() { globalLogger = prev })

	// Before Initialize, the gates report false and Log* are no-ops (must not panic).
	globalLogger = nil
	if CPUEnabled() || PPUEnabled() {
		t.Error("gates should be false with nil logger")
	}
	LogCPU("x")
	LogInfo("x") // must not panic
}

func TestInitializeFileAndClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log.txt")
	if err := Initialize(LogLevelInfo, path); err != nil {
		t.Fatalf("Initialize(file): %v", err)
	}
	LogInfo("to file %d", 42)
	Close()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if !strings.Contains(string(data), "INFO: to file 42") {
		t.Errorf("file missing log line, got %q", string(data))
	}
}

func TestInitializeBadPath(t *testing.T) {
	// A path inside a non-existent directory must fail cleanly.
	bad := filepath.Join(t.TempDir(), "nope", "deeper", "log.txt")
	if err := Initialize(LogLevelInfo, bad); err == nil {
		t.Error("Initialize with unwritable path should error")
	}
}
