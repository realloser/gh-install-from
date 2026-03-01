package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/realloser/gh-install-from/pkg/log"
	"github.com/realloser/gh-install-from/pkg/metadata"
)

func TestRunRemove(t *testing.T) {
	log.Init(false)

	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	os.Setenv("GH_INSTALL_FROM_HOME", filepath.Join(tmpHome, ".gh-install-from"))
	t.Cleanup(func() {
		os.Unsetenv("HOME")
		os.Unsetenv("GH_INSTALL_FROM_HOME")
	})

	rootDir := filepath.Join(tmpHome, ".gh-install-from")
	binDir := filepath.Join(rootDir, "bin")
	metaDir := filepath.Join(rootDir, "metadata")
	downloadsDir := filepath.Join(rootDir, "downloads", "test", "repo", "v1.0.0")

	if err := os.MkdirAll(binDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(metaDir, 0750); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		binaryName string
		setup      func(t *testing.T)
		wantErr    bool
	}{
		{
			name:       "delete existing binary with metadata",
			binaryName: "test-binary",
			setup: func(t *testing.T) {
				if err := os.MkdirAll(downloadsDir, 0750); err != nil {
					t.Fatal(err)
				}
				targetPath := filepath.Join(downloadsDir, "test-binary")
				if err := os.WriteFile(targetPath, []byte("test"), 0755); err != nil {
					t.Fatal(err)
				}
				binPath := filepath.Join(binDir, "test-binary")
				if runtime.GOOS != "windows" {
					if err := os.Symlink(targetPath, binPath); err != nil {
						t.Fatal(err)
					}
				} else {
					if err := os.WriteFile(binPath, []byte("test"), 0755); err != nil {
						t.Fatal(err)
					}
				}
				store, _ := metadata.NewStore(nil)
				store.Store(&metadata.BinaryMetadata{
					GHHost:     "github.com",
					Repository: "test/repo",
					Version:    "v1.0.0",
					BinaryPath: binPath,
				})
			},
			wantErr: false,
		},
		{
			name:       "delete non-existent binary",
			binaryName: "nonexistent",
			setup:      func(t *testing.T) {},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(t)

			err := runRemove(nil, []string{tt.binaryName})
			if (err != nil) != tt.wantErr {
				t.Errorf("runRemove() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				binPath := filepath.Join(binDir, tt.binaryName)
				if _, err := os.Lstat(binPath); !os.IsNotExist(err) {
					t.Error("binary/symlink still exists after deletion")
				}
			}
		})
	}
}
