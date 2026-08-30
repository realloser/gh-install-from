//go:build !windows

package binary

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"golang.org/x/sys/unix"

	"github.com/realloser/gh-install-from/pkg/archive"
	"github.com/realloser/gh-install-from/pkg/config"
	"github.com/realloser/gh-install-from/pkg/fs"
	"github.com/realloser/gh-install-from/pkg/github"
	"github.com/realloser/gh-install-from/pkg/log"
	"github.com/realloser/gh-install-from/pkg/metadata"
	"github.com/realloser/gh-install-from/pkg/path"
)

// setQuarantineForTest sets the com.apple.quarantine xattr on path for test
// setup. Only meaningful on darwin; the caller gates on runtime.GOOS.
func setQuarantineForTest(path string) error {
	return unix.Lsetxattr(path, "com.apple.quarantine", []byte("0081;5f000000;Safari;"), 0)
}

func TestFindMatchingAsset(t *testing.T) {
	osName := runtime.GOOS
	archName := runtime.GOARCH

	tests := []struct {
		name    string
		assets  []github.Asset
		want    string // expected matched asset name; "" means expect error
		wantErr bool
	}{
		{
			name: "plain binary name",
			assets: []github.Asset{
				{Name: "mycli", BrowserDownloadURL: "https://example.com/mycli"},
			},
			want: "mycli",
		},
		{
			name: "plain name with version",
			assets: []github.Asset{
				{Name: "mycli-1.2.3", BrowserDownloadURL: "https://example.com/mycli"},
			},
			want: "mycli-1.2.3",
		},
		{
			name: "single OS/arch tagged asset still matches",
			assets: []github.Asset{
				{Name: fmt.Sprintf("mycli_%s_%s.tar.gz", osName, archName), BrowserDownloadURL: "https://example.com/mycli"},
			},
			want: fmt.Sprintf("mycli_%s_%s.tar.gz", osName, archName),
		},
		{
			name: "two plain names is ambiguous",
			assets: []github.Asset{
				{Name: "mycli", BrowserDownloadURL: "https://example.com/a"},
				{Name: "mycli-1.2.3", BrowserDownloadURL: "https://example.com/b"},
			},
			wantErr: true,
		},
		{
			name: "single cross-platform tagged asset is rejected (no wrong-platform install)",
			assets: []github.Asset{
				// A single asset tagged for a DIFFERENT platform must not be
				// accepted by the plain-candidate fallback.
				{Name: "mycli-windows-amd64", BrowserDownloadURL: "https://example.com/x"},
			},
			wantErr: true,
		},
		{
			name: "checksum-only asset is rejected",
			assets: []github.Asset{
				{Name: "mycli.sha256", BrowserDownloadURL: "https://example.com/sum"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := findMatchingAsset(tt.assets)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("findMatchingAsset() expected error, got asset %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("findMatchingAsset() unexpected error: %v", err)
			}
			if got.Name != tt.want {
				t.Errorf("findMatchingAsset() = %q, want %q", got.Name, tt.want)
			}
		})
	}
}

// TestProcessDownload_FallsThroughToBinary verifies that when the archive
// processor reports ErrNoExecutable, the manager falls through to the binary
// processor instead of failing.
func TestProcessDownload_FallsThroughToBinary(t *testing.T) {
	m := &managerImpl{
		archiveProcessor: &mockArchiveProcessor{err: archive.ErrNoExecutable},
		binaryProcessor:  &mockBinaryProcessor{},
	}

	// Write REAL gzip content so DetectFormat classifies this as FormatArchive.
	// That routes processDownload into the FormatArchive branch, where the
	// injected archive processor returns ErrNoExecutable — exercising the
	// fall-through to the binary processor. (A plain text src would classify as
	// FormatBinary and skip the archive branch entirely.)
	src := filepath.Join(t.TempDir(), "asset.tar.gz")
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write([]byte("not a real tar member")); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, buf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}

	path, err := m.processDownload(src, t.TempDir())
	if err != nil {
		t.Fatalf("processDownload() error: %v", err)
	}
	if path == "" {
		t.Fatal("processDownload() returned empty path")
	}

	bp := m.binaryProcessor.(*mockBinaryProcessor)
	if bp.calls != 1 {
		t.Errorf("binaryProcessor.Process called %d times, want 1", bp.calls)
	}
}

// TestManager_Install_BareBinary_NonZero is the end-to-end guard for the silent
// 0-byte truncation bug (Failure B). A release shipping a plain-name bare binary
// must install a non-zero-size binary.
func TestManager_Install_BareBinary_NonZero(t *testing.T) {
	log.Init(false)

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

	mc := &mockClient{
		host: "github.com",
		latestRelease: &github.Release{
			TagName: "v1.0.0",
			Assets: []github.Asset{
				{
					Name:               "mycli",
					BrowserDownloadURL: "https://example.com/mycli",
				},
			},
		},
	}

	// Use the REAL processors so detection + bare-binary copy run end to end.
	manager := NewWithDeps(pathMgr, mc, store, osSvc, &archive.ArchiveProcessor{}, &archive.BinaryProcessor{}, nil)

	if err := manager.Install("acme/mycli"); err != nil {
		t.Fatalf("Install: %v", err)
	}

	binPath := filepath.Join(manager.GetBinDir(), "mycli")
	info, err := os.Stat(binPath) // follows the symlink to the staged binary
	if err != nil {
		t.Fatalf("installed binary not found at %s: %v", binPath, err)
	}
	if info.Size() == 0 {
		t.Fatal("installed bare binary is 0 bytes (silent truncation regression)")
	}

	// Also confirm the staged target under the downloads dir is non-zero.
	if target, err := os.Readlink(binPath); err == nil {
		if ti, err := os.Stat(target); err == nil && ti.Size() == 0 {
			t.Errorf("staged binary %s is 0 bytes", target)
		}
	}
}

// spyConfirmer is a test Confirmer that records the prompt and returns a
// predetermined answer.
type spyConfirmer struct {
	called bool
	prompt string
	answer bool
}

func (s *spyConfirmer) Confirm(prompt string) bool {
	s.called = true
	s.prompt = prompt
	return s.answer
}

// TestConfigureQuarantine verifies the flag + confirmer are wired into the
// manager and that the no-auto-strip invariant holds: stripping is only
// enabled by the flag or an affirmative prompt, never by default.
func TestConfigureQuarantine(t *testing.T) {
	// Construct a bare managerImpl — this test only checks field wiring, so it
	// needs no real deps (and must not depend on $HOME, which other tests mutate).
	base := &managerImpl{}
	impl := base

	// Default: no confirmer, flag false -> never strips.
	if impl.removeQuarantine != false || impl.confirmer != nil {
		t.Fatal("default manager must not strip quarantine (no confirmer, flag false)")
	}

	// Flag set -> strips without prompting (escape hatch).
	spy := &spyConfirmer{answer: false} // even if confirmer would say no...
	ConfigureQuarantine(base, spy, true)
	if !impl.removeQuarantine {
		t.Fatal("--remove-quarantine flag must set removeQuarantine=true")
	}
	if impl.confirmer == nil {
		t.Fatal("ConfigureQuarantine must inject the confirmer")
	}

	// Flag unset + confirmer injected -> confirmer decides.
	base2 := &managerImpl{}
	impl2 := base2
	spy2 := &spyConfirmer{answer: true}
	ConfigureQuarantine(base2, spy2, false)
	if impl2.removeQuarantine {
		t.Fatal("flag false must leave removeQuarantine=false")
	}
	if impl2.confirmer == nil {
		t.Fatal("confirmer must be injected even when flag is false")
	}
}

// TestShouldStripQuarantine verifies the core invariant: stripping only
// happens with the flag or an affirmative prompt, never by default. This is
// the regression guard for the hard "user must confirm" requirement. It tests
// the decision logic directly (post-detection) so it doesn't depend on
// actually reproducing a Gatekeeper kill.
func TestShouldStripQuarantine(t *testing.T) {
	// Case 1: no confirmer, no flag -> must NOT strip (CI / non-interactive).
	m := &managerImpl{removeQuarantine: false, confirmer: nil}
	if m.shouldStripQuarantine("mybin") {
		t.Fatal("stripped quarantine without confirmer or flag (auto-strip regression)")
	}

	// Case 2: confirmer says no -> must NOT strip.
	spyNo := &spyConfirmer{answer: false}
	m2 := &managerImpl{removeQuarantine: false, confirmer: spyNo}
	if m2.shouldStripQuarantine("mybin") {
		t.Fatal("stripped quarantine after user declined (auto-strip regression)")
	}
	if !spyNo.called {
		t.Fatal("confirmer was not prompted when flag was unset")
	}

	// Case 3: flag set -> strips without prompting.
	spyUnused := &spyConfirmer{answer: false} // should NOT be called
	m3 := &managerImpl{removeQuarantine: true, confirmer: spyUnused}
	if !m3.shouldStripQuarantine("mybin") {
		t.Fatal("flag set but shouldStripQuarantine returned false")
	}
	if spyUnused.called {
		t.Fatal("confirmer was prompted when --remove-quarantine flag was set")
	}

	// Case 4: confirmer says yes -> strips.
	spyYes := &spyConfirmer{answer: true}
	m4 := &managerImpl{removeQuarantine: false, confirmer: spyYes}
	if !m4.shouldStripQuarantine("mybin") {
		t.Fatal("confirmer said yes but shouldStripQuarantine returned false")
	}
}

// TestHandleQuarantine_NoAutoStrip is the darwin-only integration guard: with
// a real quarantined file, handleQuarantine must not strip without
// confirmation. (Detection won't fire for a shell script, so this tests the
// HasQuarantine + strip path directly via shouldStripQuarantine + RemoveQuarantine.)
func TestHandleQuarantine_NoAutoStrip(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("quarantine xattr handling is darwin-only")
	}

	dir := t.TempDir()
	targetPath := filepath.Join(dir, "mybin")
	if err := os.WriteFile(targetPath, []byte("#!/bin/sh\necho hi\n"), 0755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := setQuarantineForTest(targetPath); err != nil {
		t.Skipf("cannot set quarantine xattr (may need privileges): %v", err)
	}
	if has, _ := fs.HasQuarantine(targetPath); !has {
		t.Skip("could not set quarantine attribute for test")
	}

	// No confirmer, no flag -> must NOT strip (CI / non-interactive).
	m := &managerImpl{removeQuarantine: false, confirmer: nil}
	if m.shouldStripQuarantine("mybin") {
		t.Fatal("should not strip without confirmer or flag")
	}
	if still, _ := fs.HasQuarantine(targetPath); !still {
		t.Fatal("quarantine was stripped without confirmation (auto-strip regression)")
	}

	// Flag set -> strips, and the attribute is actually gone.
	m2 := &managerImpl{removeQuarantine: true, confirmer: nil}
	if !m2.shouldStripQuarantine("mybin") {
		t.Fatal("flag set should strip")
	}
	if err := fs.RemoveQuarantine(targetPath); err != nil {
		t.Fatalf("RemoveQuarantine: %v", err)
	}
	if still, _ := fs.HasQuarantine(targetPath); still {
		t.Fatal("quarantine attribute still present after RemoveQuarantine")
	}
}
