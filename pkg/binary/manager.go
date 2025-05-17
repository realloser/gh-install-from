// Package binary handles binary management operations
package binary

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/realloser/gh-install-from/pkg/archive"
	"github.com/realloser/gh-install-from/pkg/github"
	"github.com/realloser/gh-install-from/pkg/log"
	"github.com/realloser/gh-install-from/pkg/metadata"
	"github.com/realloser/gh-install-from/pkg/ui"
)

// Manager handles binary management operations
type Manager struct {
	binDir  string
	client  github.Client
	archive archiver
}

// archiver is an interface for archive operations
type archiver interface {
	ExtractFile(src, destDir string) (string, error)
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
		binDir:  binDir,
		client:  client,
		archive: &archive.Archiver{},
	}, nil
}

// NewWithArchiver creates a new binary Manager with a custom archiver
func NewWithArchiver(client github.Client, archive archiver) (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	binDir := filepath.Join(homeDir, ".local", "bin")
	if err := os.MkdirAll(binDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create bin directory: %w", err)
	}

	return &Manager{
		binDir:  binDir,
		client:  client,
		archive: archive,
	}, nil
}

// Install installs a binary from a GitHub repository
func (m *Manager) Install(repo string) error {
	log.Debug("fetching latest release", "repo", repo)
	release, err := m.client.GetLatestRelease(repo)
	if err != nil {
		return fmt.Errorf("failed to get latest release from %s: %w", repo, err)
	}

	if len(release.Assets) == 0 {
		return fmt.Errorf("repository %s has no assets in its latest release", repo)
	}

	asset, err := m.findMatchingAsset(release.Assets)
	if err != nil {
		return fmt.Errorf("failed to find matching asset for %s: %w", repo, err)
	}

	// Create a temporary directory for extraction
	tmpDir, err := os.MkdirTemp("", "gh-install-from-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Download to temp directory first
	tmpPath := filepath.Join(tmpDir, asset.Name)
	log.Debug("downloading asset", "path", tmpPath, "url", asset.BrowserDownloadURL)
	if err := m.client.DownloadAsset(asset.BrowserDownloadURL, tmpPath); err != nil {
		return fmt.Errorf("failed to download asset: %w", err)
	}

	// Extract the archive to the temp directory
	extractedPath, err := m.archive.ExtractFile(tmpPath, tmpDir)
	if err != nil {
		return fmt.Errorf("failed to extract file: %w", err)
	}

	// Get the binary name for the final destination
	binaryName := getBinaryName(repo)
	destPath := filepath.Join(m.binDir, binaryName)

	// Copy the binary to the final location
	if err := copyFile(extractedPath, destPath); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}

	// Store metadata for the installed binary
	meta := &metadata.BinaryMetadata{
		GHHost:         m.client.GetHost(),
		Repository:     repo,
		Version:        release.TagName,
		BinaryPath:     destPath,
		OriginalBinary: filepath.Base(extractedPath),
	}

	if err := metadata.Store(meta); err != nil {
		log.Debug("failed to store metadata", "error", err)
		// Don't fail the installation if metadata storage fails
	}

	fmt.Println(ui.FormatActionMessage("Installed", ui.FormatBinaryInfo(binaryName, destPath, release.TagName)))
	return nil
}

// getBinaryName returns the expected binary name for a repository
func getBinaryName(repo string) string {
	// Map of known repositories to their binary names
	knownBinaries := map[string]string{
		"BurntSushi/ripgrep": "rg",
		// Add more mappings as needed
	}

	if name, ok := knownBinaries[repo]; ok {
		return name
	}

	// Default to the repository name without owner
	parts := strings.Split(repo, "/")
	if len(parts) > 1 {
		return parts[1]
	}
	return repo
}

// isExecutable checks if a file mode indicates an executable
func isExecutable(mode os.FileMode) bool {
	return mode&0111 != 0
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	return nil
}

// findMatchingAsset finds the asset that matches the current OS/architecture
func (m *Manager) findMatchingAsset(assets []github.Asset) (*github.Asset, error) {
	osArch := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)

	// Map of architecture variations
	archMap := map[string][]string{
		"amd64": {"x86_64", "amd64", "x64"},
		"386":   {"i386", "x86", "386"},
		"arm64": {"arm64", "aarch64"},
	}

	// Map of OS variations
	osMap := map[string][]string{
		"darwin":  {"darwin", "macos", "osx", "mac"},
		"linux":   {"linux"},
		"windows": {"windows", "win"},
	}

	// Get possible variations for current OS/arch
	possibleArch := archMap[runtime.GOARCH]
	possibleOS := osMap[runtime.GOOS]

	var matchingAssets []github.Asset
	var ghExtensions []string

	for _, asset := range assets {
		name := strings.ToLower(asset.Name)

		// Skip checksum and signature files
		if strings.HasSuffix(name, ".sha256") ||
			strings.HasSuffix(name, ".sha512") ||
			strings.HasSuffix(name, ".asc") ||
			strings.HasSuffix(name, ".sig") ||
			strings.HasSuffix(name, ".md5") {
			continue
		}

		// Track GitHub CLI extensions
		if strings.HasPrefix(name, "gh-") {
			ghExtensions = append(ghExtensions, asset.Name)
			continue
		}

		// Check if asset name contains any of our OS variations
		var matchesOS, matchesArch bool
		for _, os := range possibleOS {
			if strings.Contains(name, os) {
				matchesOS = true
				break
			}
		}

		// Check if asset name contains any of our architecture variations
		for _, arch := range possibleArch {
			if strings.Contains(name, arch) {
				matchesArch = true
				break
			}
		}

		// If both OS and architecture match, add to matching assets
		if matchesOS && matchesArch {
			matchingAssets = append(matchingAssets, asset)
		}
	}

	// If we found matching assets, return the first one
	if len(matchingAssets) > 0 {
		return &matchingAssets[0], nil
	}

	// Construct detailed error message
	var errMsg strings.Builder
	errMsg.WriteString(fmt.Sprintf("no matching binary found for %s\n", osArch))

	if len(ghExtensions) > 0 {
		errMsg.WriteString("\nFound GitHub CLI extensions that were skipped:\n")
		for _, ext := range ghExtensions {
			errMsg.WriteString(fmt.Sprintf("- %s\n", ext))
		}
		errMsg.WriteString("\nGitHub CLI extensions should be installed using 'gh extension install' instead.\n")
	}

	return nil, fmt.Errorf(errMsg.String())
}

// findBinaryByRepo returns the binary name and metadata for a given repository
func (m *Manager) findBinaryByRepo(repo string) (string, *metadata.BinaryMetadata, error) {
	entries, err := os.ReadDir(m.binDir)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read bin directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		binaryPath := filepath.Join(m.binDir, entry.Name())
		meta, err := metadata.Load(binaryPath)
		if err != nil {
			continue // Skip binaries without metadata
		}

		if meta.Repository == repo {
			return entry.Name(), meta, nil
		}
	}

	return "", nil, fmt.Errorf("no binary found for repository %s", repo)
}

// Remove removes an installed binary and its metadata
func (m *Manager) Remove(nameOrRepo string) error {
	var binaryName string
	var meta *metadata.BinaryMetadata

	// First try to find by repository name
	foundName, foundMeta, err := m.findBinaryByRepo(nameOrRepo)
	if err == nil {
		binaryName = foundName
		meta = foundMeta
	} else {
		// If not found by repo, try as binary name
		binaryName = nameOrRepo
		binaryPath := filepath.Join(m.binDir, binaryName)

		// Check if the binary exists
		if _, err := os.Stat(binaryPath); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("binary %s is not installed", nameOrRepo)
			}
			return fmt.Errorf("failed to check binary: %w", err)
		}

		// Try to load metadata
		meta, err = metadata.Load(binaryPath)
		if err != nil {
			log.Debug("failed to load metadata", "error", err)
			// Continue with removal even if metadata is missing
		}
	}

	binaryPath := filepath.Join(m.binDir, binaryName)

	// Remove the binary
	if err := os.Remove(binaryPath); err != nil {
		return fmt.Errorf("failed to remove binary: %w", err)
	}

	// Remove metadata if it exists
	if meta != nil {
		if err := metadata.Delete(binaryPath); err != nil {
			log.Debug("failed to remove metadata", "error", err)
			// Don't fail if metadata removal fails
		}
		fmt.Println(ui.FormatActionMessage("Removed", ui.FormatBinaryInfo(binaryName, binaryPath, meta.Version)))
	} else {
		fmt.Println(ui.FormatActionMessage("Removed", ui.FormatBinaryInfo(binaryName, binaryPath, "")))
	}

	return nil
}

// Delete is deprecated, use Remove instead
func (m *Manager) Delete(nameOrRepo string) error {
	return m.Remove(nameOrRepo)
}

// InstalledBinary represents an installed binary and its metadata
type InstalledBinary struct {
	Name           string
	Path           string
	Repository     string
	Version        string
	Host           string
	OriginalBinary string
}

// ListInstalled returns a list of all installed binaries and their metadata
func (m *Manager) ListInstalled() ([]InstalledBinary, error) {
	entries, err := os.ReadDir(m.binDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read bin directory: %w", err)
	}

	log.Debug("found entries in bin directory", "count", len(entries))

	var binaries []InstalledBinary
	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip directories
		}

		binaryName := entry.Name()
		binaryPath := filepath.Join(m.binDir, binaryName)
		log.Debug("processing binary", "name", binaryName, "path", binaryPath)

		// Try to load metadata by binary name
		meta, err := metadata.Load(binaryName)
		if err != nil {
			log.Debug("failed to load metadata by name", "binary", binaryName, "error", err)
			// If that fails, try by full path
			meta, err = metadata.Load(binaryPath)
			if err != nil {
				log.Debug("failed to load metadata by path", "binary", binaryName, "error", err)
				// Include binary with minimal info if metadata is missing
				binaries = append(binaries, InstalledBinary{
					Name: binaryName,
					Path: binaryPath,
				})
				continue
			}
		}

		log.Debug("found metadata", "binary", binaryName, "repo", meta.Repository, "version", meta.Version)
		binaries = append(binaries, InstalledBinary{
			Name:           binaryName,
			Path:           binaryPath,
			Repository:     meta.Repository,
			Version:        meta.Version,
			Host:           meta.GHHost,
			OriginalBinary: meta.OriginalBinary,
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
	if err := m.Install(repo); err != nil {
		return err
	}
	fmt.Println(ui.FormatActionMessage("Updated", ui.FormatBinaryInfo("", "", repo)))
	return nil
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
