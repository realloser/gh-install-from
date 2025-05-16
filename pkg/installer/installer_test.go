package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/realloser/gh-install-from/pkg/github"
	"github.com/realloser/gh-install-from/pkg/log"
	"github.com/realloser/gh-install-from/pkg/metadata"
)

func TestInstaller_Install(t *testing.T) {
	// Initialize logger for tests
	log.Init(false)

	tests := []struct {
		name      string
		repo      string
		host      string
		setupMock func(host string) github.Client
		wantErr   bool
	}{
		{
			name: "successful installation",
			repo: "test/repo",
			host: "github.com",
			setupMock: func(host string) github.Client {
				return &github.MockClient{
					GetLatestReleaseFunc: func(repo string) (*github.Release, error) {
						return &github.Release{
							TagName: "v1.0.0",
							Assets: []github.Asset{
								{
									Name:               "test-darwin-amd64",
									BrowserDownloadURL: "https://example.com/test-darwin-amd64",
								},
							},
						}, nil
					},
					DownloadAssetFunc: func(url, destPath string) error {
						return os.WriteFile(destPath, []byte("test binary"), 0755)
					},
					GetHostFunc: func() string {
						return host
					},
				}
			},
			wantErr: false,
		},
		{
			name: "custom host installation",
			repo: "test/repo",
			host: "github.enterprise.com",
			setupMock: func(host string) github.Client {
				return &github.MockClient{
					GetLatestReleaseFunc: func(repo string) (*github.Release, error) {
						return &github.Release{
							TagName: "v1.0.0",
							Assets: []github.Asset{
								{
									Name:               "test-darwin-amd64",
									BrowserDownloadURL: "https://example.com/test-darwin-amd64",
								},
							},
						}, nil
					},
					DownloadAssetFunc: func(url, destPath string) error {
						return os.WriteFile(destPath, []byte("test binary"), 0755)
					},
					GetHostFunc: func() string {
						return host
					},
				}
			},
			wantErr: false,
		},
		{
			name: "no matching asset",
			repo: "test/repo",
			host: "github.com",
			setupMock: func(host string) github.Client {
				return &github.MockClient{
					GetLatestReleaseFunc: func(repo string) (*github.Release, error) {
						return &github.Release{
							TagName: "v1.0.0",
							Assets: []github.Asset{
								{
									Name:               "test-windows-amd64.exe",
									BrowserDownloadURL: "https://example.com/test-windows-amd64.exe",
								},
							},
						}, nil
					},
					GetHostFunc: func() string {
						return host
					},
				}
			},
			wantErr: true,
		},
		{
			name: "github api error",
			repo: "test/repo",
			host: "github.com",
			setupMock: func(host string) github.Client {
				return &github.MockClient{
					GetLatestReleaseFunc: func(repo string) (*github.Release, error) {
						return nil, fmt.Errorf("api error")
					},
					GetHostFunc: func() string {
						return host
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary home directory
			homeDir := t.TempDir()
			oldHome := os.Getenv("HOME")
			os.Setenv("HOME", homeDir)
			defer os.Setenv("HOME", oldHome)

			// Create installer with mock client
			installer := New(tt.setupMock(tt.host))

			// Run installation
			err := installer.Install(tt.repo)
			if (err != nil) != tt.wantErr {
				t.Errorf("Install() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Verify binary was installed
				binPath := filepath.Join(homeDir, ".local", "bin", "test-darwin-amd64")
				if _, err := os.Stat(binPath); err != nil {
					t.Errorf("binary not installed at %s: %v", binPath, err)
				}

				// Verify metadata was stored
				meta, err := metadata.Load(binPath)
				if err != nil {
					t.Errorf("failed to load metadata: %v", err)
				}

				expectedMeta := &metadata.BinaryMetadata{
					GHHost:     tt.host,
					Repository: tt.repo,
					Version:    "v1.0.0",
					BinaryPath: binPath,
				}

				if *meta != *expectedMeta {
					t.Errorf("metadata mismatch: got %+v, want %+v", meta, expectedMeta)
				}
			}
		})
	}
}
