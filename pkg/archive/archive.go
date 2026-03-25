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

// Archiver handles archive operations
type Archiver struct{}

// ExtractFile extracts a file from a compressed archive or copies a regular file
// Returns the path of the extracted binary file
func (a *Archiver) ExtractFile(src, dest string) (string, error) {
	switch {
	case isGzipFile(src):
		return extractGzipFile(src, dest)
	case isZipFile(src):
		return extractZipFile(src, dest)
	default:
		// For plain binary files, create the destination file path
		destFile := filepath.Join(dest, filepath.Base(src))
		if err := copyFile(src, destFile); err != nil {
			return "", err
		}
		return destFile, nil
	}
}

func extractGzipFile(src, dest string) (string, error) {
	file, err := os.Open(src)
	if err != nil {
		return "", fmt.Errorf("failed to open gzip file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			if err == nil {
				err = fmt.Errorf("failed to close file: %w", cerr)
			}
		}
	}()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() {
		if cerr := gzr.Close(); cerr != nil {
			if err == nil {
				err = fmt.Errorf("failed to close gzip reader: %w", cerr)
			}
		}
	}()

	tr := tar.NewReader(gzr)
	var extractedBinary string

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("failed to read tar header: %w", err)
		}

		// Skip if the entry is empty or just a directory
		if header.Name == "" || header.Typeflag == tar.TypeDir {
			continue
		}

		// Create the file
		outPath := filepath.Join(dest, filepath.Base(header.Name))
		out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
		if err != nil {
			return "", fmt.Errorf("failed to create file %s: %w", outPath, err)
		}

		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return "", fmt.Errorf("failed to extract file %s: %w", outPath, err)
		}
		out.Close()

		// If this file is executable, it's likely our binary
		if header.Mode&0111 != 0 {
			extractedBinary = outPath
		}
	}

	if extractedBinary == "" {
		return "", fmt.Errorf("no executable binary found in archive")
	}

	return extractedBinary, nil
}

func extractZipFile(src, dest string) (string, error) {
	r, err := zip.OpenReader(src)
	if err != nil {
		return "", fmt.Errorf("failed to open zip file: %w", err)
	}
	defer func() {
		if cerr := r.Close(); cerr != nil {
			if err == nil {
				err = fmt.Errorf("failed to close zip reader: %w", cerr)
			}
		}
	}()

	var extractedBinary string

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return "", fmt.Errorf("failed to open file in zip: %w", err)
		}

		outPath := filepath.Join(dest, filepath.Base(f.Name))
		out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return "", fmt.Errorf("failed to create file %s: %w", outPath, err)
		}

		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return "", fmt.Errorf("failed to extract file %s: %w", outPath, err)
		}

		// If this file is executable, it's likely our binary
		if f.Mode()&0111 != 0 {
			extractedBinary = outPath
		}
	}

	if extractedBinary == "" {
		return "", fmt.Errorf("no executable binary found in archive")
	}

	return extractedBinary, nil
}

func copyFile(src, dest string) error {
	// Check if dest is a directory and create a full path
	if info, err := os.Stat(dest); err == nil && info.IsDir() {
		dest = filepath.Join(dest, filepath.Base(src))
	}
	
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer func() {
		if cerr := in.Close(); cerr != nil {
			if err == nil {
				err = fmt.Errorf("failed to close input file: %w", cerr)
			}
		}
	}()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() {
		if cerr := out.Close(); cerr != nil {
			if err == nil {
				err = fmt.Errorf("failed to close output file: %w", cerr)
			}
		}
	}()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

func isGzipFile(filename string) bool {
	return filepath.Ext(filename) == ".gz" || filepath.Ext(filename) == ".tgz"
}

func isZipFile(filename string) bool {
	return filepath.Ext(filename) == ".zip"
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
