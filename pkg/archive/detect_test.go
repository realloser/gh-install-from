package archive

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		want    Format
		wantErr error
	}{
		{
			name: "gzip file routes to archive",
			setup: func(t *testing.T) string {
				return createTestTarGz(t, "testbin", "#!/bin/sh\necho test")
			},
			want: FormatArchive,
		},
		{
			name: "zip file routes to archive",
			setup: func(t *testing.T) string {
				return createTestZip(t, "testbin.exe", "test content")
			},
			want: FormatArchive,
		},
		{
			name: "plain binary file routes to binary",
			setup: func(t *testing.T) string {
				// ELF-ish content that http.DetectContentType classifies as
				// application/octet-stream, i.e. a bare binary.
				return createTestFile(t, "test content")
			},
			want: FormatBinary,
		},
		{
			name: "empty file is rejected",
			setup: func(t *testing.T) string {
				return createTestFile(t, "")
			},
			want:    FormatUnknown,
			wantErr: ErrUnsupportedFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			defer os.Remove(path)

			got, err := DetectFormat(path)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("DetectFormat() error = %v, want %v", err, tt.wantErr)
				}
				if got != tt.want {
					t.Fatalf("DetectFormat() = %v, want %v", got, tt.want)
				}
				return
			}
			if err != nil {
				t.Fatalf("DetectFormat() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("DetectFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectFormatMissingFile(t *testing.T) {
	_, err := DetectFormat(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
