package fs

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/realloser/gh-install-from/pkg/config"
)

type osServiceFactory func(*config.Config) (OSService, error)

var (
	osServiceRegistry   = make(map[string]osServiceFactory)
	osServiceRegistryMu sync.RWMutex
)

// RegisterOSService registers an OSService adapter
func RegisterOSService(name string, fn osServiceFactory) {
	osServiceRegistryMu.Lock()
	defer osServiceRegistryMu.Unlock()
	osServiceRegistry[name] = fn
}

// NewOSService returns an OSService from the registry based on config/OS
func NewOSService(cfg *config.Config) (OSService, error) {
	if cfg == nil {
		cfg = config.FromEnv()
	}
	name := cfg.OS
	if name == "" {
		name = os.Getenv("GH_INSTALL_FROM_OS")
	}
	if name == "" {
		name = runtime.GOOS
	}
	name = strings.TrimSpace(strings.ToLower(name))

	// Map "darwin" to "unix" for registry lookup - unix adapter handles darwin, linux
	if name == "darwin" || name == "linux" {
		name = "unix"
	}
	if name == "windows" {
		name = "windows"
	}

	osServiceRegistryMu.RLock()
	fn, ok := osServiceRegistry[name]
	osServiceRegistryMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown OSService adapter: %s", name)
	}

	slog.Debug("OSService adapter selected", "adapter", name)
	return fn(cfg)
}
