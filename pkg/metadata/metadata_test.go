package metadata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMetadata(t *testing.T) {
	// Create a temporary home directory for testing
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Test metadata
	testMeta := &BinaryMetadata{
		GHHost:     "github.com",
		Repository: "test/repo",
		Version:    "v1.0.0",
		BinaryPath: "/usr/local/bin/test-binary",
	}

	// Test Store
	t.Run("Store", func(t *testing.T) {
		// Test storing nil metadata
		if err := Store(nil); err == nil {
			t.Error("expected error when storing nil metadata")
		}

		// Test storing metadata with empty binary path
		if err := Store(&BinaryMetadata{}); err == nil {
			t.Error("expected error when storing metadata with empty binary path")
		}

		// Test storing valid metadata
		if err := Store(testMeta); err != nil {
			t.Fatalf("failed to store metadata: %v", err)
		}

		// Verify metadata file exists and contains correct data
		metadataDir, err := GetMetadataDir()
		if err != nil {
			t.Fatalf("failed to get metadata directory: %v", err)
		}

		metadataPath := filepath.Join(metadataDir, "test-binary.json")
		data, err := os.ReadFile(metadataPath)
		if err != nil {
			t.Fatalf("failed to read metadata file: %v", err)
		}

		var storedMeta BinaryMetadata
		if err := json.Unmarshal(data, &storedMeta); err != nil {
			t.Fatalf("failed to unmarshal stored metadata: %v", err)
		}

		if storedMeta != *testMeta {
			t.Errorf("stored metadata does not match: got %+v, want %+v", storedMeta, testMeta)
		}
	})

	// Test Load
	t.Run("Load", func(t *testing.T) {
		// Test loading with empty binary path
		if _, err := Load(""); err == nil {
			t.Error("expected error when loading with empty binary path")
		}

		// Test loading non-existent metadata
		if _, err := Load("/nonexistent"); err == nil {
			t.Error("expected error when loading non-existent metadata")
		}

		// Test loading existing metadata
		loadedMeta, err := Load(testMeta.BinaryPath)
		if err != nil {
			t.Fatalf("failed to load metadata: %v", err)
		}

		if *loadedMeta != *testMeta {
			t.Errorf("loaded metadata does not match: got %+v, want %+v", loadedMeta, testMeta)
		}
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		// Test deleting with empty binary path
		if err := Delete(""); err == nil {
			t.Error("expected error when deleting with empty binary path")
		}

		// Test deleting non-existent metadata (should not error)
		if err := Delete("/nonexistent"); err != nil {
			t.Errorf("unexpected error when deleting non-existent metadata: %v", err)
		}

		// Test deleting existing metadata
		if err := Delete(testMeta.BinaryPath); err != nil {
			t.Fatalf("failed to delete metadata: %v", err)
		}

		// Verify metadata file is deleted
		metadataDir, err := GetMetadataDir()
		if err != nil {
			t.Fatalf("failed to get metadata directory: %v", err)
		}

		metadataPath := filepath.Join(metadataDir, "test-binary.json")
		if _, err := os.Stat(metadataPath); !os.IsNotExist(err) {
			t.Error("metadata file still exists after deletion")
		}

		// Verify loading deleted metadata returns error
		if _, err := Load(testMeta.BinaryPath); err == nil {
			t.Error("expected error when loading deleted metadata")
		}
	})
}
