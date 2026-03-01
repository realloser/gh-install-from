//go:build windows

package shell

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/realloser/gh-install-from/pkg/config"
)

// Ensure PowerShellAdapter implements ShellAdapter
var _ ShellAdapter = (*PowerShellAdapter)(nil)

// PowerShellAdapter implements ShellAdapter for PowerShell on Windows
type PowerShellAdapter struct{}

func init() {
	RegisterShell("powershell", NewPowerShellAdapterFromConfig)
	RegisterShell("pwsh", NewPowerShellAdapterFromConfig)
}

// NewPowerShellAdapter creates a PowerShell ShellAdapter
func NewPowerShellAdapter() *PowerShellAdapter {
	return &PowerShellAdapter{}
}

// NewPowerShellAdapterFromConfig creates a PowerShell ShellAdapter from config
func NewPowerShellAdapterFromConfig(cfg *config.Config) (ShellAdapter, error) {
	slog.Debug("powershell adapter created")
	return NewPowerShellAdapter(), nil
}

// ApplyHint returns the PowerShell-specific instruction to apply PATH changes
func (a *PowerShellAdapter) ApplyHint() string {
	return "Restart your terminal or run '. $PROFILE' to apply."
}

// PostInitMessage returns the PowerShell-specific message after AddToPATH
func (a *PowerShellAdapter) PostInitMessage(binPath string) string {
	return fmt.Sprintf("Added %s to PATH. %s\n", binPath, a.ApplyHint())
}

// AddToPATH updates User Path environment variable (idempotent)
func (a *PowerShellAdapter) AddToPATH(binPath string) error {
	pathEnv := os.Getenv("Path")
	if pathEnv == "" {
		pathEnv = os.Getenv("PATH")
	}
	paths := filepath.SplitList(pathEnv)
	for _, p := range paths {
		if strings.TrimSpace(p) == binPath {
			return nil // already in PATH
		}
	}
	// Append to profile so it persists
	profileDir := filepath.Join(os.Getenv("USERPROFILE"), "Documents", "WindowsPowerShell")
	if err := os.MkdirAll(profileDir, 0750); err != nil {
		return fmt.Errorf("failed to create profile directory: %w", err)
	}
	profilePath := filepath.Join(profileDir, "Microsoft.PowerShell_profile.ps1")
	line := fmt.Sprintf("\n$env:Path = \"%s;\" + $env:Path\n", binPath)
	return appendIfNotPresent(profilePath, line, binPath)
}
