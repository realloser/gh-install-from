package archive

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractFile(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (string, string)
		wantErr  bool
		cleanup  func(t *testing.T, src string)
		validate func(t *testing.T, path string)
	}{
		{
			name: "tar.gz file with binary",
			setup: func(t *testing.T) (string, string) {
				src := createTestTarGz(t, "testbin", "#!/bin/sh\necho test")
				dst := filepath.Join(t.TempDir(), "extracted-bin")
				return src, dst
			},
			wantErr: false,
			cleanup: func(t *testing.T, src string) {
				os.Remove(src)
			},
			validate: func(t *testing.T, path string) {
				// Verify file exists
				if _, err := os.Stat(path); err != nil {
					t.Errorf("ExtractFile() destination file not created: %v", err)
				}

				// Verify permissions
				info, err := os.Stat(path)
				if err != nil {
					t.Errorf("Failed to stat destination file: %v", err)
				} else if info.Mode().Perm() != 0755 {
					t.Errorf("Incorrect file permissions: got %v, want %v", info.Mode().Perm(), 0755)
				}
			},
		},
		{
			name: "zip file with binary",
			setup: func(t *testing.T) (string, string) {
				src := createTestZip(t, "testbin.exe", "test content")
				dst := filepath.Join(t.TempDir(), "extracted-bin.exe")
				return src, dst
			},
			wantErr: false,
			cleanup: func(t *testing.T, src string) {
				os.Remove(src)
			},
			validate: func(t *testing.T, path string) {
				// Verify file exists
				if _, err := os.Stat(path); err != nil {
					t.Errorf("ExtractFile() destination file not created: %v", err)
				}

				// Verify permissions
				info, err := os.Stat(path)
				if err != nil {
					t.Errorf("Failed to stat destination file: %v", err)
				} else if info.Mode().Perm() != 0755 {
					t.Errorf("Incorrect file permissions: got %v, want %v", info.Mode().Perm(), 0755)
				}
			},
		},
		{
			name: "simple binary file",
			setup: func(t *testing.T) (string, string) {
				src := createTestFile(t, "test content")
				dst := filepath.Join(t.TempDir(), "copied-bin")
				return src, dst
			},
			wantErr: false,
			cleanup: func(t *testing.T, src string) {
				os.Remove(src)
			},
			validate: func(t *testing.T, path string) {
				// Verify file exists
				if _, err := os.Stat(path); err != nil {
					t.Errorf("ExtractFile() destination file not created: %v", err)
				}

				// Verify permissions
				info, err := os.Stat(path)
				if err != nil {
					t.Errorf("Failed to stat destination file: %v", err)
				} else if info.Mode().Perm() != 0755 {
					t.Errorf("Incorrect file permissions: got %v, want %v", info.Mode().Perm(), 0755)
				}
			},
		},
		{
			name: "invalid_tar.gz_file",
			setup: func(t *testing.T) (string, string) {
				src := filepath.Join(t.TempDir(), "invalid.tar.gz")
				dst := filepath.Join(t.TempDir(), "output")
				err := os.WriteFile(src, []byte("invalid tar.gz content"), 0644)
				if err != nil {
					t.Fatal(err)
				}
				return src, dst
			},
			wantErr: true,
			cleanup: func(t *testing.T, src string) {
				os.Remove(src)
			},
			validate: func(t *testing.T, path string) {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Error("expected output file to not exist")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create source file
			srcPath, destPath := tt.setup(t)
			defer tt.cleanup(t, srcPath)

			// Test extraction
			if err := ExtractFile(srcPath, destPath); (err != nil) != tt.wantErr {
				t.Errorf("ExtractFile() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				tt.validate(t, destPath)
			}
		})
	}
}

func TestIsBinaryFile(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"binary in bin directory", "bin/myapp", true},
		{"windows executable", "myapp.exe", true},
		{"no extension binary", "myapp", true},
		{"source file", "main.go", false},
		{"text file", "readme.txt", false},
		{"nested binary", "tools/bin/app", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBinaryFile(tt.path); got != tt.want {
				t.Errorf("isBinaryFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to create a test tar.gz file
func createTestTarGz(t *testing.T, name, content string) string {
	t.Helper()

	// Create temporary file
	tmpfile, err := os.CreateTemp("", "test-*.tar.gz")
	if err != nil {
		t.Fatal(err)
	}

	// Create gzip writer
	gw := gzip.NewWriter(tmpfile)
	tw := tar.NewWriter(gw)

	// Write file header
	hdr := &tar.Header{
		Name: name,
		Mode: 0755,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}

	// Write file content
	if _, err := tw.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}

	// Close writers
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	return tmpfile.Name()
}

// Helper function to create a test zip file
func createTestZip(t *testing.T, name, content string) string {
	t.Helper()

	// Create temporary file
	tmpfile, err := os.CreateTemp("", "test-*.zip")
	if err != nil {
		t.Fatal(err)
	}
	defer tmpfile.Close()

	// Create zip writer
	zw := zip.NewWriter(tmpfile)

	// Create file in zip
	w, err := zw.Create(name)
	if err != nil {
		t.Fatal(err)
	}

	// Write content
	if _, err := io.Copy(w, bytes.NewBufferString(content)); err != nil {
		t.Fatal(err)
	}

	// Close zip writer
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	return tmpfile.Name()
}

// Helper function to create a test file
func createTestFile(t *testing.T, content string) string {
	t.Helper()

	tmpfile, err := os.CreateTemp("", "test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer tmpfile.Close()

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}

	return tmpfile.Name()
}
