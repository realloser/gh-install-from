package shell

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/realloser/gh-install-from/pkg/config"
)

// Ensure BashAdapter implements ShellAdapter
var _ ShellAdapter = (*BashAdapter)(nil)

// BashAdapter implements ShellAdapter for Bash
type BashAdapter struct{}

func init() {
	RegisterShell("bash", NewBashAdapterFromConfig)
}

// NewBashAdapter creates a Bash ShellAdapter
func NewBashAdapter() *BashAdapter {
	return &BashAdapter{}
}

// NewBashAdapterFromConfig creates a Bash ShellAdapter from config
func NewBashAdapterFromConfig(cfg *config.Config) (ShellAdapter, error) {
	slog.Debug("bash adapter created")
	return NewBashAdapter(), nil
}

// ApplyHint returns the Bash-specific instruction to apply PATH changes
func (a *BashAdapter) ApplyHint() string {
	return "Restart your shell or run 'source ~/.bashrc' to apply."
}

// PostInitMessage returns the Bash-specific message after AddToPATH
func (a *BashAdapter) PostInitMessage(binPath string) string {
	return fmt.Sprintf("Added %s to PATH. %s\n", binPath, a.ApplyHint())
}

// AddToPATH appends export PATH line to ~/.bashrc (idempotent)
func (a *BashAdapter) AddToPATH(binPath string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	rcPath := filepath.Join(homeDir, ".bashrc")
	line := fmt.Sprintf("\nexport PATH=\"%s:$PATH\"\n", binPath)
	return appendIfNotPresent(rcPath, line, binPath)
}
