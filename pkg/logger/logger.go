package logger

import (
	"fmt"
	"io"
	"os"
	"time"
)

// LogLevel represents different logging levels
type LogLevel int

const (
	LogLevelOff LogLevel = iota
	LogLevelError
	LogLevelWarn
	LogLevelInfo
	LogLevelDebug
	LogLevelTrace
)

// Logger handles all logging for the emulator
type Logger struct {
	level         LogLevel
	writer        io.Writer
	cpuEnabled    bool
	ppuEnabled    bool
	apuEnabled    bool
	mapperEnabled bool
}

var globalLogger *Logger

// Initialize sets up the global logger
func Initialize(level LogLevel, filename string) error {
	var writer io.Writer = os.Stdout

	if filename != "" {
		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create log file: %w", err)
		}
		writer = file
	}

	globalLogger = &Logger{
		level:         level,
		writer:        writer,
		cpuEnabled:    true,
		ppuEnabled:    false,
		apuEnabled:    false,
		mapperEnabled: false,
	}

	return nil
}

// SetCPULogging enables or disables CPU instruction logging
func SetCPULogging(enabled bool) {
	if globalLogger != nil {
		globalLogger.cpuEnabled = enabled
	}
}

// SetPPULogging enables or disables PPU logging
func SetPPULogging(enabled bool) {
	if globalLogger != nil {
		globalLogger.ppuEnabled = enabled
	}
}

// SetAPULogging enables or disables APU logging
func SetAPULogging(enabled bool) {
	if globalLogger != nil {
		globalLogger.apuEnabled = enabled
	}
}

// SetMapperLogging enables or disables mapper logging
func SetMapperLogging(enabled bool) {
	if globalLogger != nil {
		globalLogger.mapperEnabled = enabled
	}
}

// LogCPU logs CPU instruction execution (disabled for performance)
func LogCPU(format string, args ...interface{}) {
	if globalLogger != nil && globalLogger.cpuEnabled && globalLogger.level >= LogLevelDebug {
		timestamp := time.Now().Format("15:04:05.000")
		message := fmt.Sprintf(format, args...)
		fmt.Fprintf(globalLogger.writer, "[%s] CPU: %s\n", timestamp, message)
	}
}

// LogPPU logs PPU operations
func LogPPU(format string, args ...interface{}) {
	if globalLogger != nil && globalLogger.ppuEnabled && globalLogger.level >= LogLevelTrace {
		timestamp := time.Now().Format("15:04:05.000")
		message := fmt.Sprintf(format, args...)
		fmt.Fprintf(globalLogger.writer, "[%s] PPU: %s\n", timestamp, message)
	}
}

// LogAPU logs APU operations
func LogAPU(format string, args ...interface{}) {
	if globalLogger != nil && globalLogger.apuEnabled && globalLogger.level >= LogLevelDebug {
		timestamp := time.Now().Format("15:04:05.000")
		message := fmt.Sprintf(format, args...)
		fmt.Fprintf(globalLogger.writer, "[%s] APU: %s\n", timestamp, message)
	}
}

// LogMapper logs mapper operations
func LogMapper(format string, args ...interface{}) {
	if globalLogger != nil && globalLogger.mapperEnabled && globalLogger.level >= LogLevelDebug {
		timestamp := time.Now().Format("15:04:05.000")
		message := fmt.Sprintf(format, args...)
		fmt.Fprintf(globalLogger.writer, "[%s] MAPPER: %s\n", timestamp, message)
	}
}

// LogInfo logs general information
func LogInfo(format string, args ...interface{}) {
	if globalLogger != nil && globalLogger.level >= LogLevelInfo {
		timestamp := time.Now().Format("15:04:05.000")
		message := fmt.Sprintf(format, args...)
		fmt.Fprintf(globalLogger.writer, "[%s] INFO: %s\n", timestamp, message)
	}
}

// LogError logs errors
func LogError(format string, args ...interface{}) {
	if globalLogger != nil && globalLogger.level >= LogLevelError {
		timestamp := time.Now().Format("15:04:05.000")
		message := fmt.Sprintf(format, args...)
		fmt.Fprintf(globalLogger.writer, "[%s] ERROR: %s\n", timestamp, message)
	}
}

// LogDebug logs debug information
func LogDebug(format string, args ...interface{}) {
	if globalLogger != nil && globalLogger.level >= LogLevelDebug {
		timestamp := time.Now().Format("15:04:05.000")
		message := fmt.Sprintf(format, args...)
		fmt.Fprintf(globalLogger.writer, "[%s] DEBUG: %s\n", timestamp, message)
	}
}

// GetLogLevelFromString converts string to LogLevel
func GetLogLevelFromString(level string) LogLevel {
	switch level {
	case "off":
		return LogLevelOff
	case "error":
		return LogLevelError
	case "warn":
		return LogLevelWarn
	case "info":
		return LogLevelInfo
	case "debug":
		return LogLevelDebug
	case "trace":
		return LogLevelTrace
	default:
		return LogLevelInfo
	}
}

// Close closes the logger and any associated files
func Close() {
	if globalLogger != nil {
		if file, ok := globalLogger.writer.(*os.File); ok && file != os.Stdout && file != os.Stderr {
			file.Close()
		}
	}
}
