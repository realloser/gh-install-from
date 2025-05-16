package log

import (
	"bytes"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestInit(t *testing.T) {
	tests := []struct {
		name     string
		verbose  bool
		wantLogs bool
	}{
		{
			name:     "verbose mode enabled",
			verbose:  true,
			wantLogs: true,
		},
		{
			name:     "verbose mode disabled",
			verbose:  false,
			wantLogs: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture log output
			var buf bytes.Buffer
			origStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			// Initialize logger
			Init(tt.verbose)

			// Write debug message
			Debug("test debug message")

			// Restore stderr
			w.Close()
			os.Stderr = origStderr

			// Read captured output
			buf.ReadFrom(r)
			output := buf.String()

			// Check if debug message was logged
			hasDebugLog := strings.Contains(output, "test debug message")
			if hasDebugLog != tt.wantLogs {
				t.Errorf("Debug logging = %v, want %v", hasDebugLog, tt.wantLogs)
			}

			// Verify verbose state
			if IsVerbose() != tt.verbose {
				t.Errorf("IsVerbose() = %v, want %v", IsVerbose(), tt.verbose)
			}
		})
	}
}

func TestLogLevels(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	Logger = slog.New(handler)

	tests := []struct {
		name     string
		logFunc  func(string, ...any)
		message  string
		wantText string
	}{
		{
			name:     "info level",
			logFunc:  Info,
			message:  "info message",
			wantText: "INFO info message",
		},
		{
			name:     "warn level",
			logFunc:  Warn,
			message:  "warn message",
			wantText: "WARN warn message",
		},
		{
			name:     "error level",
			logFunc:  Error,
			message:  "error message",
			wantText: "ERROR error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc(tt.message)
			if !strings.Contains(buf.String(), tt.wantText) {
				t.Errorf("log output = %q, want to contain %q", buf.String(), tt.wantText)
			}
		})
	}
} 