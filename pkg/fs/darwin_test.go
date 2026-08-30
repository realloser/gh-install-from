//go:build darwin

package fs

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

// TestRemoveQuarantine is the xattr round-trip: set the quarantine attribute on
// a file, confirm HasQuarantine sees it, strip it, confirm it is gone, and
// confirm removing it again (ENODATA) is a no-op.
func TestRemoveQuarantine(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bin")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := unix.Lsetxattr(p, quarantineAttr, []byte("0000;...;"), 0); err != nil {
		t.Skipf("cannot set quarantine xattr (may need privileges): %v", err)
	}
	if ok, _ := HasQuarantine(p); !ok {
		t.Fatal("expected quarantine attribute present after Lsetxattr")
	}
	if err := RemoveQuarantine(p); err != nil {
		t.Fatal(err)
	}
	if ok, _ := HasQuarantine(p); ok {
		t.Fatal("quarantine attribute still present after RemoveQuarantine")
	}
	// removing again (ENODATA) must be a no-op, not an error
	if err := RemoveQuarantine(p); err != nil {
		t.Errorf("RemoveQuarantine on clean file should be a no-op, got %v", err)
	}
}
