package binary

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/realloser/gh-install-from/pkg/archive"
	"github.com/realloser/gh-install-from/pkg/fs"
	"github.com/realloser/gh-install-from/pkg/github"
	"github.com/realloser/gh-install-from/pkg/metadata"
	"github.com/realloser/gh-install-from/pkg/path"
	"github.com/realloser/gh-install-from/pkg/ui"
)

// Ensure managerImpl implements Manager
var _ Manager = (*managerImpl)(nil)

type archiver interface {
	ExtractFile(src, destDir string) (string, error)
}

// Ensure archive.Archiver implements archiver
var _ archiver = (*archive.Archiver)(nil)

type managerImpl struct {
	pathMgr  *path.Manager
	client   github.Client
	store    metadata.MetadataStore
	osSvc    fs.OSService
	archiver archiver
}

func (m *managerImpl) Install(repo string) error {
	slog.Debug("fetching latest release", "repo", repo)
	release, err := m.client.GetLatestRelease(repo)
	if err != nil {
		return fmt.Errorf("failed to get latest release from %s: %w", repo, err)
	}
	if len(release.Assets) == 0 {
		return fmt.Errorf("repository %s has no assets in its latest release", repo)
	}

	asset, err := findMatchingAsset(release.Assets)
	if err != nil {
		return fmt.Errorf("failed to find matching asset for %s: %w", repo, err)
	}

	tmpDir, err := os.MkdirTemp("", "gh-install-from-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpPath := filepath.Join(tmpDir, asset.Name)
	slog.Debug("downloading asset", "path", tmpPath, "url", asset.BrowserDownloadURL)
	if err := m.client.DownloadAsset(asset.BrowserDownloadURL, tmpPath); err != nil {
		return fmt.Errorf("failed to download asset: %w", err)
	}

	extractedPath, err := m.archiver.ExtractFile(tmpPath, tmpDir)
	if err != nil {
		return fmt.Errorf("failed to extract file: %w", err)
	}

	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repository format: %s", repo)
	}
	owner, repoName := parts[0], parts[1]
	tag := strings.TrimPrefix(release.TagName, "v")

	downloadsDir := filepath.Join(m.pathMgr.GetDownloadsDir(), owner, repoName, tag)
	if err := os.MkdirAll(downloadsDir, 0750); err != nil {
		return fmt.Errorf("failed to create downloads directory: %w", err)
	}

	extractedName := filepath.Base(extractedPath)
	targetPath := filepath.Join(downloadsDir, extractedName)
	if err := copyFile(extractedPath, targetPath); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(targetPath, 0755); err != nil {
			return fmt.Errorf("failed to chmod binary: %w", err)
		}
	}

	binaryName := getBinaryName(repo)
	binDir := m.pathMgr.GetBinDir()
	if err := m.osSvc.InstallBinary(binDir, binaryName, targetPath); err != nil {
		return fmt.Errorf("failed to create symlink/shim: %w", err)
	}

	binPath := filepath.Join(binDir, binaryName)
	if runtime.GOOS == "windows" {
		binPath = filepath.Join(binDir, binaryName+".cmd")
	}

	meta := &metadata.BinaryMetadata{
		GHHost:         m.client.GetHost(),
		Repository:     repo,
		Version:        release.TagName,
		BinaryPath:     binPath,
		OriginalBinary: extractedName,
	}
	if err := m.store.Store(meta); err != nil {
		slog.Debug("failed to store metadata", "error", err)
	}

	fmt.Println(ui.FormatActionMessage("Installed", ui.FormatBinaryInfo(binaryName, binPath, release.TagName)))
	return nil
}

func (m *managerImpl) Update(repo string) error {
	slog.Info("updating binary", "repo", repo)
	return m.Install(repo)
}

func (m *managerImpl) UpdateAll() error {
	slog.Info("updating all installed binaries")
	binaries, err := m.ListInstalled()
	if err != nil {
		return fmt.Errorf("failed to list installed: %w", err)
	}

	if len(binaries) == 0 {
		slog.Debug("no binaries installed")
		return nil
	}

	candidates, err := CheckUpdates(binaries, m.client, 0)
	if err != nil {
		return fmt.Errorf("failed to check updates: %w", err)
	}

	if len(candidates) == 0 {
		slog.Info("all binaries up to date")
		return nil
	}

	var updateErrors []string
	for _, c := range candidates {
		if err := m.Update(c.InstalledBinary.Repository); err != nil {
			updateErrors = append(updateErrors, fmt.Sprintf("failed to update %s: %v", c.InstalledBinary.Repository, err))
		}
	}

	if len(updateErrors) > 0 {
		return fmt.Errorf("update completed with errors:\n%s", strings.Join(updateErrors, "\n"))
	}
	return nil
}

func (m *managerImpl) Remove(nameOrRepo string) error {
	meta, err := m.findMeta(nameOrRepo)
	if err != nil {
		return err
	}

	binaryName := filepath.Base(meta.BinaryPath)
	if runtime.GOOS == "windows" && strings.HasSuffix(binaryName, ".cmd") {
		binaryName = strings.TrimSuffix(binaryName, ".cmd")
	}

	binDir := m.pathMgr.GetBinDir()
	if err := m.osSvc.RemoveBinary(binDir, binaryName); err != nil {
		return fmt.Errorf("failed to remove binary: %w", err)
	}

	if err := m.store.Delete(binaryName); err != nil {
		slog.Debug("failed to remove metadata", "error", err)
	}

	fmt.Println(ui.FormatActionMessage("Removed", ui.FormatBinaryInfo(binaryName, meta.BinaryPath, meta.Version)))
	return nil
}

func (m *managerImpl) ListInstalled() ([]InstalledBinary, error) {
	metaList, err := m.store.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list metadata: %w", err)
	}

	var result []InstalledBinary
	for _, meta := range metaList {
		name := filepath.Base(meta.BinaryPath)
		if runtime.GOOS == "windows" && strings.HasSuffix(name, ".cmd") {
			name = strings.TrimSuffix(name, ".cmd")
		}
		result = append(result, InstalledBinary{
			Name:           name,
			Path:           meta.BinaryPath,
			Repository:     meta.Repository,
			Version:        meta.Version,
			Host:           meta.GHHost,
			OriginalBinary: meta.OriginalBinary,
		})
	}
	return result, nil
}

func (m *managerImpl) GetBinDir() string {
	return m.pathMgr.GetBinDir()
}

func (m *managerImpl) CheckUpdates(binaries []InstalledBinary) ([]UpdateCandidate, error) {
	return CheckUpdates(binaries, m.client, 0)
}

func (m *managerImpl) findMeta(nameOrRepo string) (*metadata.BinaryMetadata, error) {
	metaList, err := m.store.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list metadata: %w", err)
	}

	for _, meta := range metaList {
		if meta.Repository == nameOrRepo {
			return meta, nil
		}
		binaryName := filepath.Base(meta.BinaryPath)
		if runtime.GOOS == "windows" && strings.HasSuffix(binaryName, ".cmd") {
			binaryName = strings.TrimSuffix(binaryName, ".cmd")
		}
		if binaryName == nameOrRepo {
			return meta, nil
		}
	}

	return nil, fmt.Errorf("binary %s is not installed", nameOrRepo)
}

func findMatchingAsset(assets []github.Asset) (*github.Asset, error) {
	archMap := map[string][]string{
		"amd64": {"x86_64", "amd64", "x64"},
		"386":   {"i386", "x86", "386"},
		"arm64": {"arm64", "aarch64"},
	}
	osMap := map[string][]string{
		"darwin":  {"darwin", "macos", "osx", "mac"},
		"linux":   {"linux"},
		"windows": {"windows", "win"},
	}
	possibleArch := archMap[runtime.GOARCH]
	possibleOS := osMap[runtime.GOOS]

	var matchingAssets []github.Asset
	var ghExtensions []string

	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		if strings.HasSuffix(name, ".sha256") || strings.HasSuffix(name, ".sha512") ||
			strings.HasSuffix(name, ".asc") || strings.HasSuffix(name, ".sig") || strings.HasSuffix(name, ".md5") {
			continue
		}
		if strings.HasPrefix(name, "gh-") {
			ghExtensions = append(ghExtensions, asset.Name)
			continue
		}
		var matchesOS, matchesArch bool
		for _, osVar := range possibleOS {
			if strings.Contains(name, osVar) {
				matchesOS = true
				break
			}
		}
		for _, arch := range possibleArch {
			if strings.Contains(name, arch) {
				matchesArch = true
				break
			}
		}
		if matchesOS && matchesArch {
			matchingAssets = append(matchingAssets, asset)
		}
	}
	if len(matchingAssets) > 0 {
		return &matchingAssets[0], nil
	}
	var errMsg strings.Builder
	errMsg.WriteString(fmt.Sprintf("no matching binary found for %s_%s\n", runtime.GOOS, runtime.GOARCH))
	if len(ghExtensions) > 0 {
		errMsg.WriteString("\nFound GitHub CLI extensions that were skipped.\n")
	}
	return nil, fmt.Errorf("%s", errMsg.String())
}

func getBinaryName(repo string) string {
	knownBinaries := map[string]string{"BurntSushi/ripgrep": "rg"}
	if name, ok := knownBinaries[repo]; ok {
		return name
	}
	parts := strings.Split(repo, "/")
	if len(parts) > 1 {
		return parts[1]
	}
	return repo
}

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
