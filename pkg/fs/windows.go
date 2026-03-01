//go:build windows

package fs

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/realloser/gh-install-from/pkg/config"
)

// Ensure WindowsFSService implements OSService
var _ OSService = (*WindowsFSService)(nil)

// WindowsFSService implements OSService using .cmd shims on Windows
type WindowsFSService struct{}

func init() {
	RegisterOSService("windows", NewWindowsFSServiceFromConfig)
}

// NewWindowsFSService creates a Windows OSService
func NewWindowsFSService() *WindowsFSService {
	return &WindowsFSService{}
}

// NewWindowsFSServiceFromConfig creates a Windows OSService from config
func NewWindowsFSServiceFromConfig(cfg *config.Config) (OSService, error) {
	slog.Debug("windows OSService created")
	return NewWindowsFSService(), nil
}

// InstallBinary creates a .cmd shim that invokes the target .exe with %*
func (s *WindowsFSService) InstallBinary(binDir, binaryName, targetPath string) error {
	if err := os.MkdirAll(binDir, 0750); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}
	baseName := strings.TrimSuffix(strings.TrimSuffix(binaryName, ".exe"), ".EXE")
	shimPath := filepath.Join(binDir, baseName+".cmd")
	targetPath = filepath.Clean(targetPath)
	content := "@echo off\r\n\"" + targetPath + "\" %*\r\n"
	if err := os.WriteFile(shimPath, []byte(content), 0755); err != nil {
		return fmt.Errorf("failed to create shim %s: %w", shimPath, err)
	}
	slog.Debug("shim created", "shim", shimPath, "target", targetPath)
	return nil
}

// RemoveBinary removes the .cmd shim from binDir
func (s *WindowsFSService) RemoveBinary(binDir, binaryName string) error {
	baseName := strings.TrimSuffix(strings.TrimSuffix(binaryName, ".exe"), ".EXE")
	shimPath := filepath.Join(binDir, baseName+".cmd")
	if err := os.Remove(shimPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to remove shim %s: %w", shimPath, err)
	}
	slog.Debug("shim removed", "shim", shimPath)
	return nil
}
