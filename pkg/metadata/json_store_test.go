package metadata

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJSONStore_StoreLoadDelete(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONStore(tmpDir)

	meta := &BinaryMetadata{
		GHHost:     "github.com",
		Repository: "test/repo",
		Version:    "v1.0.0",
		BinaryPath: "/some/bin/test-binary",
	}

	if err := store.Store(meta); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	loaded, err := store.Load("test-binary")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Repository != meta.Repository || loaded.Version != meta.Version {
		t.Errorf("Load() got %+v, want %+v", loaded, meta)
	}

	if err := store.Delete("test-binary"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if _, err := store.Load("test-binary"); err == nil {
		t.Error("Load() after Delete expected error")
	}
}

func TestJSONStore_List(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONStore(tmpDir)

	store.Store(&BinaryMetadata{Repository: "a/a", Version: "v1", BinaryPath: "/bin/a"})
	store.Store(&BinaryMetadata{Repository: "b/b", Version: "v2", BinaryPath: "/bin/b"})

	list, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 2 {
		t.Errorf("List() got %d items, want 2", len(list))
	}
}

func TestNewStore(t *testing.T) {
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	os.Unsetenv("GH_INSTALL_FROM_HOME")
	defer func() {
		os.Unsetenv("HOME")
	}()

	s, err := NewStore(nil)
	if err != nil {
		t.Fatalf("NewStore(nil) error = %v", err)
	}
	if s == nil {
		t.Fatal("NewStore(nil) returned nil")
	}

	meta := &BinaryMetadata{
		GHHost:     "github.com",
		Repository: "test/repo",
		Version:    "v1",
		BinaryPath: filepath.Join(tmpHome, ".gh-install-from", "bin", "test"),
	}
	if err := s.Store(meta); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	loaded, err := s.Load("test")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Repository != "test/repo" {
		t.Errorf("Load() got repo %q, want test/repo", loaded.Repository)
	}
}
