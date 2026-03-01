// Package metadata handles storage and retrieval of binary installation metadata
package metadata

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/realloser/gh-install-from/pkg/config"
)

// BinaryMetadata stores information about an installed binary
type BinaryMetadata struct {
	GHHost         string `json:"gh_host"`         // GitHub host (e.g., "github.com")
	Repository     string `json:"repository"`      // Repository in owner/repo format
	Version        string `json:"version"`         // Installed version (tag or commit)
	BinaryPath     string `json:"binary_path"`     // Path to the installed binary
	OriginalBinary string `json:"original_binary"` // Original name of the binary in the archive
}

// Store saves metadata for an installed binary (convenience wrapper using default adapter)
func Store(metadata *BinaryMetadata) error {
	s, err := NewStore(config.FromEnv())
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}
	return s.Store(metadata)
}

// Load retrieves metadata for an installed binary (convenience wrapper using default Store adapter)
func Load(binaryPath string) (*BinaryMetadata, error) {
	if binaryPath == "" {
		return nil, fmt.Errorf("binary path cannot be empty")
	}
	s, err := NewStore(config.FromEnv())
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}
	binaryName := filepath.Base(binaryPath)
	return s.Load(binaryName)
}

// Delete removes metadata for an installed binary (convenience wrapper using default Store adapter)
func Delete(binaryPath string) error {
	if binaryPath == "" {
		return fmt.Errorf("binary path cannot be empty")
	}
	s, err := NewStore(config.FromEnv())
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}
	binaryName := filepath.Base(binaryPath)
	return s.Delete(binaryName)
}

// GetMetadataDir returns the directory where metadata files are stored (default location)
func GetMetadataDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	root := os.Getenv("GH_INSTALL_FROM_HOME")
	if root == "" {
		root = filepath.Join(homeDir, ".gh-install-from")
	}
	return filepath.Join(root, "metadata"), nil
}
