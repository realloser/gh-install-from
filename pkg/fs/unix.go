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
	if err := os.Symlink(targetPath, linkPath); err != nil {
		return fmt.Errorf("failed to create symlink %s -> %s: %w", linkPath, targetPath, err)
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
