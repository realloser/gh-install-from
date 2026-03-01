//go:build !windows

package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/realloser/gh-install-from/pkg/config"
)

func TestUnixFSService_InstallRemoveBinary(t *testing.T) {
	binDir := t.TempDir()
	targetDir := t.TempDir()
	targetPath := filepath.Join(targetDir, "my-binary")
	if err := os.WriteFile(targetPath, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("create target: %v", err)
	}

	svc := NewUnixFSService()
	if err := svc.InstallBinary(binDir, "my-binary", targetPath); err != nil {
		t.Fatalf("InstallBinary: %v", err)
	}

	linkPath := filepath.Join(binDir, "my-binary")
	if _, err := os.Lstat(linkPath); err != nil {
		t.Fatalf("symlink not created: %v", err)
	}

	if err := svc.RemoveBinary(binDir, "my-binary"); err != nil {
		t.Fatalf("RemoveBinary: %v", err)
	}

	if _, err := os.Lstat(linkPath); !os.IsNotExist(err) {
		t.Errorf("symlink still exists after RemoveBinary")
	}
}

func TestNewOSService_Unix(t *testing.T) {
	cfg := &config.Config{OS: "darwin"}
	svc, err := NewOSService(cfg)
	if err != nil {
		t.Fatalf("NewOSService(darwin): %v", err)
	}
	if svc == nil {
		t.Fatal("NewOSService returned nil")
	}
	if _, ok := svc.(*UnixFSService); !ok {
		t.Errorf("NewOSService(darwin) got %T, want *UnixFSService", svc)
	}
}
