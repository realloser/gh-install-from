//go:build !windows

package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/realloser/gh-install-from/pkg/config"
)

func TestBashAdapter_AddToPATH(t *testing.T) {
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	defer os.Unsetenv("HOME")

	adapter := NewBashAdapter()
	binPath := filepath.Join(tmpHome, ".gh-install-from", "bin")

	if err := adapter.AddToPATH(binPath); err != nil {
		t.Fatalf("AddToPATH: %v", err)
	}

	rcPath := filepath.Join(tmpHome, ".bashrc")
	content, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("read .bashrc: %v", err)
	}
	if !strings.Contains(string(content), binPath) {
		t.Errorf(".bashrc does not contain %q", binPath)
	}

	// Idempotent: second call should not duplicate
	if err := adapter.AddToPATH(binPath); err != nil {
		t.Fatalf("AddToPATH second call: %v", err)
	}
	content2, _ := os.ReadFile(rcPath)
	if strings.Count(string(content2), binPath) != 1 {
		t.Error("AddToPATH duplicated path (not idempotent)")
	}
}

func TestNewShellAdapter_ByConfig(t *testing.T) {
	cfg := &config.Config{Shell: "bash"}
	adapter, err := NewShellAdapter(cfg)
	if err != nil {
		t.Fatalf("NewShellAdapter(bash): %v", err)
	}
	if adapter == nil {
		t.Fatal("NewShellAdapter returned nil")
	}
	if _, ok := adapter.(*BashAdapter); !ok {
		t.Errorf("NewShellAdapter(bash) got %T, want *BashAdapter", adapter)
	}
}

func TestNewShellAdapter_UnknownShell(t *testing.T) {
	cfg := &config.Config{Shell: "unknown-shell-xyz"}
	_, err := NewShellAdapter(cfg)
	if err == nil {
		t.Error("NewShellAdapter(unknown) expected error")
	}
}
