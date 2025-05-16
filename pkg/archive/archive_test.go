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
		setup    func(t *testing.T) string
		destName string
		wantErr  bool
	}{
		{
			name: "tar.gz file with binary",
			setup: func(t *testing.T) string {
				return createTestTarGz(t, "testbin", "#!/bin/sh\necho test")
			},
			destName: "extracted-bin",
			wantErr:  false,
		},
		{
			name: "zip file with binary",
			setup: func(t *testing.T) string {
				return createTestZip(t, "testbin.exe", "test content")
			},
			destName: "extracted-bin.exe",
			wantErr:  false,
		},
		{
			name: "simple binary file",
			setup: func(t *testing.T) string {
				return createTestFile(t, "test content")
			},
			destName: "copied-bin",
			wantErr:  false,
		},
		{
			name: "invalid tar.gz file",
			setup: func(t *testing.T) string {
				return createTestFile(t, "invalid tar.gz content")
			},
			destName: "should-fail.tar.gz",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create source file
			srcPath := tt.setup(t)
			defer os.Remove(srcPath)

			// Create destination path
			destPath := filepath.Join(t.TempDir(), tt.destName)

			// Test extraction
			err := ExtractFile(srcPath, destPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify file exists
				if _, err := os.Stat(destPath); err != nil {
					t.Errorf("ExtractFile() destination file not created: %v", err)
				}

				// Verify permissions
				info, err := os.Stat(destPath)
				if err != nil {
					t.Errorf("Failed to stat destination file: %v", err)
				} else if info.Mode().Perm() != 0755 {
					t.Errorf("Incorrect file permissions: got %v, want %v", info.Mode().Perm(), 0755)
				}
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