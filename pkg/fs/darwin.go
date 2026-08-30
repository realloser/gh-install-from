//go:build darwin

package fs

import (
	"errors"

	"golang.org/x/sys/unix"
)

// Darwin-specific quarantine support. On macOS, files downloaded via browsers
// or curl carry a com.apple.quarantine extended attribute. If present, Gatekeeper
// refuses to execute the binary ("cannot be opened because the developer cannot
// be verified"). This attribute is never stripped without explicit user opt-in.

const quarantineAttr = "com.apple.quarantine"

// HasQuarantine reports whether path carries the com.apple.quarantine xattr.
// It does NOT follow symlinks (lgetxattr on the path itself), so callers can
// inspect a symlink without reading through to its target.
func HasQuarantine(path string) (bool, error) {
	_, err := unix.Lgetxattr(path, quarantineAttr, nil)
	if isAttrAbsent(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// RemoveQuarantine strips the com.apple.quarantine xattr from path.
// It does NOT follow symlinks (Lremovexattr), so it strips the attribute on the
// path as given — the caller decides whether that is a symlink or its target.
// Removing a non-existent attribute is a no-op (ENODATA is swallowed).
func RemoveQuarantine(path string) error {
	err := unix.Lremovexattr(path, quarantineAttr)
	if isAttrAbsent(err) {
		return nil
	}
	return err
}

// isAttrAbsent reports whether err means the attribute does not exist. The
// "attribute not found" errno is ENODATA on Linux but ENOATTR on macOS; treat
// both as "absent" so a missing attribute is never surfaced as an error.
func isAttrAbsent(err error) bool {
	return errors.Is(err, unix.ENODATA) || errors.Is(err, unix.ENOATTR)
}
