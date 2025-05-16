package version

import "testing"

func TestInfo(t *testing.T) {
	// Store original values
	origVersion := Version
	origCommit := Commit
	origDate := Date
	
	// Restore original values after test
	defer func() {
		Version = origVersion
		Commit = origCommit
		Date = origDate
	}()

	// Test cases
	tests := []struct {
		name     string
		version  string
		want     string
	}{
		{
			name:     "default values",
			version:  "dev",
			want:     "dev",
		},
		{
			name:     "release version",
			version:  "1.0.0",
			want:     "1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Version = tt.version
			if got := Info(); got != tt.want {
				t.Errorf("Info() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetailedInfo(t *testing.T) {
	// Store original values
	origVersion := Version
	origCommit := Commit
	origDate := Date
	
	// Restore original values after test
	defer func() {
		Version = origVersion
		Commit = origCommit
		Date = origDate
	}()

	// Test cases
	tests := []struct {
		name     string
		version  string
		commit   string
		date     string
		want     string
	}{
		{
			name:     "default values",
			version:  "dev",
			commit:   "none",
			date:     "unknown",
			want:     "Version: dev\nCommit: none\nBuild Date: unknown",
		},
		{
			name:     "release values",
			version:  "1.0.0",
			commit:   "abc123",
			date:     "2024-01-01",
			want:     "Version: 1.0.0\nCommit: abc123\nBuild Date: 2024-01-01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Version = tt.version
			Commit = tt.commit
			Date = tt.date
			if got := DetailedInfo(); got != tt.want {
				t.Errorf("DetailedInfo() = %v, want %v", got, tt.want)
			}
		})
	}
} 