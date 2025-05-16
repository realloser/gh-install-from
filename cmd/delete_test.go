package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/realloser/gh-install-from/pkg/log"
	"github.com/realloser/gh-install-from/pkg/metadata"
)

func TestRunDelete(t *testing.T) {
	// Initialize logger for tests
	log.Init(false)

	// Create a temporary home directory
	tmpHome := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", oldHome)

	// Create the bin directory
	binDir := filepath.Join(tmpHome, ".local", "bin")
	if err := os.MkdirAll(binDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Test cases
	tests := []struct {
		name       string
		binaryName string
		setup      func(t *testing.T, binDir string) // Setup function to create test files
		wantErr    bool
	}{
		{
			name:       "delete existing binary with metadata",
			binaryName: "test-binary",
			setup: func(t *testing.T, binDir string) {
				// Create test binary
				binaryPath := filepath.Join(binDir, "test-binary")
				if err := os.WriteFile(binaryPath, []byte("test binary"), 0755); err != nil {
					t.Fatal(err)
				}

				// Create metadata
				meta := &metadata.BinaryMetadata{
					GHHost:     "github.com",
					Repository: "test/repo",
					Version:    "v1.0.0",
					BinaryPath: binaryPath,
				}
				if err := metadata.Store(meta); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: false,
		},
		{
			name:       "delete existing binary without metadata",
			binaryName: "test-binary-no-meta",
			setup: func(t *testing.T, binDir string) {
				// Create test binary only
				binaryPath := filepath.Join(binDir, "test-binary-no-meta")
				if err := os.WriteFile(binaryPath, []byte("test binary"), 0755); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: false,
		},
		{
			name:       "delete non-existent binary",
			binaryName: "nonexistent",
			setup:      func(t *testing.T, binDir string) {}, // No setup needed
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run setup
			tt.setup(t, binDir)

			// Run delete command
			err := runDelete(nil, []string{tt.binaryName})
			if (err != nil) != tt.wantErr {
				t.Errorf("runDelete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify binary was deleted
				binaryPath := filepath.Join(binDir, tt.binaryName)
				if _, err := os.Stat(binaryPath); !os.IsNotExist(err) {
					t.Error("binary file still exists after deletion")
				}

				// Verify metadata was deleted
				if _, err := metadata.Load(binaryPath); err == nil {
					t.Error("metadata still exists after deletion")
				}
			}
		})
	}
}
