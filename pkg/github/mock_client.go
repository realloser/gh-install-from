package github

// MockClient implements Client interface for testing
type MockClient struct {
	GetLatestReleaseFunc func(repo string) (*Release, error)
	DownloadAssetFunc    func(url, destPath string) error
	GetHostFunc          func() string
}

// GetLatestRelease implements Client.GetLatestRelease
func (m *MockClient) GetLatestRelease(repo string) (*Release, error) {
	if m.GetLatestReleaseFunc != nil {
		return m.GetLatestReleaseFunc(repo)
	}
	return nil, nil
}

// DownloadAsset implements Client.DownloadAsset
func (m *MockClient) DownloadAsset(url, destPath string) error {
	if m.DownloadAssetFunc != nil {
		return m.DownloadAssetFunc(url, destPath)
	}
	return nil
}

// GetHost implements Client.GetHost
func (m *MockClient) GetHost() string {
	if m.GetHostFunc != nil {
		return m.GetHostFunc()
	}
	return "github.com"
}
