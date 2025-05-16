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

func init() {
	// Initialize with default settings
	Init(false)
}

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
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Remove time from output
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			// Remove source file info
			if a.Key == slog.SourceKey {
				return slog.Attr{}
			}
			// Clean up level prefix
			if a.Key == slog.LevelKey {
				return slog.Attr{
					Key:   "level",
					Value: a.Value,
				}
			}
			return a
		},
	}

	// Create a text handler for human-readable output
	handler := slog.NewTextHandler(os.Stderr, opts)
	Logger = slog.New(handler)

	// Set as default logger
	slog.SetDefault(Logger)
}

// IsVerbose returns whether verbose logging is enabled
func IsVerbose() bool {
	return verbose
}

// Debug logs at debug level if verbose mode is enabled
func Debug(msg string, args ...any) {
	if verbose {
		// Convert args to proper key-value pairs
		kvs := make([]any, 0, len(args))
		for i := 0; i < len(args); i++ {
			if i+1 < len(args) {
				kvs = append(kvs, args[i], args[i+1])
				i++
			} else {
				kvs = append(kvs, "value", args[i])
			}
		}
		Logger.Debug(msg, kvs...)
	}
}

// Info logs at info level
func Info(msg string, args ...any) {
	// Convert args to proper key-value pairs
	kvs := make([]any, 0, len(args))
	for i := 0; i < len(args); i++ {
		if i+1 < len(args) {
			kvs = append(kvs, args[i], args[i+1])
			i++
		} else {
			kvs = append(kvs, "value", args[i])
		}
	}
	Logger.Info(msg, kvs...)
}

// Warn logs at warn level
func Warn(msg string, args ...any) {
	// Convert args to proper key-value pairs
	kvs := make([]any, 0, len(args))
	for i := 0; i < len(args); i++ {
		if i+1 < len(args) {
			kvs = append(kvs, args[i], args[i+1])
			i++
		} else {
			kvs = append(kvs, "value", args[i])
		}
	}
	Logger.Warn(msg, kvs...)
}

// Error logs at error level
func Error(msg string, args ...any) {
	// Convert args to proper key-value pairs
	kvs := make([]any, 0, len(args))
	for i := 0; i < len(args); i++ {
		if i+1 < len(args) {
			kvs = append(kvs, args[i], args[i+1])
			i++
		} else {
			kvs = append(kvs, "value", args[i])
		}
	}
	Logger.Error(msg, kvs...)
}
