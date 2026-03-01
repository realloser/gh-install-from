// Package fs provides OSService for file operations (symlinks, shims)
package fs

// OSService defines the interface for binary installation file operations
type OSService interface {
	// InstallBinary creates a symlink (Unix) or .cmd shim (Windows) in binDir pointing to targetPath
	InstallBinary(binDir, binaryName, targetPath string) error
	// RemoveBinary removes the symlink or shim from binDir
	RemoveBinary(binDir, binaryName string) error
}
