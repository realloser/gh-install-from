package binary

import (
	"errors"
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

// ErrPathSetup indicates the install failed at the symlink/shim (PATH) step,
// meaning the user likely needs to run 'gh install-from init'. Other install
// errors (repo not found, no matching asset, download failure) are unrelated
// to PATH setup and should not show the init hint.
var ErrPathSetup = errors.New("path setup required")

// processor turns a downloaded release asset (at src) into a single binary file
// located under destDir, returning the resulting path.
type processor interface {
	Process(src, destDir string) (string, error)
}

// Ensure archive processors implement processor
var _ processor = (*archive.ArchiveProcessor)(nil)
var _ processor = (*archive.BinaryProcessor)(nil)

// Confirmer asks the user a yes/no question. Used for the macOS quarantine
// removal prompt so the manager never touches the terminal directly.
// Implementations decide whether a TTY is present and return false without
// prompting if they cannot interact (e.g. CI).
type Confirmer interface {
	Confirm(prompt string) bool
}

type managerImpl struct {
	pathMgr          *path.Manager
	client           github.Client
	store            metadata.MetadataStore
	osSvc            fs.OSService
	archiveProcessor processor
	binaryProcessor  processor
	removeQuarantine bool
	confirmer        Confirmer
}

// WithConfirmer injects a Confirmer for the macOS quarantine prompt.
// Used by the cmd layer to provide the TTY implementation.
func (m *managerImpl) WithConfirmer(c Confirmer) *managerImpl {
	m.confirmer = c
	return m
}

// WithRemoveQuarantine sets the --remove-quarantine flag (non-interactive
// escape hatch that strips the quarantine xattr without prompting).
func (m *managerImpl) WithRemoveQuarantine(b bool) *managerImpl {
	m.removeQuarantine = b
	return m
}

// ConfigureQuarantine wires the Confirmer and --remove-quarantine flag into a
// Manager built by NewManager. It is the exported entry point the cmd layer
// uses (managerImpl is unexported). It returns the same Manager.
func ConfigureQuarantine(m Manager, c Confirmer, remove bool) Manager {
	if impl, ok := m.(*managerImpl); ok {
		impl.confirmer = c
		impl.removeQuarantine = remove
	}
	return m
}

func (m *managerImpl) Install(repo string) error {
	slog.Info("fetching latest release", "repo", repo)
	release, err := m.client.GetLatestRelease(repo)
	if err != nil {
		return fmt.Errorf("failed to get latest release: %w", err)
	}
	if len(release.Assets) == 0 {
		return fmt.Errorf("repository %s has no assets in its latest release\n\nThis may be a source-only release. Check the release page for downloadable binaries", repo)
	}

	asset, err := findMatchingAsset(release.Assets)
	if err != nil {
		return fmt.Errorf("failed to find matching asset for %s: %w\n\nNo asset matched your OS (%s) and arch (%s). Check the release page for available assets, or pass a full URL to a specific asset", repo, err, runtime.GOOS, runtime.GOARCH)
	}

	tmpDir, err := os.MkdirTemp("", "gh-install-from-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Download into a dedicated .src subdir so the processed output (in tmpDir)
	// can never collide with the download path for either processor. This
	// permanently removes the self-truncate class of bug.
	srcSubDir := filepath.Join(tmpDir, ".src")
	if err := os.MkdirAll(srcSubDir, 0700); err != nil {
		return fmt.Errorf("failed to create download subdirectory: %w", err)
	}
	tmpPath := filepath.Join(srcSubDir, asset.Name)
	slog.Info("downloading asset", "name", asset.Name)
	if err := m.client.DownloadAsset(asset.BrowserDownloadURL, tmpPath); err != nil {
		return fmt.Errorf("failed to download asset: %w", err)
	}

	binaryPath, err := m.processDownload(tmpPath, tmpDir)
	if err != nil {
		return fmt.Errorf("failed to process downloaded file: %w", err)
	}

	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repository format: %s\n\nUse owner/repo (e.g. BurntSushi/ripgrep) or a full URL (e.g. https://github.com/BurntSushi/ripgrep)", repo)
	}
	owner, repoName := parts[0], parts[1]
	tag := strings.TrimPrefix(release.TagName, "v")

	downloadsDir := filepath.Join(m.pathMgr.GetDownloadsDir(), owner, repoName, tag)
	if err := os.MkdirAll(downloadsDir, 0750); err != nil {
		return fmt.Errorf("failed to create downloads directory: %w", err)
	}

	extractedName := filepath.Base(binaryPath)
	targetPath := filepath.Join(downloadsDir, extractedName)
	if err := copyFile(binaryPath, targetPath); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(targetPath, 0755); err != nil {
			return fmt.Errorf("failed to chmod binary: %w", err)
		}
	}

	binaryName := getBinaryName(repo)
	binDir := m.pathMgr.GetBinDir()
	slog.Info("installing binary", "name", binaryName)
	if err := m.osSvc.InstallBinary(binDir, binaryName, targetPath); err != nil {
		return fmt.Errorf("failed to create symlink/shim: %w", errors.Join(err, ErrPathSetup))
	}

	binPath := filepath.Join(binDir, binaryName)
	if runtime.GOOS == "windows" {
		binPath = filepath.Join(binDir, binaryName+".cmd")
	}

	// macOS quarantine handling: detect whether Gatekeeper blocked the
	// installed binary, and if so offer to strip the com.apple.quarantine
	// xattr. Stripping NEVER happens without explicit confirmation — via
	// the interactive Confirmer prompt or the --remove-quarantine flag.
	// Detection is advisory and never aborts the install.
	if runtime.GOOS == "darwin" {
		m.handleQuarantine(binPath, binaryName, targetPath)
	}

	meta := &metadata.BinaryMetadata{
		GHHost:         m.client.GetHost(),
		Repository:     repo,
		Version:        release.TagName,
		BinaryPath:     binPath,
		OriginalBinary: extractedName,
	}
	if err := m.store.Store(meta); err != nil {
		slog.Warn("failed to store metadata; 'list' and 'update' may not track this binary", "error", err)
	}

	fmt.Println(ui.FormatActionMessage("Installed", ui.FormatBinaryInfo(binaryName, binPath, release.TagName)))
	return nil
}

// processDownload routes a downloaded asset to the right processor based on its
// detected content. Archives go through the archive processor; if the archive
// holds no executable member (ErrNoExecutable), the whole file is treated as a
// bare binary. Everything else is handled by the binary processor.
func (m *managerImpl) processDownload(src, destDir string) (string, error) {
	format, err := archive.DetectFormat(src)
	if err != nil {
		return "", err
	}
	switch format {
	case archive.FormatArchive:
		p, perr := m.archiveProcessor.Process(src, destDir)
		if errors.Is(perr, archive.ErrNoExecutable) {
			// Archive had no executable member — treat the whole file as a bare binary.
			return m.binaryProcessor.Process(src, destDir)
		}
		return p, perr
	case archive.FormatBinary:
		return m.binaryProcessor.Process(src, destDir)
	default:
		return "", archive.ErrUnsupportedFormat
	}
}

// handleQuarantine detects whether macOS Gatekeeper blocked the installed
// binary and, if so, offers to strip the com.apple.quarantine xattr. Stripping
// requires explicit confirmation — via the --remove-quarantine flag (escape
// hatch) or the interactive Confirmer prompt. It never happens automatically.
// Detection is advisory: this method never returns an error and never aborts
// the install. No-op on non-darwin (caller gates on runtime.GOOS).
func (m *managerImpl) handleQuarantine(binPath, binaryName, targetPath string) {
	quarantined, _ := detectQuarantine(binPath)
	if !quarantined {
		return
	}

	// Confirm the attribute is actually present before acting — a binary can
	// be signal-killed for other reasons. HasQuarantine is the strongest
	// discriminator.
	has, err := fs.HasQuarantine(targetPath)
	if err != nil {
		slog.Debug("failed to check quarantine attribute", "path", targetPath, "error", err)
		return
	}
	if !has {
		return
	}

	strip := m.shouldStripQuarantine(binaryName)

	if strip {
		if err := fs.RemoveQuarantine(targetPath); err != nil {
			fmt.Println(ui.FormatErrorMessage(fmt.Sprintf(
				"failed to remove quarantine attribute from %s: %v", binaryName, err)))
			return
		}
		fmt.Println(ui.FormatActionMessage("Removed quarantine attribute", binaryName))
		return
	}

	// Not stripped — print guidance so the user knows how to resolve it.
	fmt.Println(ui.FormatErrorMessage(fmt.Sprintf(
		"macOS has quarantined %s — Gatekeeper may block it from running.\n"+
			"To fix, either:\n"+
			"  - rerun with: gh install-from --remove-quarantine <owner/repo>\n"+
			"  - approve it in System Settings > Privacy & Security > Security\n"+
			"  - run manually: xattr -d com.apple.quarantine %s",
		binaryName, targetPath)))
}

// shouldStripQuarantine decides whether to strip the quarantine attribute,
// given that detection confirmed the binary is quarantined. Stripping requires
// explicit confirmation: the --remove-quarantine flag (escape hatch, no prompt)
// or an affirmative interactive Confirmer prompt. It NEVER strips by default
// (no confirmer = CI / non-interactive).
func (m *managerImpl) shouldStripQuarantine(binaryName string) bool {
	switch {
	case m.removeQuarantine:
		// --remove-quarantine flag: non-interactive escape hatch, strip directly.
		return true
	case m.confirmer != nil:
		// Interactive prompt — the prompt IS the confirmation.
		return m.confirmer.Confirm(fmt.Sprintf(
			"macOS quarantined %s — Gatekeeper may block it from running.\nRemove the quarantine attribute now? [y/N] ",
			binaryName))
	default:
		// No confirmer (e.g. CI / non-interactive) — never auto-strip.
		return false
	}
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
	slog.Info("removing binary", "name", nameOrRepo)
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
		slog.Warn("failed to remove metadata; the binary was removed but its metadata record may persist", "error", err)
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

	return nil, fmt.Errorf("binary %s is not installed\n\nRun 'gh install-from list' to see installed binaries", nameOrRepo)
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
	var plainCandidates []github.Asset
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
		// A plain-name candidate (e.g. a bare binary named "mycli" or
		// "mycli-1.2.3") is one whose name is platform-agnostic: it contains no
		// OS or arch token from ANY platform. A name like "mycli-linux-amd64"
		// embeds a specific platform, so it is never treated as a plain
		// candidate — we must not silently install a wrong-platform binary.
		if !isPlatformTagged(name, osMap, archMap) {
			plainCandidates = append(plainCandidates, asset)
		}
	}
	if len(matchingAssets) > 0 {
		return &matchingAssets[0], nil
	}
	// No OS/arch-tagged asset matched. Accept a single plain (platform-agnostic)
	// candidate — Failure A. If there is more than one, it's ambiguous; don't
	// guess.
	if len(plainCandidates) == 1 {
		return &plainCandidates[0], nil
	}
	var errMsg strings.Builder
	fmt.Fprintf(&errMsg, "no matching binary found for %s_%s\n", runtime.GOOS, runtime.GOARCH)
	if len(ghExtensions) > 0 {
		errMsg.WriteString("\nFound GitHub CLI extensions that were skipped.\n")
	}
	return nil, fmt.Errorf("%s", errMsg.String())
}

// isPlatformTagged reports whether an asset name embeds any OS or arch token
// from ANY supported platform. A name like "mycli" or "mycli-1.2.3" is
// platform-agnostic and returns false; "mycli-linux-amd64" or "mycli-win" is
// tagged and returns true. Used to keep the plain-candidate fallback from
// installing a wrong-platform binary.
func isPlatformTagged(name string, osMap, archMap map[string][]string) bool {
	for _, tokens := range osMap {
		for _, t := range tokens {
			if strings.Contains(name, t) {
				return true
			}
		}
	}
	for _, tokens := range archMap {
		for _, t := range tokens {
			if strings.Contains(name, t) {
				return true
			}
		}
	}
	return false
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
