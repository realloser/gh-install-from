package metadata

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/realloser/gh-install-from/pkg/config"
)

type storeFactory func(*config.Config) (MetadataStore, error)

var (
	storeRegistry   = make(map[string]storeFactory)
	storeRegistryMu sync.RWMutex
)

// RegisterStore registers a store adapter
func RegisterStore(name string, fn storeFactory) {
	storeRegistryMu.Lock()
	defer storeRegistryMu.Unlock()
	storeRegistry[name] = fn
}

// NewStore returns a MetadataStore from the registry based on config
func NewStore(cfg *config.Config) (MetadataStore, error) {
	if cfg == nil {
		cfg = config.FromEnv()
	}
	name := cfg.Store
	if name == "" {
		name = os.Getenv("GH_INSTALL_FROM_STORE")
	}
	if name == "" {
		name = "json"
	}
	name = strings.TrimSpace(strings.ToLower(name))

	storeRegistryMu.RLock()
	fn, ok := storeRegistry[name]
	storeRegistryMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown store adapter: %s", name)
	}

	slog.Debug("store adapter selected", "adapter", name)
	return fn(cfg)
}
