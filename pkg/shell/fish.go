package shell

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/realloser/gh-install-from/pkg/config"
)

// Ensure FishAdapter implements ShellAdapter
var _ ShellAdapter = (*FishAdapter)(nil)

// FishAdapter implements ShellAdapter for Fish
type FishAdapter struct{}

func init() {
	RegisterShell("fish", NewFishAdapterFromConfig)
}

// NewFishAdapter creates a Fish ShellAdapter
func NewFishAdapter() *FishAdapter {
	return &FishAdapter{}
}

// NewFishAdapterFromConfig creates a Fish ShellAdapter from config
func NewFishAdapterFromConfig(cfg *config.Config) (ShellAdapter, error) {
	slog.Debug("fish adapter created")
	return NewFishAdapter(), nil
}

// ApplyHint returns the Fish-specific instruction to apply PATH changes
func (a *FishAdapter) ApplyHint() string {
	return "Restart your shell or open a new fish session to apply."
}

// PostInitMessage returns the Fish-specific message after AddToPATH
func (a *FishAdapter) PostInitMessage(binPath string) string {
	return fmt.Sprintf("Added %s to fish_user_paths. %s\n", binPath, a.ApplyHint())
}

// AddToPATH adds bin path to fish_user_paths (idempotent)
func (a *FishAdapter) AddToPATH(binPath string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	configDir := filepath.Join(homeDir, ".config", "fish")
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return fmt.Errorf("failed to create fish config directory: %w", err)
	}
	rcPath := filepath.Join(configDir, "config.fish")
	line := fmt.Sprintf("\nset -U fish_user_paths %s $fish_user_paths\n", binPath)
	// Fish uses set -U; check for existing path in config
	return appendIfNotPresent(rcPath, line, binPath)
}
