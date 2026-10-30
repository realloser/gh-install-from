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

func TestUnixFSService_InstallBinary_ReplacesExistingSymlink(t *testing.T) {
	svc := &UnixFSService{}
	binDir := t.TempDir()
	oldTarget := filepath.Join(t.TempDir(), "v1")
	newTarget := filepath.Join(t.TempDir(), "v2")
	for _, p := range []string{oldTarget, newTarget} {
		if err := os.WriteFile(p, []byte("x"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := svc.InstallBinary(binDir, "tool", oldTarget); err != nil {
		t.Fatalf("first install: %v", err)
	}
	if err := svc.InstallBinary(binDir, "tool", newTarget); err != nil {
		t.Fatalf("update over existing symlink: %v", err)
	}
	got, err := os.Readlink(filepath.Join(binDir, "tool"))
	if err != nil || got != newTarget {
		t.Errorf("Readlink = %q, %v; want %q", got, err, newTarget)
	}
}

func TestUnixFSService_InstallBinary_DoesNotClobberRegularFile(t *testing.T) {
	svc := &UnixFSService{}
	binDir := t.TempDir()
	existing := filepath.Join(binDir, "tool")
	if err := os.WriteFile(existing, []byte("user file"), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "v1")
	if err := os.WriteFile(target, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := svc.InstallBinary(binDir, "tool", target); err == nil {
		t.Error("expected an error instead of overwriting a regular file")
	}
	if b, _ := os.ReadFile(existing); string(b) != "user file" {
		t.Errorf("regular file was modified: %q", b)
	}
}
