package path

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManager_GetRoot(t *testing.T) {
	oldHome := os.Getenv("GH_INSTALL_FROM_HOME")
	defer func() { os.Setenv("GH_INSTALL_FROM_HOME", oldHome) }()

	// Clear override to use default
	os.Unsetenv("GH_INSTALL_FROM_HOME")
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)

	m, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	want := filepath.Join(tmpHome, ".gh-install-from")
	if got := m.GetRoot(); got != want {
		t.Errorf("GetRoot() = %v, want %v", got, want)
	}
}

func TestManager_GetRoot_EnvOverride(t *testing.T) {
	oldHome := os.Getenv("GH_INSTALL_FROM_HOME")
	defer func() { os.Setenv("GH_INSTALL_FROM_HOME", oldHome) }()

	customRoot := t.TempDir()
	os.Setenv("GH_INSTALL_FROM_HOME", customRoot)

	m, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if got := m.GetRoot(); got != customRoot {
		t.Errorf("GetRoot() = %v, want %v", got, customRoot)
	}
}

func TestManager_GetBinDir(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	got := m.GetBinDir()
	want := filepath.Join(m.GetRoot(), dirBin)
	if got != want {
		t.Errorf("GetBinDir() = %v, want %v", got, want)
	}
}

func TestManager_GetDownloadsDir(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	got := m.GetDownloadsDir()
	want := filepath.Join(m.GetRoot(), dirDownloads)
	if got != want {
		t.Errorf("GetDownloadsDir() = %v, want %v", got, want)
	}
}

func TestManager_GetMetadataDir(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	got := m.GetMetadataDir()
	want := filepath.Join(m.GetRoot(), dirMetadata)
	if got != want {
		t.Errorf("GetMetadataDir() = %v, want %v", got, want)
	}
}
