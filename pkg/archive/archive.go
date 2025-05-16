package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractFile extracts a binary from a compressed file
func ExtractFile(src, dest string) error {
	switch {
	case strings.HasSuffix(src, ".tar.gz") || strings.HasSuffix(src, ".tgz"):
		return extractTarGz(src, dest)
	case strings.HasSuffix(src, ".zip"):
		return extractZip(src, dest)
	default:
		// If not compressed, just copy the file
		return copyFile(src, dest)
	}
}

func extractTarGz(src, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	var foundBinary bool

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Skip directories and non-regular files
		if header.Typeflag != tar.TypeReg {
			continue
		}

		// Look for binary file (usually in bin/ directory or root)
		if isBinaryFile(header.Name) {
			out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return fmt.Errorf("failed to create destination file: %w", err)
			}
			defer out.Close()

			if _, err := io.Copy(out, tr); err != nil {
				return fmt.Errorf("failed to extract file: %w", err)
			}
			foundBinary = true
			break
		}
	}

	if !foundBinary {
		return fmt.Errorf("no binary found in archive")
	}
	return nil
}

func extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer r.Close()

	var foundBinary bool
	for _, f := range r.File {
		// Skip directories and non-regular files
		if f.FileInfo().IsDir() {
			continue
		}

		// Look for binary file
		if isBinaryFile(f.Name) {
			rc, err := f.Open()
			if err != nil {
				return fmt.Errorf("failed to open file in zip: %w", err)
			}

			out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				rc.Close()
				return fmt.Errorf("failed to create destination file: %w", err)
			}

			_, err = io.Copy(out, rc)
			rc.Close()
			out.Close()
			if err != nil {
				return fmt.Errorf("failed to extract file: %w", err)
			}

			foundBinary = true
			break
		}
	}

	if !foundBinary {
		return fmt.Errorf("no binary found in archive")
	}
	return nil
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// isBinaryFile checks if the file path looks like a binary
func isBinaryFile(path string) bool {
	name := filepath.Base(path)
	ext := filepath.Ext(name)

	// Check common binary locations
	if strings.Contains(path, "/bin/") {
		return true
	}

	// Check if it's an executable
	if ext == ".exe" {
		return true
	}

	// Check if it has no extension (common for Unix binaries)
	if ext == "" {
		return true
	}

	return false
} 