package archive

import (
	"fmt"
	"os"
	"path/filepath"
)

// Processor turns a downloaded release asset (at src) into a single binary file
// located under destDir, returning the resulting path. The two implementations —
// ArchiveProcessor and BinaryProcessor — let pkg/binary route by detected content
// rather than by filename extension.
type Processor interface {
	Process(src, destDir string) (string, error)
}

// ArchiveProcessor extracts an executable member from a gzip/tar or zip archive.
type ArchiveProcessor struct{}

// Process determines the archive sub-format from the sniffed magic bytes and
// delegates to the existing extraction helpers. When no executable member is
// found, it returns ErrNoExecutable so callers can fall through to binary handling.
func (a *ArchiveProcessor) Process(src, destDir string) (string, error) {
	f, err := os.Open(src)
	if err != nil {
		return "", fmt.Errorf("failed to open archive: %w", err)
	}
	header := make([]byte, 512)
	n, _ := f.Read(header)
	f.Close()
	header = header[:n]

	switch {
	case isGzipMagic(header):
		return extractGzipFile(src, destDir)
	case isZipMagic(header):
		return extractZipFile(src, destDir)
	default:
		return "", ErrUnsupportedFormat
	}
}

// BinaryProcessor copies a bare binary to a distinct path under destDir.
type BinaryProcessor struct{}

// Process copies src to filepath.Join(destDir, filepath.Base(src)) and returns
// the destination. It never writes onto src itself: if the computed destination
// is the same path as src (the historical self-truncate case), it refuses rather
// than truncating the shared inode to zero bytes.
func (b *BinaryProcessor) Process(src, destDir string) (string, error) {
	dest := filepath.Join(destDir, filepath.Base(src))

	// Distinct-path guard: dest == src would open the same inode with O_TRUNC and
	// zero it before reading. Refuse instead. In normal operation the manager
	// downloads into a .src subdir, so this is defense-in-depth.
	if samePath(src, dest) {
		return "", fmt.Errorf("%w: destination path collides with source %s", ErrUnsupportedFormat, src)
	}

	if err := copyFile(src, dest); err != nil {
		return "", err
	}
	return dest, nil
}

// samePath reports whether two paths resolve to the same file.
func samePath(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA == nil && errB == nil {
		return absA == absB
	}
	return a == b
}
