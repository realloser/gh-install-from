package binary

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/realloser/gh-install-from/pkg/github"
	"github.com/realloser/gh-install-from/pkg/log"
	"github.com/realloser/gh-install-from/pkg/metadata"
)

// mockArchive is used to replace the archive package during testing
type mockArchive struct{}

func (m *mockArchive) ExtractFile(src, destDir string) (string, error) {
	// Create a mock binary in the destination directory
	binaryName := filepath.Base(src)
	binaryPath := filepath.Join(destDir, binaryName)

	// Create parent directory if it doesn't exist
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", err
	}

	// Create a mock binary file
	content := []byte("#!/bin/sh\necho 'test binary'\n")
	if err := os.WriteFile(binaryPath, content, 0755); err != nil {
		return "", err
	}

	return binaryPath, nil
}

// mockClient implements github.Client for testing
type mockClient struct {
	host             string
	latestRelease    *github.Release
	latestReleaseErr error
	downloadAssetErr error
}

func (m *mockClient) GetLatestRelease(repo string) (*github.Release, error) {
	return m.latestRelease, m.latestReleaseErr
}

func (m *mockClient) DownloadAsset(url, destPath string) error {
	if m.downloadAssetErr != nil {
		return m.downloadAssetErr
	}

	// Create parent directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	// Create a dummy file for testing
	content := []byte("#!/bin/sh\necho 'test binary'\n")
	return os.WriteFile(destPath, content, 0755)
}

func (m *mockClient) GetHost() string {
	return m.host
}

func TestManager_Delete(t *testing.T) {
	// Initialize logger for tests
	log.Init(false)

	// Create a temporary home directory
	tmpHome := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", oldHome)

	// Create a mock client
	mockClient := &mockClient{
		host: "github.com",
	}

	// Create a manager instance
	manager, err := New(mockClient)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// Test cases
	tests := []struct {
		name       string
		binaryName string
		setup      func(t *testing.T, manager *Manager) // Setup function to create test files
		wantErr    bool
	}{
		{
			name:       "delete existing binary with metadata",
			binaryName: "test-binary",
			setup: func(t *testing.T, manager *Manager) {
				// Create test binary
				binaryPath := manager.GetBinaryPath("test-binary")
				if err := os.WriteFile(binaryPath, []byte("test binary"), 0755); err != nil {
					t.Fatal(err)
				}

				// Create metadata
				meta := &metadata.BinaryMetadata{
					GHHost:     "github.com",
					Repository: "test/repo",
					Version:    "v1.0.0",
					BinaryPath: binaryPath,
				}
				if err := metadata.Store(meta); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: false,
		},
		{
			name:       "delete existing binary without metadata",
			binaryName: "test-binary-no-meta",
			setup: func(t *testing.T, manager *Manager) {
				// Create test binary only
				binaryPath := manager.GetBinaryPath("test-binary-no-meta")
				if err := os.WriteFile(binaryPath, []byte("test binary"), 0755); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: false,
		},
		{
			name:       "delete non-existent binary",
			binaryName: "nonexistent",
			setup:      func(t *testing.T, manager *Manager) {}, // No setup needed
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run setup
			tt.setup(t, manager)

			// Run delete operation
			err := manager.Delete(tt.binaryName)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify binary was deleted
				binaryPath := manager.GetBinaryPath(tt.binaryName)
				if _, err := os.Stat(binaryPath); !os.IsNotExist(err) {
					t.Error("binary file still exists after deletion")
				}

				// Verify metadata was deleted
				if _, err := metadata.Load(binaryPath); err == nil {
					t.Error("metadata still exists after deletion")
				}
			}
		})
	}
}

func TestManager_New(t *testing.T) {
	// Create a temporary home directory
	tmpHome := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", oldHome)

	mockClient := &mockClient{
		host: "github.com",
	}

	// Test successful creation
	t.Run("successful creation", func(t *testing.T) {
		manager, err := New(mockClient)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		expectedBinDir := filepath.Join(tmpHome, ".local", "bin")
		if manager.GetBinDir() != expectedBinDir {
			t.Errorf("binDir = %v, want %v", manager.GetBinDir(), expectedBinDir)
		}

		// Verify bin directory was created
		if _, err := os.Stat(expectedBinDir); err != nil {
			t.Errorf("bin directory was not created: %v", err)
		}
	})

	// Test with unwritable home directory
	t.Run("unwritable home", func(t *testing.T) {
		// Skip on Windows as it handles permissions differently
		if os.Getenv("GOOS") == "windows" {
			t.Skip("skipping on Windows")
		}

		// Create an unwritable directory
		unwritableHome := filepath.Join(tmpHome, "unwritable")
		if err := os.MkdirAll(unwritableHome, 0500); err != nil {
			t.Fatal(err)
		}
		os.Setenv("HOME", unwritableHome)

		_, err := New(mockClient)
		if err == nil {
			t.Error("New() expected error with unwritable home directory")
		}
	})
}

func TestManager_Install(t *testing.T) {
	// Initialize logger for tests
	log.Init(false)

	// Create a temporary home directory
	tmpHome := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", oldHome)

	// Get current OS/arch for test assets
	osName := runtime.GOOS
	archName := runtime.GOARCH

	tests := []struct {
		name      string
		repo      string
		mockSetup func() *mockClient
		wantErr   bool
		errCheck  func(error) bool // Optional error check function
	}{
		{
			name: "successful installation",
			repo: "test/repo",
			mockSetup: func() *mockClient {
				return &mockClient{
					host: "github.com",
					latestRelease: &github.Release{
						TagName: "v1.0.0",
						Assets: []github.Asset{
							{
								Name:               fmt.Sprintf("test-binary_%s_%s", osName, archName),
								BrowserDownloadURL: "https://example.com/test-binary",
							},
						},
					},
				}
			},
			wantErr: false,
		},
		{
			name: "skip gh extension",
			repo: "test/gh-extension",
			mockSetup: func() *mockClient {
				return &mockClient{
					host: "github.com",
					latestRelease: &github.Release{
						TagName: "v1.0.0",
						Assets: []github.Asset{
							{
								Name:               fmt.Sprintf("gh-extension_%s_%s", osName, archName),
								BrowserDownloadURL: "https://example.com/gh-extension",
							},
						},
					},
				}
			},
			wantErr: true,
			errCheck: func(err error) bool {
				return strings.Contains(err.Error(), "GitHub CLI extensions should be installed using 'gh extension install' instead")
			},
		},
		{
			name: "skip gh extension but install regular binary",
			repo: "test/mixed-binaries",
			mockSetup: func() *mockClient {
				return &mockClient{
					host: "github.com",
					latestRelease: &github.Release{
						TagName: "v1.0.0",
						Assets: []github.Asset{
							{
								Name:               fmt.Sprintf("gh-extension_%s_%s", osName, archName),
								BrowserDownloadURL: "https://example.com/gh-extension",
							},
							{
								Name:               fmt.Sprintf("regular-binary_%s_%s", osName, archName),
								BrowserDownloadURL: "https://example.com/regular-binary",
							},
						},
					},
				}
			},
			wantErr: false,
		},
		{
			name: "failed to get latest release",
			repo: "test/repo",
			mockSetup: func() *mockClient {
				return &mockClient{
					host:             "github.com",
					latestReleaseErr: fmt.Errorf("API error"),
				}
			},
			wantErr: true,
		},
		{
			name: "no matching asset",
			repo: "test/repo",
			mockSetup: func() *mockClient {
				return &mockClient{
					host: "github.com",
					latestRelease: &github.Release{
						TagName: "v1.0.0",
						Assets: []github.Asset{
							{
								Name:               "test-binary_windows_amd64",
								BrowserDownloadURL: "https://example.com/test-binary",
							},
						},
					},
				}
			},
			wantErr: true,
		},
		{
			name: "download error",
			repo: "test/repo",
			mockSetup: func() *mockClient {
				return &mockClient{
					host: "github.com",
					latestRelease: &github.Release{
						TagName: "v1.0.0",
						Assets: []github.Asset{
							{
								Name:               fmt.Sprintf("test-binary_%s_%s", osName, archName),
								BrowserDownloadURL: "https://example.com/test-binary",
							},
						},
					},
					downloadAssetErr: fmt.Errorf("download failed"),
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := tt.mockSetup()
			manager, err := NewWithArchiver(mockClient, &mockArchive{})
			if err != nil {
				t.Fatalf("failed to create manager: %v", err)
			}

			err = manager.Install(tt.repo)
			if (err != nil) != tt.wantErr {
				t.Errorf("Install() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.errCheck != nil && err != nil {
				if !tt.errCheck(err) {
					t.Errorf("Install() error = %v, did not match expected error condition", err)
				}
			}

			if !tt.wantErr {
				// Verify binary was installed
				binaryName := filepath.Base(tt.repo)
				binaryPath := manager.GetBinaryPath(binaryName)
				if _, err := os.Stat(binaryPath); err != nil {
					t.Error("binary file was not created")
				}

				// Verify metadata was stored
				meta, err := metadata.Load(binaryPath)
				if err != nil {
					t.Error("metadata was not stored")
				}
				if meta != nil {
					if meta.Repository != tt.repo {
						t.Errorf("metadata repository = %v, want %v", meta.Repository, tt.repo)
					}
					if meta.Version != mockClient.latestRelease.TagName {
						t.Errorf("metadata version = %v, want %v", meta.Version, mockClient.latestRelease.TagName)
					}
				}
			}
		})
	}
}
