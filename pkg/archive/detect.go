package archive

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

// Format classifies the coarse content type of a downloaded release asset.
// The archive sub-format (gzip vs zip) is decided separately inside
// ArchiveProcessor from the sniffed magic bytes.
type Format int

const (
	// FormatUnknown means the file could not be classified (e.g. empty).
	FormatUnknown Format = iota
	// FormatArchive covers both gzip (tar.gz/tgz) and zip archives.
	FormatArchive
	// FormatBinary covers bare executables (ELF/Mach-O/etc).
	FormatBinary
)

// Sentinel errors exposed via errors.Is so the manager can route and fall through.
var (
	// ErrUnsupportedFormat indicates a file that is neither a recognized archive
	// nor a usable bare binary.
	ErrUnsupportedFormat = errors.New("unsupported file format")
	// ErrNoExecutable indicates an archive that contained no executable member.
	ErrNoExecutable = errors.New("no executable binary found in archive")
)

// DetectFormat sniffs the content of the file at path and classifies it as an
// archive or a bare binary. An empty (0-byte) file is rejected as
// ErrUnsupportedFormat — a zero-byte download is exactly the corruption we must
// never install.
func DetectFormat(path string) (Format, error) {
	f, err := os.Open(path)
	if err != nil {
		return FormatUnknown, fmt.Errorf("failed to open file for detection: %w", err)
	}
	defer f.Close()

	// http.DetectContentType inspects the first 512 bytes. A short read that ends
	// in io.EOF is fine (it just means the file is smaller than the window); an
	// empty read is handed to detectFromBytes, which rejects it as
	// ErrUnsupportedFormat. Any other read error is propagated.
	header := make([]byte, 512)
	n, err := f.Read(header)
	if err != nil && err != io.EOF {
		return FormatUnknown, fmt.Errorf("failed to read file header: %w", err)
	}
	header = header[:n]

	return detectFromBytes(header)
}

// detectFromBytes classifies a byte window (already read from the file) using the
// same magic sniffing DetectFormat performs, so ArchiveProcessor can reuse it to
// pick gzip vs zip without duplicating I/O.
func detectFromBytes(header []byte) (Format, error) {
	if len(header) == 0 {
		return FormatUnknown, fmt.Errorf("empty file: %w", ErrUnsupportedFormat)
	}

	// Short-circuit on the definitive magic prefixes first.
	if isGzipMagic(header) {
		return FormatArchive, nil
	}
	if isZipMagic(header) {
		return FormatArchive, nil
	}

	// Fall back to http.DetectContentType for anything else. Bare binaries
	// (ELF/Mach-O) come back as application/octet-stream.
	ct := http.DetectContentType(header)
	switch ct {
	case "application/x-gzip", "application/zip":
		return FormatArchive, nil
	default:
		return FormatBinary, nil
	}
}

// isGzipMagic reports whether the byte window starts with the gzip magic 0x1f 0x8b.
func isGzipMagic(b []byte) bool {
	return len(b) >= 2 && b[0] == 0x1f && b[1] == 0x8b
}

// isZipMagic reports whether the byte window starts with the zip magic PK\x03\x04.
func isZipMagic(b []byte) bool {
	return len(b) >= 4 && b[0] == 'P' && b[1] == 'K' && b[2] == 0x03 && b[3] == 0x04
}
