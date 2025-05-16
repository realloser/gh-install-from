// Package installer handles binary installation from GitHub releases
package installer

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

// Installer handles binary installation from GitHub releases
type Installer struct {
	client github.Client
}

// New creates a new Installer
func New(client github.Client) *Installer {
	return &Installer{client: client}
}

// Install installs a binary from a GitHub repository
func (i *Installer) Install(repo string) error {
	log.Debug("fetching latest release for", repo)
	release, err := i.client.GetLatestRelease(repo)
	if err != nil {
		return fmt.Errorf("failed to get latest release: %w", err)
	}

	asset, err := i.findMatchingAsset(release.Assets)
	if err != nil {
		return fmt.Errorf("failed to find matching asset: %w", err)
	}

	binDir := filepath.Join(os.Getenv("HOME"), ".local", "bin")
	if err := os.MkdirAll(binDir, 0750); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	destPath := filepath.Join(binDir, asset.Name)
	log.Debug("downloading asset to", destPath)
	if err := i.client.DownloadAsset(asset.BrowserDownloadURL, destPath); err != nil {
		return fmt.Errorf("failed to download asset: %w", err)
	}

	if err := archive.ExtractFile(destPath, destPath); err != nil {
		return fmt.Errorf("failed to extract file: %w", err)
	}

	// Store metadata for the installed binary
	meta := &metadata.BinaryMetadata{
		GHHost:     i.client.GetHost(),
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
func (i *Installer) findMatchingAsset(assets []github.Asset) (*github.Asset, error) {
	osArch := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
	for _, asset := range assets {
		if archive.IsBinaryForPlatform(asset.Name, osArch) {
			return &asset, nil
		}
	}
	return nil, fmt.Errorf("no matching asset found for %s", osArch)
}
