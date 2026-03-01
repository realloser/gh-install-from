package github

import (
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/realloser/gh-install-from/pkg/config"
)

type clientFactory func(*config.Config) (Client, error)

var (
	clientRegistry   = make(map[string]clientFactory)
	clientRegistryMu sync.RWMutex
)

func init() {
	RegisterClient("gh", func(cfg *config.Config) (Client, error) {
		return newGhCliClient()
	})
	RegisterClient("mock", func(cfg *config.Config) (Client, error) {
		return &MockClient{}, nil
	})
}

// RegisterClient registers a client adapter
func RegisterClient(name string, fn clientFactory) {
	clientRegistryMu.Lock()
	defer clientRegistryMu.Unlock()
	clientRegistry[name] = fn
}

// NewClient returns a Client from the registry based on config
func NewClient(cfg *config.Config) (Client, error) {
	if cfg == nil {
		cfg = config.FromEnv()
	}
	name := cfg.Client
	if name == "" {
		name = os.Getenv("GH_INSTALL_FROM_CLIENT")
	}
	if name == "" {
		name = "gh"
	}
	clientRegistryMu.RLock()
	fn, ok := clientRegistry[name]
	clientRegistryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown client adapter: %s", name)
	}
	slog.Debug("client adapter selected", "adapter", name)
	return fn(cfg)
}
