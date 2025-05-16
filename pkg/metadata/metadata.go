// Package metadata handles storage and retrieval of binary installation metadata
package metadata

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// BinaryMetadata stores information about an installed binary
type BinaryMetadata struct {
	GHHost     string `json:"gh_host"`     // GitHub host (e.g., "github.com")
	Repository string `json:"repository"`  // Repository in owner/repo format
	Version    string `json:"version"`     // Installed version (tag or commit)
	BinaryPath string `json:"binary_path"` // Path to the installed binary
}

// Store saves metadata for an installed binary
func Store(metadata *BinaryMetadata) error {
	if metadata == nil {
		return fmt.Errorf("metadata cannot be nil")
	}

	if metadata.BinaryPath == "" {
		return fmt.Errorf("binary path cannot be empty")
	}

	metadataDir, err := getMetadataDir()
	if err != nil {
		return fmt.Errorf("failed to get metadata directory: %w", err)
	}

	// Create metadata directory if it doesn't exist
	if err := os.MkdirAll(metadataDir, 0750); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	// Use binary name as the metadata file name
	binaryName := filepath.Base(metadata.BinaryPath)
	metadataPath := filepath.Join(metadataDir, binaryName+".json")

	// Marshal metadata to JSON
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Write metadata file
	if err := os.WriteFile(metadataPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}

// Load retrieves metadata for an installed binary
func Load(binaryPath string) (*BinaryMetadata, error) {
	if binaryPath == "" {
		return nil, fmt.Errorf("binary path cannot be empty")
	}

	metadataDir, err := getMetadataDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata directory: %w", err)
	}

	// Get metadata file path
	binaryName := filepath.Base(binaryPath)
	metadataPath := filepath.Join(metadataDir, binaryName+".json")

	// Read metadata file
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no metadata found for binary %s", binaryName)
		}
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	// Unmarshal metadata
	var metadata BinaryMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &metadata, nil
}

// Delete removes metadata for an installed binary
func Delete(binaryPath string) error {
	if binaryPath == "" {
		return fmt.Errorf("binary path cannot be empty")
	}

	metadataDir, err := getMetadataDir()
	if err != nil {
		return fmt.Errorf("failed to get metadata directory: %w", err)
	}

	// Get metadata file path
	binaryName := filepath.Base(binaryPath)
	metadataPath := filepath.Join(metadataDir, binaryName+".json")

	// Remove metadata file
	if err := os.Remove(metadataPath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete metadata file: %w", err)
	}

	return nil
}

// getMetadataDir returns the directory where metadata files are stored
func getMetadataDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(homeDir, ".local", "share", "gh-install-from", "metadata"), nil
}
