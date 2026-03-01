package shell

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/realloser/gh-install-from/pkg/config"
)

type shellFactory func(*config.Config) (ShellAdapter, error)

var (
	shellRegistry   = make(map[string]shellFactory)
	shellRegistryMu sync.RWMutex
)

// RegisterShell registers a shell adapter
func RegisterShell(name string, fn shellFactory) {
	shellRegistryMu.Lock()
	defer shellRegistryMu.Unlock()
	shellRegistry[strings.ToLower(name)] = fn
}

// NewShellAdapter returns a ShellAdapter from the registry based on config or auto-detected shell
func NewShellAdapter(cfg *config.Config) (ShellAdapter, error) {
	if cfg == nil {
		cfg = config.FromEnv()
	}
	name := cfg.Shell
	if name == "" {
		name = os.Getenv("GH_INSTALL_FROM_SHELL")
	}
	if name == "" {
		name = detectShell()
	}
	name = strings.TrimSpace(strings.ToLower(name))

	shellRegistryMu.RLock()
	fn, ok := shellRegistry[name]
	shellRegistryMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown shell adapter: %s", name)
	}

	slog.Debug("shell adapter selected", "adapter", name)
	return fn(cfg)
}
