package binary

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/realloser/gh-install-from/pkg/archive"
	"github.com/realloser/gh-install-from/pkg/config"
	"github.com/realloser/gh-install-from/pkg/fs"
	"github.com/realloser/gh-install-from/pkg/github"
	"github.com/realloser/gh-install-from/pkg/metadata"
	"github.com/realloser/gh-install-from/pkg/path"
)

type managerFactory func(*config.Config) (Manager, error)

var (
	managerRegistry   = make(map[string]managerFactory)
	managerRegistryMu sync.RWMutex
)

func init() {
	RegisterManager("darwin", newManagerFromConfig)
	RegisterManager("linux", newManagerFromConfig)
	RegisterManager("windows", newManagerFromConfig)
}

// RegisterManager registers a manager adapter
func RegisterManager(name string, fn managerFactory) {
	managerRegistryMu.Lock()
	defer managerRegistryMu.Unlock()
	managerRegistry[strings.ToLower(name)] = fn
}

// NewManager returns a Manager from the registry based on config/OS
func NewManager(cfg *config.Config) (Manager, error) {
	if cfg == nil {
		cfg = config.FromEnv()
	}
	name := cfg.ManagerOS
	if name == "" {
		name = os.Getenv("GH_INSTALL_FROM_MANAGER_OS")
	}
	if name == "" {
		name = runtime.GOOS
	}
	name = strings.TrimSpace(strings.ToLower(name))

	managerRegistryMu.RLock()
	fn, ok := managerRegistry[name]
	managerRegistryMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown manager adapter: %s", name)
	}

	slog.Debug("manager adapter selected", "adapter", name)
	return fn(cfg)
}

func newManagerFromConfig(cfg *config.Config) (Manager, error) {
	pathMgr, err := path.New()
	if err != nil {
		return nil, fmt.Errorf("path manager: %w", err)
	}
	store, err := metadata.NewStore(cfg)
	if err != nil {
		return nil, fmt.Errorf("metadata store: %w", err)
	}
	osSvc, err := fs.NewOSService(cfg)
	if err != nil {
		return nil, fmt.Errorf("OS service: %w", err)
	}
	client, err := github.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("github client: %w", err)
	}

	return &managerImpl{
		pathMgr:  pathMgr,
		client:   client,
		store:    store,
		osSvc:    osSvc,
		archiver: &archive.Archiver{},
	}, nil
}

// NewWithDeps creates a Manager with injected dependencies (for tests)
func NewWithDeps(pathMgr *path.Manager, client github.Client, store metadata.MetadataStore, osSvc fs.OSService, archiver interface{ ExtractFile(src, destDir string) (string, error) }) Manager {
	return &managerImpl{
		pathMgr:  pathMgr,
		client:   client,
		store:    store,
		osSvc:    osSvc,
		archiver: archiver,
	}
}
