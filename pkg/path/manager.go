// Package path handles managed directory resolution for gh-install-from
package path

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	dirBin       = "bin"
	dirDownloads = "downloads"
	dirMetadata  = "metadata"
)

// Manager provides central path resolution for the managed gh-install-from directory
type Manager struct {
	root string
}

// New creates a new path Manager with the configured root
func New() (*Manager, error) {
	root, err := getRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to get root directory: %w", err)
	}
	return &Manager{root: root}, nil
}

// getRoot returns the root directory: $GH_INSTALL_FROM_HOME or ~/.gh-install-from
func getRoot() (string, error) {
	if v := os.Getenv("GH_INSTALL_FROM_HOME"); v != "" {
		return filepath.Clean(v), nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".gh-install-from"), nil
}

// GetRoot returns the root directory of the managed gh-install-from installation
func (m *Manager) GetRoot() string {
	return m.root
}

// GetBinDir returns the bin directory where symlinks/shim executables live
func (m *Manager) GetBinDir() string {
	return filepath.Join(m.root, dirBin)
}

// GetDownloadsDir returns the downloads directory for actual binaries
func (m *Manager) GetDownloadsDir() string {
	return filepath.Join(m.root, dirDownloads)
}

// GetMetadataDir returns the metadata directory for binary metadata JSON files
func (m *Manager) GetMetadataDir() string {
	return filepath.Join(m.root, dirMetadata)
}
