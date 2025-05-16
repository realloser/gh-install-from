package log

import (
	"log/slog"
	"os"
)

var (
	// Logger is the global logger instance
	Logger *slog.Logger

	// verbose controls whether debug logs are enabled
	verbose bool
)

// Init initializes the logger with the given verbosity level
func Init(isVerbose bool) {
	verbose = isVerbose

	// Create handler with appropriate level
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level: level,
		// Add source file and line to log output in verbose mode
		AddSource: verbose,
	}

	// Create a text handler for human-readable output
	handler := slog.NewTextHandler(os.Stderr, opts)
	Logger = slog.New(handler)

	// Set as default logger
	slog.SetDefault(Logger)

	if verbose {
		Logger.Debug("verbose logging enabled")
	}
}

// IsVerbose returns whether verbose logging is enabled
func IsVerbose() bool {
	return verbose
}

// Debug logs at debug level if verbose mode is enabled
func Debug(msg string, args ...any) {
	if verbose {
		Logger.Debug(msg, args...)
	}
}

// Info logs at info level
func Info(msg string, args ...any) {
	Logger.Info(msg, args...)
}

// Warn logs at warn level
func Warn(msg string, args ...any) {
	Logger.Warn(msg, args...)
}

// Error logs at error level
func Error(msg string, args ...any) {
	Logger.Error(msg, args...)
}
