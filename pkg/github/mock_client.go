package github

import (
	"fmt"
	"os"
	"path/filepath"
)

// MockClient is a configurable implementation of Client for testing and for the
// "mock" client adapter. It is registered under the "mock" name so consumers can
// select it via GH_INSTALL_FROM_CLIENT=mock or Config{Client: "mock"}.
type MockClient struct {
	Host             string
	LatestRelease    *Release
	LatestReleaseErr error
	DownloadAssetErr error
}

// GetLatestRelease implements Client.GetLatestRelease.
func (m *MockClient) GetLatestRelease(repo string) (*Release, error) {
	if m.LatestReleaseErr != nil {
		return nil, m.LatestReleaseErr
	}
	if m.LatestRelease == nil {
		return nil, fmt.Errorf("no release configured for %s", repo)
	}
	return m.LatestRelease, nil
}

// DownloadAsset implements Client.DownloadAsset.
func (m *MockClient) DownloadAsset(url, destPath string) error {
	if m.DownloadAssetErr != nil {
		return m.DownloadAssetErr
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(destPath, []byte("#!/bin/sh\necho 'mock binary'\n"), 0755)
}

// GetHost implements Client.GetHost.
func (m *MockClient) GetHost() string {
	if m.Host == "" {
		return "github.com"
	}
	return m.Host
}

// Ensure MockClient implements Client
var _ Client = (*MockClient)(nil)
