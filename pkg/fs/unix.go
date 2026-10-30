//go:build !windows

package fs

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/realloser/gh-install-from/pkg/config"
)

// Ensure UnixFSService implements OSService
var _ OSService = (*UnixFSService)(nil)

// UnixFSService implements OSService using symlinks on Unix
type UnixFSService struct{}

func init() {
	RegisterOSService("unix", NewUnixFSServiceFromConfig)
	RegisterOSService("darwin", NewUnixFSServiceFromConfig)
	RegisterOSService("linux", NewUnixFSServiceFromConfig)
}

// NewUnixFSService creates a Unix OSService
func NewUnixFSService() *UnixFSService {
	return &UnixFSService{}
}

// NewUnixFSServiceFromConfig creates a Unix OSService from config
func NewUnixFSServiceFromConfig(cfg *config.Config) (OSService, error) {
	slog.Debug("unix OSService created")
	return NewUnixFSService(), nil
}

// InstallBinary creates a symlink in binDir pointing to targetPath
func (s *UnixFSService) InstallBinary(binDir, binaryName, targetPath string) error {
	if err := os.MkdirAll(binDir, 0750); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}
	linkPath := filepath.Join(binDir, binaryName)
	// Never clobber anything that is not a symlink (e.g. a user's own binary).
	if fi, err := os.Lstat(linkPath); err == nil && fi.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("refusing to replace %s: it exists and is not a symlink", linkPath)
	}
	// Build the new link beside the old one and rename it into place. rename(2)
	// swaps atomically, so the command never disappears mid-update and a failed
	// attempt leaves the previous version untouched.
	removeStaleTempLinks(linkPath)
	tmpLink := fmt.Sprintf("%s.tmp-%d", linkPath, os.Getpid())
	if err := os.Symlink(targetPath, tmpLink); err != nil {
		return fmt.Errorf("failed to create symlink %s -> %s: %w", linkPath, targetPath, err)
	}
	if err := os.Rename(tmpLink, linkPath); err != nil {
		_ = os.Remove(tmpLink)
		return fmt.Errorf("failed to replace symlink %s: %w", linkPath, err)
	}
	slog.Debug("symlink created", "link", linkPath, "target", targetPath)
	return nil
}

// RemoveBinary removes the symlink from binDir
func (s *UnixFSService) RemoveBinary(binDir, binaryName string) error {
	linkPath := filepath.Join(binDir, binaryName)
	if err := os.Remove(linkPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to remove symlink %s: %w", linkPath, err)
	}
	slog.Debug("symlink removed", "link", linkPath)
	return nil
}

// removeStaleTempLinks deletes "<link>.tmp-<pid>" symlinks left behind by an
// install that was killed between creating the temporary link and renaming it.
// Only symlinks are removed.
func removeStaleTempLinks(linkPath string) {
	matches, err := filepath.Glob(linkPath + ".tmp-*")
	if err != nil {
		return
	}
	for _, m := range matches {
		if fi, err := os.Lstat(m); err == nil && fi.Mode()&os.ModeSymlink != 0 {
			_ = os.Remove(m)
		}
	}
}
