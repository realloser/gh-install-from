package metadata

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/realloser/gh-install-from/pkg/config"
)

// Ensure JSONStore implements MetadataStore
var _ MetadataStore = (*JSONStore)(nil)

// JSONStore implements Store using JSON files in a directory
type JSONStore struct {
	metadataDir string
	mu          sync.RWMutex
}

func init() {
	RegisterStore("json", NewJSONStoreFromConfig)
}

// NewJSONStore creates a JSONStore with the given metadata directory
func NewJSONStore(metadataDir string) *JSONStore {
	return &JSONStore{metadataDir: metadataDir}
}

// NewJSONStoreFromConfig creates a JSONStore from config (used by registry)
func NewJSONStoreFromConfig(cfg *config.Config) (MetadataStore, error) {
	if cfg == nil {
		cfg = config.FromEnv()
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	root := os.Getenv("GH_INSTALL_FROM_HOME")
	if root == "" {
		root = filepath.Join(homeDir, ".gh-install-from")
	}
	metadataDir := filepath.Join(root, "metadata")
	slog.Debug("json store created", "metadata_dir", metadataDir)
	return NewJSONStore(metadataDir), nil
}

// Store saves metadata for an installed binary
func (s *JSONStore) Store(meta *BinaryMetadata) error {
	if meta == nil {
		return fmt.Errorf("metadata cannot be nil")
	}
	if meta.BinaryPath == "" {
		return fmt.Errorf("binary path cannot be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.metadataDir, 0750); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	binaryName := filepath.Base(meta.BinaryPath)
	metadataPath := filepath.Join(s.metadataDir, binaryName+".json")

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	slog.Debug("metadata stored", "binary", binaryName, "path", metadataPath)
	return nil
}

// Load retrieves metadata for a binary by name
func (s *JSONStore) Load(binaryName string) (*BinaryMetadata, error) {
	if binaryName == "" {
		return nil, fmt.Errorf("binary name cannot be empty")
	}
	binaryName = strings.TrimSpace(filepath.Base(binaryName))
	s.mu.RLock()
	defer s.mu.RUnlock()

	metadataPath := filepath.Join(s.metadataDir, binaryName+".json")

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no metadata found for binary %s", binaryName)
		}
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	var meta BinaryMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &meta, nil
}

// Delete removes metadata for a binary
func (s *JSONStore) Delete(binaryName string) error {
	if binaryName == "" {
		return fmt.Errorf("binary name cannot be empty")
	}
	binaryName = strings.TrimSpace(filepath.Base(binaryName))
	s.mu.Lock()
	defer s.mu.Unlock()

	metadataPath := filepath.Join(s.metadataDir, binaryName+".json")

	if err := os.Remove(metadataPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to delete metadata file: %w", err)
	}

	slog.Debug("metadata deleted", "binary", binaryName)
	return nil
}

// List returns all stored metadata
func (s *JSONStore) List() ([]*BinaryMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.metadataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read metadata directory: %w", err)
	}

	var result []*BinaryMetadata
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		metadataPath := filepath.Join(s.metadataDir, entry.Name())
		data, err := os.ReadFile(metadataPath)
		if err != nil {
			slog.Debug("skipping metadata file", "file", entry.Name(), "error", err)
			continue
		}
		var meta BinaryMetadata
		if err := json.Unmarshal(data, &meta); err != nil {
			slog.Debug("skipping metadata file", "file", entry.Name(), "error", err)
			continue
		}
		result = append(result, &meta)
	}

	return result, nil
}
