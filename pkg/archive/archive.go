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

// ExtractFile extracts a file from a compressed archive or copies a regular file
func ExtractFile(src, dest string) error {
	switch {
	case isGzipFile(src):
		return extractGzipFile(src, dest)
	case isZipFile(src):
		return extractZipFile(src, dest)
	default:
		return copyFile(src, dest)
	}
}

func extractGzipFile(src, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open gzip file: %w", err)
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
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() {
		if cerr := gzr.Close(); cerr != nil {
			if err == nil {
				err = fmt.Errorf("failed to close gzip reader: %w", cerr)
			}
		}
	}()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("binary not found in archive")
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		if header.Typeflag == tar.TypeReg {
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

			if _, err := io.Copy(out, tr); err != nil {
				return fmt.Errorf("failed to extract file: %w", err)
			}
			return nil
		}
	}
}

func extractZipFile(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer func() {
		if cerr := r.Close(); cerr != nil {
			if err == nil {
				err = fmt.Errorf("failed to close zip reader: %w", cerr)
			}
		}
	}()

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

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
		if err != nil {
			rc.Close()
			out.Close()
			return fmt.Errorf("failed to extract file: %w", err)
		}

		if err := rc.Close(); err != nil {
			out.Close()
			return fmt.Errorf("failed to close zip file: %w", err)
		}
		if err := out.Close(); err != nil {
			return fmt.Errorf("failed to close output file: %w", err)
		}
		return nil
	}

	return fmt.Errorf("no files found in zip archive")
}

func copyFile(src, dest string) error {
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
