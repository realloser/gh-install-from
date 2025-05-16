// Package binary handles binary management operations
package binary

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/realloser/gh-install-from/pkg/archive"
	"github.com/realloser/gh-install-from/pkg/github"
	"github.com/realloser/gh-install-from/pkg/log"
	"github.com/realloser/gh-install-from/pkg/metadata"
)

// Manager handles binary management operations
type Manager struct {
	binDir string
	client github.Client
}

// New creates a new binary Manager
func New(client github.Client) (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	binDir := filepath.Join(homeDir, ".local", "bin")
	if err := os.MkdirAll(binDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create bin directory: %w", err)
	}

	return &Manager{
		binDir: binDir,
		client: client,
	}, nil
}

// Install installs a binary from a GitHub repository
func (m *Manager) Install(repo string) error {
	log.Debug("fetching latest release for", repo)
	release, err := m.client.GetLatestRelease(repo)
	if err != nil {
		return fmt.Errorf("failed to get latest release: %w", err)
	}

	asset, err := m.findMatchingAsset(release.Assets)
	if err != nil {
		return fmt.Errorf("failed to find matching asset: %w", err)
	}

	destPath := filepath.Join(m.binDir, asset.Name)
	log.Debug("downloading asset to", destPath)
	if err := m.client.DownloadAsset(asset.BrowserDownloadURL, destPath); err != nil {
		return fmt.Errorf("failed to download asset: %w", err)
	}

	if err := archive.ExtractFile(destPath, destPath); err != nil {
		return fmt.Errorf("failed to extract file: %w", err)
	}

	// Store metadata for the installed binary
	meta := &metadata.BinaryMetadata{
		GHHost:     m.client.GetHost(),
		Repository: repo,
		Version:    release.TagName,
		BinaryPath: destPath,
	}

	if err := metadata.Store(meta); err != nil {
		log.Debug("failed to store metadata:", err)
		// Don't fail the installation if metadata storage fails
	}

	log.Info("installed", asset.Name, "to", destPath)
	return nil
}

// findMatchingAsset finds the asset that matches the current OS/architecture
func (m *Manager) findMatchingAsset(assets []github.Asset) (*github.Asset, error) {
	osArch := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
	for _, asset := range assets {
		if archive.IsBinaryForPlatform(asset.Name, osArch) {
			return &asset, nil
		}
	}
	return nil, fmt.Errorf("no matching asset found for %s", osArch)
}

// Delete removes an installed binary and its metadata
func (m *Manager) Delete(binaryName string) error {
	binaryPath := filepath.Join(m.binDir, binaryName)

	// Check if the binary exists
	if _, err := os.Stat(binaryPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("binary %s is not installed", binaryName)
		}
		return fmt.Errorf("failed to check binary: %w", err)
	}

	// Load metadata before deleting the binary
	meta, err := metadata.Load(binaryPath)
	if err != nil {
		log.Debug("failed to load metadata:", err)
		// Continue with deletion even if metadata is missing
	}

	// Delete the binary
	if err := os.Remove(binaryPath); err != nil {
		return fmt.Errorf("failed to delete binary: %w", err)
	}

	// Delete metadata if it exists
	if meta != nil {
		if err := metadata.Delete(binaryPath); err != nil {
			log.Debug("failed to delete metadata:", err)
			// Don't fail if metadata deletion fails
		}
	}

	log.Info("deleted", binaryName)
	return nil
}

// GetBinaryPath returns the full path for a binary name
func (m *Manager) GetBinaryPath(binaryName string) string {
	return filepath.Join(m.binDir, binaryName)
}

// GetBinDir returns the binary installation directory
func (m *Manager) GetBinDir() string {
	return m.binDir
}
