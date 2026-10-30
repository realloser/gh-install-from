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

// mockArchiveProcessor replaces the archive.Processor (archive path) during tests.
type mockArchiveProcessor struct {
	err error // if set, Process returns this error
}

func (m *mockArchiveProcessor) Process(src, destDir string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return writeMockBinary(src, destDir, "test-binary"), nil
}

// mockBinaryProcessor replaces the binary.Processor (bare-binary path) during tests.
type mockBinaryProcessor struct {
	err   error // if set, Process returns this error
	calls int   // records how many times Process was invoked
}

func (m *mockBinaryProcessor) Process(src, destDir string) (string, error) {
	m.calls++
	if m.err != nil {
		return "", m.err
	}
	return writeMockBinary(src, destDir, "mycli"), nil
}

// writeMockBinary writes a non-trivial shell script to destDir under the given
// name and returns its path, mirroring what the real processors produce.
func writeMockBinary(src, destDir, name string) string {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return ""
	}
	binaryPath := filepath.Join(destDir, name)
	_ = os.WriteFile(binaryPath, []byte("#!/bin/sh\necho 'test binary'\n"), 0755)
	return binaryPath
}

// mockClient implements github.Client for testing
type mockClient struct {
	host             string
	latestRelease    *github.Release
	latestReleaseErr error
	downloadAssetErr error
	lastRepo         string // repo argument of the most recent GetLatestRelease call
}

func (m *mockClient) GetLatestRelease(repo string) (*github.Release, error) {
	m.lastRepo = repo
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

	return NewWithDeps(pathMgr, mc, store, osSvc, &mockArchiveProcessor{}, &mockBinaryProcessor{}, nil)
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
			name: "full URL on a non-default host",
			repo: "https://git.example.com/test/repo",
			mockSetup: func() *mockClient {
				return &mockClient{
					host: "git.example.com",
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
			name: "checksums file is never installed as the binary",
			repo: "test/gh-extension",
			mockSetup: func() *mockClient {
				return &mockClient{
					host: "github.com",
					latestRelease: &github.Release{
						TagName: "v1.0.0",
						Assets: []github.Asset{
							{Name: "checksums.txt", BrowserDownloadURL: "https://example.com/checksums.txt"},
							{Name: fmt.Sprintf("gh-extension-%s-%s", osName, archName), BrowserDownloadURL: "https://example.com/gh-extension"},
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

func TestPinnedRepo(t *testing.T) {
	tests := []struct{ repo, host, want string }{
		{"owner/repo", "ghe.example.com", "https://ghe.example.com/owner/repo"},
		{"owner/repo", "github.com", "https://github.com/owner/repo"},
		{"owner/repo", "", "owner/repo"},
		{"https://ghe.example.com/owner/repo", "ghe.example.com", "https://ghe.example.com/owner/repo"},
		{"ghe.example.com/owner/repo", "ghe.example.com", "ghe.example.com/owner/repo"},
	}
	for _, tt := range tests {
		if got := pinnedRepo(tt.repo, tt.host); got != tt.want {
			t.Errorf("pinnedRepo(%q, %q) = %q, want %q", tt.repo, tt.host, got, tt.want)
		}
	}
}

func TestManager_UpdatePinsInstalledHost(t *testing.T) {
	log.Init(false)
	mc := &mockClient{
		host: "ghe.example.com",
		latestRelease: &github.Release{
			TagName: "v1.0.0",
			Assets: []github.Asset{{
				Name:               fmt.Sprintf("tool_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH),
				BrowserDownloadURL: "https://example.com/tool",
			}},
		},
	}
	manager := setupManagerForTest(t, mc)

	// Install through a host-qualified URL: owner/repo and host are stored separately.
	if err := manager.Install("https://ghe.example.com/test/tool"); err != nil {
		t.Fatalf("Install: %v", err)
	}
	installed, err := manager.ListInstalled()
	if err != nil || len(installed) != 1 {
		t.Fatalf("ListInstalled = %v, %v", installed, err)
	}
	if installed[0].Repository != "test/tool" || installed[0].Host != "ghe.example.com" {
		t.Errorf("stored Repository/Host = %q/%q, want test/tool / ghe.example.com", installed[0].Repository, installed[0].Host)
	}

	// Updating by plain owner/repo, by name, or by URL must all pin the stored host.
	for i, ref := range []string{"test/tool", "tool", "https://ghe.example.com/test/tool"} {
		mc.latestRelease.TagName = fmt.Sprintf("v1.%d.0", i+1) // a real update has a newer version
		mc.lastRepo = ""
		if err := manager.Update(ref); err != nil {
			t.Fatalf("Update(%q): %v", ref, err)
		}
		if want := "https://ghe.example.com/test/tool"; mc.lastRepo != want {
			t.Errorf("Update(%q) looked up %q, want %q", ref, mc.lastRepo, want)
		}
	}
}

func TestManager_InstallRefusesNameCollision(t *testing.T) {
	log.Init(false)
	rel := func() *github.Release {
		return &github.Release{
			TagName: "v1.0.0",
			Assets: []github.Asset{{
				Name:               fmt.Sprintf("tool_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH),
				BrowserDownloadURL: "https://example.com/tool",
			}},
		}
	}
	mc := &mockClient{host: "ghe.example.com", latestRelease: rel()}
	manager := setupManagerForTest(t, mc)

	if err := manager.Install("https://ghe.example.com/test/tool"); err != nil {
		t.Fatalf("first Install: %v", err)
	}

	// Same name, different owner: must be refused and leave the original intact.
	if err := manager.Install("https://ghe.example.com/other/tool"); err == nil {
		t.Fatal("expected collision error for a different owner, got nil")
	}
	// Same owner/repo on a different host: also refused.
	mc.host = "github.com"
	if err := manager.Install("https://github.com/test/tool"); err == nil {
		t.Fatal("expected collision error for a different host, got nil")
	}
	installed, err := manager.ListInstalled()
	if err != nil || len(installed) != 1 || installed[0].Repository != "test/tool" || installed[0].Host != "ghe.example.com" {
		t.Fatalf("original install was disturbed: %v, %v", installed, err)
	}

	// Reinstalling the same repo on the same host is still fine, regardless of
	// the case the user types (GitHub names are case-insensitive).
	mc.host = "GHE.Example.com"
	if err := manager.Install("https://GHE.Example.com/Test/Tool"); err != nil {
		t.Fatalf("mixed-case reinstall: %v", err)
	}
	mc.host = "ghe.example.com"
	if err := manager.Install("https://ghe.example.com/test/tool"); err != nil {
		t.Fatalf("reinstall of same repo: %v", err)
	}
}
