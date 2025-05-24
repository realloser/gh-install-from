package ui

import (
	"testing"
)

func TestFormatBinaryInfo(t *testing.T) {
	tests := []struct {
		name     string
		binName  string
		path     string
		version  string
		expected string
	}{
		{
			name:     "all fields",
			binName:  "test-binary",
			path:     "/usr/local/bin",
			version:  "v1.0.0",
			expected: "test-binary in /usr/local/bin (v1.0.0) ",
		},
		{
			name:     "name only",
			binName:  "test-binary",
			path:     "",
			version:  "",
			expected: "test-binary ",
		},
		{
			name:     "name and path",
			binName:  "test-binary",
			path:     "/usr/local/bin",
			version:  "",
			expected: "test-binary in /usr/local/bin ",
		},
		{
			name:     "name and version",
			binName:  "test-binary",
			path:     "",
			version:  "v1.0.0",
			expected: "test-binary (v1.0.0) ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatBinaryInfo(tt.binName, tt.path, tt.version)
			if got != tt.expected {
				t.Errorf("FormatBinaryInfo() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFormatActionMessage(t *testing.T) {
	tests := []struct {
		name     string
		action   string
		details  string
		expected string
	}{
		{
			name:     "basic action message",
			action:   "Installed",
			details:  "test-binary v1.0.0",
			expected: "Installed test-binary v1.0.0",
		},
		{
			name:     "empty details",
			action:   "Installed",
			details:  "",
			expected: "Installed ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatActionMessage(tt.action, tt.details)
			if got != tt.expected {
				t.Errorf("FormatActionMessage() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFormatErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "basic error message",
			message:  "Failed to install binary",
			expected: "Failed to install binary",
		},
		{
			name:     "empty message",
			message:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatErrorMessage(tt.message)
			if got != tt.expected {
				t.Errorf("FormatErrorMessage() = %q, want %q", got, tt.expected)
			}
		})
	}
}
