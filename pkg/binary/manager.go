// Package binary handles binary management operations
package binary

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

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
	log.Debug("fetching latest release", "repo", repo)
	release, err := m.client.GetLatestRelease(repo)
	if err != nil {
		log.Error("failed to get latest release", "repo", repo, "error", err)
		return fmt.Errorf("failed to get latest release: %w", err)
	}

	asset, err := m.findMatchingAsset(release.Assets)
	if err != nil {
		log.Error("failed to find matching asset", "repo", repo, "error", err)
		return fmt.Errorf("failed to find matching asset: %w", err)
	}

	destPath := filepath.Join(m.binDir, asset.Name)
	log.Debug("downloading asset", "path", destPath, "url", asset.BrowserDownloadURL)
	if err := m.client.DownloadAsset(asset.BrowserDownloadURL, destPath); err != nil {
		log.Error("failed to download asset", "path", destPath, "error", err)
		return fmt.Errorf("failed to download asset: %w", err)
	}

	if err := archive.ExtractFile(destPath, destPath); err != nil {
		log.Error("failed to extract file", "path", destPath, "error", err)
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
		log.Debug("failed to store metadata", "error", err)
		// Don't fail the installation if metadata storage fails
	}

	log.Info("installed binary", "name", asset.Name, "path", destPath, "version", release.TagName)
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
		log.Debug("failed to load metadata", "error", err)
		// Continue with deletion even if metadata is missing
	}

	// Delete the binary
	if err := os.Remove(binaryPath); err != nil {
		return fmt.Errorf("failed to delete binary: %w", err)
	}

	// Delete metadata if it exists
	if meta != nil {
		if err := metadata.Delete(binaryPath); err != nil {
			log.Debug("failed to delete metadata", "error", err)
			// Don't fail if metadata deletion fails
		}
	}

	log.Info("deleted binary", "name", binaryName)
	return nil
}

// InstalledBinary represents an installed binary and its metadata
type InstalledBinary struct {
	Name       string
	Path       string
	Repository string
	Version    string
	Host       string
}

// ListInstalled returns a list of all installed binaries and their metadata
func (m *Manager) ListInstalled() ([]InstalledBinary, error) {
	entries, err := os.ReadDir(m.binDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read bin directory: %w", err)
	}

	var binaries []InstalledBinary
	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip directories
		}

		binaryPath := filepath.Join(m.binDir, entry.Name())
		meta, err := metadata.Load(binaryPath)
		if err != nil {
			log.Debug("failed to load metadata", "binary", entry.Name(), "error", err)
			// Include binary with minimal info if metadata is missing
			binaries = append(binaries, InstalledBinary{
				Name: entry.Name(),
				Path: binaryPath,
			})
			continue
		}

		binaries = append(binaries, InstalledBinary{
			Name:       entry.Name(),
			Path:       binaryPath,
			Repository: meta.Repository,
			Version:    meta.Version,
			Host:       meta.GHHost,
		})
	}

	return binaries, nil
}

// GetBinaryPath returns the full path for a binary name
func (m *Manager) GetBinaryPath(binaryName string) string {
	return filepath.Join(m.binDir, binaryName)
}

// GetBinDir returns the binary installation directory
func (m *Manager) GetBinDir() string {
	return m.binDir
}

// Update updates a specific binary from its repository
func (m *Manager) Update(repo string) error {
	log.Info("updating binary", "repo", repo)
	return m.Install(repo)
}

// UpdateAll updates all installed binaries that were installed from GitHub repositories
func (m *Manager) UpdateAll() error {
	log.Info("updating all installed binaries")
	entries, err := os.ReadDir(m.binDir)
	if err != nil {
		return fmt.Errorf("failed to read install directory: %w", err)
	}

	var updateErrors []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		binaryPath := filepath.Join(m.binDir, entry.Name())
		meta, err := metadata.Load(binaryPath)
		if err != nil {
			log.Debug("skipping binary without metadata", "name", entry.Name())
			continue
		}

		if meta.Repository == "" {
			log.Debug("skipping binary without repository info", "name", entry.Name())
			continue
		}

		log.Info("updating binary", "repo", meta.Repository)
		if err := m.Update(meta.Repository); err != nil {
			msg := fmt.Sprintf("failed to update %s: %v", meta.Repository, err)
			log.Error("update error", "repo", meta.Repository, "error", err)
			updateErrors = append(updateErrors, msg)
		}
	}

	if len(updateErrors) > 0 {
		return fmt.Errorf("update completed with errors:\n%s", strings.Join(updateErrors, "\n"))
	}

	return nil
}
