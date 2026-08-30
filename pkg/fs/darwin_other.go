//go:build !darwin

package fs

// Non-darwin platforms have no macOS quarantine attribute. All functions are
// no-ops so the call site compiles everywhere and behavior is unchanged.

func HasQuarantine(path string) (bool, error) { return false, nil }
func RemoveQuarantine(path string) error      { return nil }
