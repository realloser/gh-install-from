//go:build !darwin

package fs

import "testing"

// TestNonDarwinQuarantineNoops asserts that on non-darwin platforms the
// quarantine helpers are no-ops: HasQuarantine always reports false and
// RemoveQuarantine never errors. This guards the "non-darwin completely
// unaffected" requirement.
func TestNonDarwinQuarantineNoops(t *testing.T) {
	ok, err := HasQuarantine("/definitely/not/a/real/path")
	if err != nil {
		t.Fatalf("HasQuarantine returned error on non-darwin: %v", err)
	}
	if ok {
		t.Fatal("HasQuarantine returned true on non-darwin")
	}
	if err := RemoveQuarantine("/definitely/not/a/real/path"); err != nil {
		t.Fatalf("RemoveQuarantine returned error on non-darwin: %v", err)
	}
}
