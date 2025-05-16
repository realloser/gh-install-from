package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/realloser/gh-install-from/pkg/github"
	"github.com/realloser/gh-install-from/pkg/log"
)

func TestRunInstall(t *testing.T) {
	// Initialize logger
	log.Init(false)

	tests := []struct {
		name    string
		repo    string
		mock    *github.MockClient
		wantErr bool
	}{
		{
			name: "successful installation",
			repo: "test/repo",
			mock: &github.MockClient{
				GetLatestReleaseFunc: func(repo string) (*github.Release, error) {
					return &github.Release{
						Assets: []github.Asset{
							{
								Name:               "test-darwin-amd64",
								BrowserDownloadURL: "https://example.com/test-darwin-amd64",
							},
						},
					}, nil
				},
				DownloadAssetFunc: func(url, destPath string) error {
					// Create a dummy binary file
					return os.WriteFile(destPath, []byte("test binary"), 0755)
				},
			},
			wantErr: false,
		},
		{
			name: "no matching asset",
			repo: "test/repo",
			mock: &github.MockClient{
				GetLatestReleaseFunc: func(repo string) (*github.Release, error) {
					return &github.Release{
						Assets: []github.Asset{
							{
								Name:               "test-linux-amd64",
								BrowserDownloadURL: "https://example.com/test-linux-amd64",
							},
						},
					}, nil
				},
			},
			wantErr: true,
		},
		{
			name: "github api error",
			repo: "test/repo",
			mock: &github.MockClient{
				GetLatestReleaseFunc: func(repo string) (*github.Release, error) {
					return nil, fmt.Errorf("api error")
				},
			},
			wantErr: true,
		},
		{
			name: "invalid repo format",
			repo: "invalid/repo/format",
			mock: &github.MockClient{
				GetLatestReleaseFunc: func(repo string) (*github.Release, error) {
					return nil, fmt.Errorf("invalid repository format")
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Override the NewGhCliClient function
			origNewClient := github.NewGhCliClient
			github.NewGhCliClient = func() (github.Client, error) {
				return tt.mock, nil
			}
			defer func() { github.NewGhCliClient = origNewClient }()

			// Create a temporary home directory
			tmpHome := t.TempDir()
			origHome := os.Getenv("HOME")
			os.Setenv("HOME", tmpHome)
			defer os.Setenv("HOME", origHome)

			// Create the bin directory
			binDir := filepath.Join(tmpHome, ".local", "bin")
			if err := os.MkdirAll(binDir, 0750); err != nil {
				t.Fatal(err)
			}

			err := runInstall(nil, []string{tt.repo})
			if (err != nil) != tt.wantErr {
				t.Errorf("runInstall() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Check if the binary was installed
				binPath := filepath.Join(binDir, "test-darwin-amd64")
				if _, err := os.Stat(binPath); err != nil {
					t.Errorf("binary not installed at %s: %v", binPath, err)
				}
			}
		})
	}
}
