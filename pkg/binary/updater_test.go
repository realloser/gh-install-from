package binary

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/realloser/gh-install-from/pkg/config"
)

func setupUpdaterTest(t *testing.T) {
	t.Helper()
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	os.Setenv("GH_INSTALL_FROM_HOME", filepath.Join(tmpHome, ".gh-install-from"))
	t.Cleanup(func() {
		os.Unsetenv("HOME")
		os.Unsetenv("GH_INSTALL_FROM_HOME")
	})
}

func TestCheckUpdates_Empty(t *testing.T) {
	setupUpdaterTest(t)
	manager, err := NewManager(&config.Config{Client: "mock"})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	candidates, err := manager.CheckUpdates(nil)
	if err != nil {
		t.Fatalf("CheckUpdates: %v", err)
	}
	if len(candidates) != 0 {
		t.Errorf("CheckUpdates(nil) got %d candidates, want 0", len(candidates))
	}
}

func TestCheckUpdates_WithMock(t *testing.T) {
	setupUpdaterTest(t)
	manager, err := NewManager(&config.Config{Client: "mock"})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	installed := []InstalledBinary{
		{Repository: "test/repo", Version: "v1.0.0", Name: "repo"},
	}

	// Mock client returns nil for GetLatestRelease, so no updates found
	candidates, err := manager.CheckUpdates(installed)
	if err != nil {
		t.Fatalf("CheckUpdates: %v", err)
	}
	if len(candidates) != 0 {
		t.Errorf("CheckUpdates with mock got %d candidates, want 0", len(candidates))
	}
}
