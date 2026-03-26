// Mocked function signature
func DownloadAssetFunc(repo, tag, assetName, destPath string) error {
    // Function implementation goes here...
}

// Updated method
func (c *Client) DownloadAsset(repo, tag, assetName, destPath string) error {
    // Method implementation goes here...
}