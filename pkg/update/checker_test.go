package update

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/realloser/gh-install-from/pkg/github"
)

type mockClient struct {
	latestRelease    *github.Release
	latestReleaseErr error
}

func (m *mockClient) GetLatestRelease(repo string) (*github.Release, error) {
	return m.latestRelease, m.latestReleaseErr
}

func (m *mockClient) DownloadAsset(url, destPath string) error {
	return nil
}

func (m *mockClient) GetHost() string {
	return "github.com"
}

func TestChecker_Check(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	tests := []struct {
		name           string
		noVersionCheck bool
		mockClient     *mockClient
		setupLastCheck func(string)
		wantErr        bool
	}{
		{
			name:           "skip check when noVersionCheck is true",
			noVersionCheck: true,
			mockClient:     &mockClient{},
			wantErr:        false,
		},
		{
			name:           "check when last check is old",
			noVersionCheck: false,
			mockClient: &mockClient{
				latestRelease: &github.Release{
					TagName: "v2.0.0",
				},
			},
			setupLastCheck: func(cacheDir string) {
				file := filepath.Join(cacheDir, "last_check")
				oldTime := time.Now().Add(-25 * time.Hour)
				os.WriteFile(file, []byte(oldTime.Format(time.RFC3339)), 0600)
			},
			wantErr: false,
		},
		{
			name:           "skip check when last check is recent",
			noVersionCheck: false,
			mockClient:     &mockClient{},
			setupLastCheck: func(cacheDir string) {
				file := filepath.Join(cacheDir, "last_check")
				recentTime := time.Now().Add(-1 * time.Hour)
				os.WriteFile(file, []byte(recentTime.Format(time.RFC3339)), 0600)
			},
			wantErr: false,
		},
		{
			name:           "handle client error",
			noVersionCheck: false,
			mockClient: &mockClient{
				latestReleaseErr: os.ErrNotExist,
			},
			setupLastCheck: func(cacheDir string) {
				file := filepath.Join(cacheDir, "last_check")
				oldTime := time.Now().Add(-25 * time.Hour)
				os.WriteFile(file, []byte(oldTime.Format(time.RFC3339)), 0600)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker, err := NewChecker(tt.mockClient, "test/repo")
			if err != nil {
				t.Fatalf("NewChecker() error = %v", err)
			}

			if tt.setupLastCheck != nil {
				tt.setupLastCheck(checker.cacheDir)
			}

			err = checker.Check(tt.noVersionCheck)
			if (err != nil) != tt.wantErr {
				t.Errorf("Check() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
