package github

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
if [ "$1" = "auth" ] && [ "$2" = "status" ]; then
	exit 1
fi
exit 0
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

	// Test with custom host from auth status --json hosts
	t.Run("custom host", func(t *testing.T) {
		// Create mock gh that returns a custom host via --json hosts
		err := os.WriteFile(mockGh, []byte(`#!/bin/sh
if [ "$1" = "auth" ] && [ "$2" = "status" ]; then
	echo '{"hosts":{"github.enterprise.com":[{"state":"success","active":true,"host":"github.enterprise.com","login":"testuser"}]}}'
	exit 0
fi
exit 1
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

	// Test with multiple hosts — active host is preferred
	t.Run("multiple hosts active preferred", func(t *testing.T) {
		err := os.WriteFile(mockGh, []byte(`#!/bin/sh
if [ "$1" = "auth" ] && [ "$2" = "status" ]; then
	echo '{"hosts":{"github.com":[{"state":"success","active":false,"host":"github.com","login":"user"}],"git.corp.com":[{"state":"success","active":true,"host":"git.corp.com","login":"user"}]}}'
	exit 0
fi
exit 1
`), 0755)
		if err != nil {
			t.Fatal(err)
		}

		client, err := newGhCliClient()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if host := client.GetHost(); host != "git.corp.com" {
			t.Errorf("expected active host git.corp.com, got %s", host)
		}
	})

	// Test with malformed auth status output
	t.Run("malformed auth status", func(t *testing.T) {
		// Create mock gh that returns malformed output
		err := os.WriteFile(mockGh, []byte(`#!/bin/sh
if [ "$1" = "auth" ] && [ "$2" = "status" ]; then
	echo "Invalid output format"
	exit 0
fi
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

	// Test with real gh auth status --json hosts output format
	t.Run("real auth status format", func(t *testing.T) {
		// Create mock gh that returns real --json hosts output
		err := os.WriteFile(mockGh, []byte(`#!/bin/sh
if [ "$1" = "auth" ] && [ "$2" = "status" ]; then
	echo '{"hosts":{"github.com":[{"state":"success","active":true,"host":"github.com","login":"realloser"}]}}'
	exit 0
fi
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
			t.Errorf("expected host github.com, got %s", host)
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
if [ "$1" = "api" ] && [ "$2" = "repos/valid/repo/releases/latest" ]; then
	echo '{"tag_name":"v1.0.0","assets":[{"name":"test.tar.gz","browser_download_url":"https://example.com/test.tar.gz"}]}'
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
				TagName: "v1.0.0",
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
			if !tt.wantErr && got != nil {
				if got.TagName != tt.want.TagName {
					t.Errorf("GetLatestRelease() tag = %v, want %v", got.TagName, tt.want.TagName)
				}
				if len(got.Assets) != len(tt.want.Assets) {
					t.Errorf("GetLatestRelease() assets count = %v, want %v", len(got.Assets), len(tt.want.Assets))
				}
				if len(got.Assets) > 0 {
					if got.Assets[0].Name != tt.want.Assets[0].Name {
						t.Errorf("GetLatestRelease() asset name = %v, want %v", got.Assets[0].Name, tt.want.Assets[0].Name)
					}
					if got.Assets[0].BrowserDownloadURL != tt.want.Assets[0].BrowserDownloadURL {
						t.Errorf("GetLatestRelease() asset URL = %v, want %v", got.Assets[0].BrowserDownloadURL, tt.want.Assets[0].BrowserDownloadURL)
					}
				}
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
if [ "$1" = "api" ] && [ "$2" = "https://example.com/test.tar.gz" ]; then
	echo "test data"
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
			name:     "valid url",
			url:      "https://example.com/test.tar.gz",
			destPath: filepath.Join(tmpDir, "test.tar.gz"),
			wantErr:  false,
		},
		{
			name:     "invalid url",
			url:      "https://example.com/invalid.tar.gz",
			destPath: filepath.Join(tmpDir, "invalid.tar.gz"),
			wantErr:  true,
		},
	}

	client := &ghCliClient{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.DownloadAsset(tt.url, tt.destPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("DownloadAsset() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Verify file exists and has correct content
				data, err := os.ReadFile(tt.destPath)
				if err != nil {
					t.Errorf("failed to read downloaded file: %v", err)
					return
				}
				if strings.TrimSpace(string(data)) != "test data" {
					t.Errorf("downloaded file content = %v, want %v", string(data), "test data")
				}
			}
		})
	}
}

func TestGetLatestRelease(t *testing.T) {
	// Create a temporary directory for our mock gh command
	tmpDir, err := os.MkdirTemp("", "gh-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a mock gh command
	mockGh := filepath.Join(tmpDir, "gh")
	err = os.WriteFile(mockGh, []byte(`#!/bin/sh
if [ "$1" = "api" ] && [ "$2" = "repos/valid/repo/releases/latest" ]; then
	echo '{"tag_name":"v1.0.0","assets":[{"name":"test.tar.gz","browser_download_url":"https://example.com/test.tar.gz"}]}'
	exit 0
else
	echo "gh: Not Found (HTTP 404)" >&2
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
		wantErr bool
	}{
		{
			name:    "invalid empty repo",
			repo:    "",
			wantErr: true,
		},
		{
			name:    "invalid repo format",
			repo:    "invalid",
			wantErr: true,
		},
		{
			name:    "valid repo format",
			repo:    "valid/repo",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetLatestRelease(tt.repo)
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
			repo: "a/b" + string(make([]byte, 256)),
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

// TestGetLatestRelease_MultiHostProbe verifies the core multi-host behavior:
// when a repo is not found on the first host (404), the client probes the next
// authenticated host and succeeds there. This is the regression guard for the
// "404 on the wrong host" bug.
func TestGetLatestRelease_MultiHostProbe(t *testing.T) {
	tmpDir := t.TempDir()
	mockGh := filepath.Join(tmpDir, "gh")

	// Mock gh: auth status returns two hosts (github.com inactive,
	// git.enterprise.com active). api calls 404 on github.com but succeed on
	// the enterprise host.
	err := os.WriteFile(mockGh, []byte(`#!/bin/sh
if [ "$1" = "auth" ] && [ "$2" = "status" ]; then
	echo '{"hosts":{"github.com":[{"state":"success","active":false,"host":"github.com","login":"user"}],"git.enterprise.com":[{"state":"success","active":true,"host":"git.enterprise.com","login":"user"}]}}'
	exit 0
fi
if [ "$1" = "api" ]; then
	# Determine the target host: --hostname <h> may be $2/$3, or absent (github.com).
	host="github.com"
	if [ "$2" = "--hostname" ]; then
		host="$3"
	fi
	if [ "$host" = "git.enterprise.com" ] && echo "$@" | grep -q "repos/acme/tool/releases/latest"; then
		echo '{"tag_name":"v2.0.0","assets":[{"name":"tool","browser_download_url":"https://git.enterprise.com/tool"}]}'
		exit 0
	fi
	echo "gh: Not Found (HTTP 404)" >&2
	exit 1
fi
exit 1
`), 0755)
	if err != nil {
		t.Fatal(err)
	}

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", fmt.Sprintf("%s:%s", tmpDir, origPath))
	defer os.Setenv("PATH", origPath)

	client, err := newGhCliClient()
	if err != nil {
		t.Fatalf("newGhCliClient: %v", err)
	}

	release, err := client.GetLatestRelease("acme/tool")
	if err != nil {
		t.Fatalf("GetLatestRelease should succeed by probing the enterprise host: %v", err)
	}
	if release.TagName != "v2.0.0" {
		t.Errorf("tag = %s, want v2.0.0", release.TagName)
	}
	// The winning host must be remembered for the subsequent download.
	if host := client.GetHost(); host != "git.enterprise.com" {
		t.Errorf("resolved host = %s, want git.enterprise.com (must be remembered for download)", host)
	}
}

// TestGetLatestRelease_NotOnAnyHost verifies that when the repo 404s on every
// authenticated host, the error lists the hosts tried.
func TestGetLatestRelease_NotOnAnyHost(t *testing.T) {
	tmpDir := t.TempDir()
	mockGh := filepath.Join(tmpDir, "gh")

	err := os.WriteFile(mockGh, []byte(`#!/bin/sh
if [ "$1" = "auth" ] && [ "$2" = "status" ]; then
	echo '{"hosts":{"github.com":[{"state":"success","active":true,"host":"github.com","login":"user"}],"git.enterprise.com":[{"state":"success","active":false,"host":"git.enterprise.com","login":"user"}]}}'
	exit 0
fi
if [ "$1" = "api" ]; then
	echo "gh: Not Found (HTTP 404)" >&2
	exit 1
fi
exit 1
`), 0755)
	if err != nil {
		t.Fatal(err)
	}

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", fmt.Sprintf("%s:%s", tmpDir, origPath))
	defer os.Setenv("PATH", origPath)

	client, err := newGhCliClient()
	if err != nil {
		t.Fatalf("newGhCliClient: %v", err)
	}

	_, err = client.GetLatestRelease("acme/missing")
	if err == nil {
		t.Fatal("expected error when repo not found on any host")
	}
	if !strings.Contains(err.Error(), "not found on any authenticated host") {
		t.Errorf("error should mention 'not found on any authenticated host', got: %v", err)
	}
}

// TestGetLatestRelease_AmbiguousMultiHost verifies that when a repo exists on
// multiple authenticated hosts, the install is aborted with a message listing
// both URLs and the exact --host command to disambiguate — never silently
// guessing.
func TestGetLatestRelease_AmbiguousMultiHost(t *testing.T) {
	tmpDir := t.TempDir()
	mockGh := filepath.Join(tmpDir, "gh")

	// Mock gh: two hosts, both have the repo -> ambiguous.
	err := os.WriteFile(mockGh, []byte(`#!/bin/sh
if [ "$1" = "auth" ] && [ "$2" = "status" ]; then
	echo '{"hosts":{"github.com":[{"state":"success","active":true,"host":"github.com","login":"user"}],"git.enterprise.com":[{"state":"success","active":true,"host":"git.enterprise.com","login":"user"}]}}'
	exit 0
fi
if [ "$1" = "api" ] && echo "$@" | grep -q "repos/acme/tool/releases/latest"; then
	echo '{"tag_name":"v1.0.0","assets":[]}'
	exit 0
fi
echo "gh: Not Found (HTTP 404)" >&2
exit 1
`), 0755)
	if err != nil {
		t.Fatal(err)
	}

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", fmt.Sprintf("%s:%s", tmpDir, origPath))
	defer os.Setenv("PATH", origPath)

	client, err := newGhCliClient()
	if err != nil {
		t.Fatalf("newGhCliClient: %v", err)
	}

	_, err = client.GetLatestRelease("acme/tool")
	if err == nil {
		t.Fatal("expected ambiguous-match error when repo exists on multiple hosts")
	}
	msg := err.Error()
	if !strings.Contains(msg, "multiple authenticated hosts") {
		t.Errorf("error should mention 'multiple authenticated hosts', got: %v", err)
	}
	// Both URLs must be listed.
	if !strings.Contains(msg, "https://github.com/acme/tool") {
		t.Errorf("error should list github.com URL, got: %v", err)
	}
	if !strings.Contains(msg, "https://git.enterprise.com/acme/tool") {
		t.Errorf("error should list enterprise URL, got: %v", err)
	}
	// The disambiguation command must show the full URL form.
	if !strings.Contains(msg, "gh install-from https://") {
		t.Errorf("error should show the full-URL disambiguation command, got: %v", err)
	}
}

// TestGetLatestRelease_ExplicitHost verifies that --host pins the client: it
// probes only the specified host and does not enumerate or disambiguate.
func TestGetLatestRelease_ExplicitHost(t *testing.T) {
	tmpDir := t.TempDir()
	mockGh := filepath.Join(tmpDir, "gh")

	// Mock gh: api succeeds only on git.enterprise.com; auth status lists both.
	err := os.WriteFile(mockGh, []byte(`#!/bin/sh
if [ "$1" = "auth" ] && [ "$2" = "status" ]; then
	echo '{"hosts":{"github.com":[{"state":"success","active":true,"host":"github.com","login":"user"}],"git.enterprise.com":[{"state":"success","active":false,"host":"git.enterprise.com","login":"user"}]}}'
	exit 0
fi
if [ "$1" = "api" ]; then
	host="github.com"
	if [ "$2" = "--hostname" ]; then host="$3"; fi
	if [ "$host" = "git.enterprise.com" ] && echo "$@" | grep -q "repos/acme/tool/releases/latest"; then
		echo '{"tag_name":"v2.0.0","assets":[]}'
		exit 0
	fi
	echo "gh: Not Found (HTTP 404)" >&2
	exit 1
fi
exit 1
`), 0755)
	if err != nil {
		t.Fatal(err)
	}

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", fmt.Sprintf("%s:%s", tmpDir, origPath))
	defer os.Setenv("PATH", origPath)

	// Pass the full URL to pin the host — no probing of github.com, no ambiguity.
	client, err := newGhCliClient()
	if err != nil {
		t.Fatalf("newGhCliClient: %v", err)
	}

	release, err := client.GetLatestRelease("https://git.enterprise.com/acme/tool")
	if err != nil {
		t.Fatalf("full URL should succeed without ambiguity: %v", err)
	}
	if release.TagName != "v2.0.0" {
		t.Errorf("tag = %s, want v2.0.0", release.TagName)
	}
	if host := client.GetHost(); host != "git.enterprise.com" {
		t.Errorf("host = %s, want git.enterprise.com", host)
	}
}
