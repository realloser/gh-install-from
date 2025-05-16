package github

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNewGhCliClient(t *testing.T) {
	// Save original PATH and create a temporary directory for test binaries
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	tmpDir := t.TempDir()
	os.Setenv("PATH", tmpDir)

	// Test when gh is not installed
	t.Run("gh not installed", func(t *testing.T) {
		client, err := newGhCliClient()
		if err == nil {
			t.Error("expected error when gh is not installed")
		}
		if client != nil {
			t.Error("expected nil client when gh is not installed")
		}
	})

	// Create a mock gh executable
	mockGh := filepath.Join(tmpDir, "gh")
	if runtime.GOOS == "windows" {
		mockGh += ".exe"
	}

	// Test with default host (github.com)
	t.Run("default host", func(t *testing.T) {
		// Create mock gh that returns error for auth status
		err := os.WriteFile(mockGh, []byte(`#!/bin/sh
exit 1
`), 0755)
		if err != nil {
			t.Fatal(err)
		}

		client, err := newGhCliClient()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if host := client.GetHost(); host != "github.com" {
			t.Errorf("expected default host github.com, got %s", host)
		}
	})

	// Test with custom host from auth status
	t.Run("custom host", func(t *testing.T) {
		// Create mock gh that returns a custom host
		err := os.WriteFile(mockGh, []byte(`#!/bin/sh
echo "Logged in to github.enterprise.com as testuser"
`), 0755)
		if err != nil {
			t.Fatal(err)
		}

		client, err := newGhCliClient()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if host := client.GetHost(); host != "github.enterprise.com" {
			t.Errorf("expected host github.enterprise.com, got %s", host)
		}
	})

	// Test with malformed auth status output
	t.Run("malformed auth status", func(t *testing.T) {
		// Create mock gh that returns malformed output
		err := os.WriteFile(mockGh, []byte(`#!/bin/sh
echo "Invalid output format"
`), 0755)
		if err != nil {
			t.Fatal(err)
		}

		client, err := newGhCliClient()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if host := client.GetHost(); host != "github.com" {
			t.Errorf("expected default host github.com, got %s", host)
		}
	})
}

func TestGhCliClient_GetLatestRelease(t *testing.T) {
	// Create a temporary directory for our mock gh command
	tmpDir, err := os.MkdirTemp("", "gh-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a mock gh command
	mockGh := filepath.Join(tmpDir, "gh")
	err = os.WriteFile(mockGh, []byte(`#!/bin/sh
if [ "$3" = "repos/valid/repo/releases/latest" ]; then
	echo '{"assets":[{"name":"test.tar.gz","browser_download_url":"https://example.com/test.tar.gz"}]}'
	exit 0
else
	echo "Invalid repository" >&2
	exit 1
fi
`), 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Add our mock gh to the PATH
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", fmt.Sprintf("%s:%s", tmpDir, origPath))
	defer os.Setenv("PATH", origPath)

	tests := []struct {
		name    string
		repo    string
		want    *Release
		wantErr bool
	}{
		{
			name: "valid repo",
			repo: "valid/repo",
			want: &Release{
				Assets: []Asset{
					{
						Name:               "test.tar.gz",
						BrowserDownloadURL: "https://example.com/test.tar.gz",
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid repo",
			repo:    "invalid/repo",
			want:    nil,
			wantErr: true,
		},
	}

	client := &ghCliClient{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := client.GetLatestRelease(tt.repo)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetLatestRelease() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("GetLatestRelease() returned nil release")
			}
		})
	}
}

func TestGhCliClient_DownloadAsset(t *testing.T) {
	// Create a temporary directory for our mock gh command
	tmpDir, err := os.MkdirTemp("", "gh-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a mock gh command
	mockGh := filepath.Join(tmpDir, "gh")
	err = os.WriteFile(mockGh, []byte(`#!/bin/sh
if [ "$4" = "https://example.com/test.tar.gz" ]; then
	echo "test data" > "$1"
	exit 0
else
	echo "Invalid URL" >&2
	exit 1
fi
`), 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Add our mock gh to the PATH
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", fmt.Sprintf("%s:%s", tmpDir, origPath))
	defer os.Setenv("PATH", origPath)

	tests := []struct {
		name     string
		url      string
		destPath string
		wantErr  bool
	}{
		{
			name:     "invalid url",
			url:      "https://example.com/invalid.tar.gz",
			destPath: filepath.Join(tmpDir, "test1"),
			wantErr:  true,
		},
		{
			name:     "create file error",
			url:      "https://example.com/test.tar.gz",
			destPath: "/nonexistent/test",
			wantErr:  true,
		},
	}

	client := &ghCliClient{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.DownloadAsset(tt.url, tt.destPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("DownloadAsset() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetLatestRelease(t *testing.T) {
	client := &ghCliClient{host: "github.com"}

	tests := []struct {
		name    string
		repo    string
		wantErr bool
	}{
		{
			name:    "invalid empty repo",
			repo:    "",
			wantErr: true,
		},
		{
			name:    "invalid repo format",
			repo:    "invalid/repo/format",
			wantErr: true,
		},
		{
			name:    "valid repo format",
			repo:    "owner/repo",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetLatestRelease(tt.repo)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetLatestRelease() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsValidRepo(t *testing.T) {
	tests := []struct {
		name string
		repo string
		want bool
	}{
		{
			name: "empty repo",
			repo: "",
			want: false,
		},
		{
			name: "starts with slash",
			repo: "/owner/repo",
			want: false,
		},
		{
			name: "ends with slash",
			repo: "owner/repo/",
			want: false,
		},
		{
			name: "valid format",
			repo: "owner/repo",
			want: true,
		},
		{
			name: "too long",
			repo: string(make([]byte, 257)),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidRepo(tt.repo); got != tt.want {
				t.Errorf("isValidRepo() = %v, want %v", got, tt.want)
			}
		})
	}
}
