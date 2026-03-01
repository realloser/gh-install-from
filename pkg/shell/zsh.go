package shell

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/realloser/gh-install-from/pkg/config"
)

// Ensure ZshAdapter implements ShellAdapter
var _ ShellAdapter = (*ZshAdapter)(nil)

// ZshAdapter implements ShellAdapter for Zsh
type ZshAdapter struct{}

func init() {
	RegisterShell("zsh", NewZshAdapterFromConfig)
}

// NewZshAdapter creates a Zsh ShellAdapter
func NewZshAdapter() *ZshAdapter {
	return &ZshAdapter{}
}

// NewZshAdapterFromConfig creates a Zsh ShellAdapter from config
func NewZshAdapterFromConfig(cfg *config.Config) (ShellAdapter, error) {
	slog.Debug("zsh adapter created")
	return NewZshAdapter(), nil
}

// ApplyHint returns the Zsh-specific instruction to apply PATH changes
func (a *ZshAdapter) ApplyHint() string {
	return "Restart your shell or run 'source ~/.zshrc' to apply."
}

// PostInitMessage returns the Zsh-specific message after AddToPATH
func (a *ZshAdapter) PostInitMessage(binPath string) string {
	return fmt.Sprintf("Added %s to PATH. %s\n", binPath, a.ApplyHint())
}

// AddToPATH appends export PATH line to ~/.zshrc (idempotent)
func (a *ZshAdapter) AddToPATH(binPath string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	rcPath := filepath.Join(homeDir, ".zshrc")
	line := fmt.Sprintf("\nexport PATH=\"%s:$PATH\"\n", binPath)
	return appendIfNotPresent(rcPath, line, binPath)
}
