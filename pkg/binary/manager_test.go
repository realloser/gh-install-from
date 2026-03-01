//go:build !windows

package binary

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/realloser/gh-install-from/pkg/config"
	"github.com/realloser/gh-install-from/pkg/fs"
	"github.com/realloser/gh-install-from/pkg/github"
	"github.com/realloser/gh-install-from/pkg/log"
	"github.com/realloser/gh-install-from/pkg/metadata"
	"github.com/realloser/gh-install-from/pkg/path"
)

// mockArchive is used to replace the archive package during testing
type mockArchive struct{}

func (m *mockArchive) ExtractFile(src, destDir string) (string, error) {
	// Extract to a simple binary name (e.g. "repo" for test/repo)
	base := filepath.Base(src)
	binaryName := strings.TrimSuffix(strings.TrimSuffix(base, ".tar.gz"), ".zip")
	if idx := strings.Index(binaryName, "_"); idx > 0 {
		binaryName = binaryName[:idx] // "test-binary_linux_amd64" -> "test-binary"
	}
	binaryPath := filepath.Join(destDir, binaryName)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", err
	}
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
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}
	content := []byte("#!/bin/sh\necho 'test binary'\n")
	return os.WriteFile(destPath, content, 0755)
}

func (m *mockClient) GetHost() string {
	return m.host
}

func setupManagerForTest(t *testing.T, mc *mockClient) Manager {
	t.Helper()
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	os.Setenv("GH_INSTALL_FROM_HOME", tmpHome+"/.gh-install-from")
	t.Cleanup(func() {
		os.Unsetenv("HOME")
		os.Unsetenv("GH_INSTALL_FROM_HOME")
	})

	pathMgr, err := path.New()
	if err != nil {
		t.Fatalf("path.New: %v", err)
	}
	for _, d := range []string{pathMgr.GetBinDir(), pathMgr.GetDownloadsDir(), pathMgr.GetMetadataDir()} {
		if err := os.MkdirAll(d, 0750); err != nil {
			t.Fatalf("MkdirAll %s: %v", d, err)
		}
	}

	store, err := metadata.NewStore(&config.Config{Store: "json"})
	if err != nil {
		t.Fatalf("metadata.NewStore: %v", err)
	}

	osSvc, err := fs.NewOSService(&config.Config{OS: "unix"})
	if err != nil {
		t.Fatalf("fs.NewOSService: %v", err)
	}

	return NewWithDeps(pathMgr, mc, store, osSvc, &mockArchive{})
}

func TestManager_Remove(t *testing.T) {
	log.Init(false)

	mc := &mockClient{host: "github.com"}
	manager := setupManagerForTest(t, mc)

	// Setup: create target, symlink, metadata
	pathMgr, _ := path.New()
	targetPath := filepath.Join(pathMgr.GetDownloadsDir(), "test", "repo", "1.0.0", "test-binary")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetPath, []byte("test"), 0755); err != nil {
		t.Fatal(err)
	}
	binPath := filepath.Join(pathMgr.GetBinDir(), "test-binary")
	if err := os.Symlink(targetPath, binPath); err != nil {
		t.Fatal(err)
	}
	store, _ := metadata.NewStore(nil)
	store.Store(&metadata.BinaryMetadata{
		GHHost:     "github.com",
		Repository: "test/repo",
		Version:    "v1.0.0",
		BinaryPath: binPath,
	})

	err := manager.Remove("test-binary")
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}

	if _, err := os.Lstat(binPath); !os.IsNotExist(err) {
		t.Error("symlink still exists after Remove")
	}
}

func TestManager_NewManager(t *testing.T) {
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	os.Setenv("GH_INSTALL_FROM_HOME", tmpHome+"/.gh-install-from")
	t.Cleanup(func() {
		os.Unsetenv("HOME")
		os.Unsetenv("GH_INSTALL_FROM_HOME")
	})

	manager, err := NewManager(&config.Config{Client: "mock"})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	expectedBinDir := filepath.Join(tmpHome, ".gh-install-from", "bin")
	if manager.GetBinDir() != expectedBinDir {
		t.Errorf("GetBinDir() = %v, want %v", manager.GetBinDir(), expectedBinDir)
	}
}

func TestManager_Install(t *testing.T) {
	log.Init(false)

	osName := runtime.GOOS
	archName := runtime.GOARCH

	tests := []struct {
		name      string
		repo      string
		mockSetup func() *mockClient
		wantErr   bool
		errCheck  func(error) bool
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
								Name:               fmt.Sprintf("test-binary_%s_%s.tar.gz", osName, archName),
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
				return strings.Contains(err.Error(), "no matching binary found")
			},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := tt.mockSetup()
			manager := setupManagerForTest(t, mc)

			err := manager.Install(tt.repo)
			if (err != nil) != tt.wantErr {
				t.Errorf("Install() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.errCheck != nil && err != nil && !tt.errCheck(err) {
				t.Errorf("Install() error = %v", err)
			}
			if !tt.wantErr {
				binaryName := filepath.Base(tt.repo)
				binPath := filepath.Join(manager.GetBinDir(), binaryName)
				if _, err := os.Lstat(binPath); err != nil {
					t.Error("symlink was not created")
				}
				store, _ := metadata.NewStore(nil)
				meta, err := store.Load(binaryName)
				if err != nil || meta == nil {
					t.Error("metadata was not stored")
				}
			}
		})
	}
}
