package github

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
)

// Release represents a GitHub release
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// Asset represents a GitHub release asset
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Client defines the interface for GitHub API operations
type Client interface {
	// GetLatestRelease fetches the latest release for a repository
	GetLatestRelease(repo string) (*Release, error)
	// DownloadAsset downloads a release asset to a specified path
	DownloadAsset(url, destPath string) error
	// GetHost returns the GitHub host being used (e.g., "github.com")
	GetHost() string
}

// ghCliClient implements the Client interface using gh cli commands
type ghCliClient struct {
	host string
}

// newGhCliClient creates a new GitHub client using gh cli
func newGhCliClient() (Client, error) {
	// Check if gh is installed
	if _, err := exec.LookPath("gh"); err != nil {
		return nil, fmt.Errorf("gh cli is not installed: %w", err)
	}

	// Get the host from gh auth status, default to github.com
	host := "github.com"
	cmd := exec.Command("gh", "auth", "status")
	if output, err := cmd.Output(); err == nil {
		// Parse the first line which contains the host
		lines := strings.Split(string(output), "\n")
		if len(lines) > 0 {
			// The line format is typically: "Logged in to github.com as USERNAME"
			fields := strings.Fields(lines[0])
			if len(fields) >= 4 && fields[3] != "" {
				host = fields[3]
			}
		}
	}

	return &ghCliClient{host: host}, nil
}

// NewGhCliClient is a variable so it can be overridden in tests
var NewGhCliClient = newGhCliClient

// GetLatestRelease implements Client.GetLatestRelease using gh api
func (c *ghCliClient) GetLatestRelease(repo string) (*Release, error) {
	// Validate repo format
	if !isValidRepo(repo) {
		return nil, fmt.Errorf("invalid repository format: %s", repo)
	}

	cmd := exec.Command("gh", "api", fmt.Sprintf("repos/%s/releases/latest", repo))

	// Capture both stdout and stderr
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		if stderrStr != "" {
			return nil, fmt.Errorf("failed to get latest release: %w\nCommand output: %s", err, stderrStr)
		}
		return nil, fmt.Errorf("failed to get latest release: %w", err)
	}

	var release Release
	if err := json.Unmarshal([]byte(stdout.String()), &release); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &release, nil
}

// GetHost implements Client.GetHost
func (c *ghCliClient) GetHost() string {
	return c.host
}

// DownloadAsset implements Client.DownloadAsset using gh api
func (c *ghCliClient) DownloadAsset(downloadURL, destPath string) error {
	// Create the destination directory if it doesn't exist
	if err := os.MkdirAll(path.Dir(destPath), 0750); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Use gh api with --method GET and write output to file
	// Note: We don't use --raw flag as it's not supported in all gh versions
	cmd := exec.Command("gh", "api", downloadURL)

	// Capture stderr
	var stderr strings.Builder
	cmd.Stderr = &stderr

	// Open the destination file
	out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() {
		if cerr := out.Close(); cerr != nil {
			if err == nil {
				err = fmt.Errorf("failed to close output file: %w", cerr)
			}
		}
	}()

	// Set the output to the file
	cmd.Stdout = out

	// Run the command
	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		if stderrStr != "" {
			// Check for specific error messages and provide clearer errors
			if strings.Contains(stderrStr, "Not Found") {
				return fmt.Errorf("asset not found at %s", downloadURL)
			}
			if strings.Contains(stderrStr, "no such file") {
				return fmt.Errorf("failed to create file at %s: directory may not exist", destPath)
			}
			if strings.Contains(stderrStr, "permission denied") {
				return fmt.Errorf("permission denied when writing to %s", destPath)
			}
			if strings.Contains(stderrStr, "unknown flag") {
				return fmt.Errorf("incompatible gh CLI version. Please update gh CLI with 'brew upgrade gh' or your system's package manager")
			}
			return fmt.Errorf("failed to download asset: %w\nCommand output: %s", err, stderrStr)
		}
		return fmt.Errorf("failed to download asset: %w", err)
	}

	// Make the file executable
	if err := os.Chmod(destPath, 0755); err != nil {
		return fmt.Errorf("failed to make file executable: %w", err)
	}

	return nil
}

// isValidRepo checks if a repository string is in the correct format (owner/repo)
func isValidRepo(repo string) bool {
	// Simple check for now, could be more sophisticated
	return len(repo) > 0 && repo[0] != '/' && repo[len(repo)-1] != '/' && len(repo) < 256
}

// GetLatestRelease is a convenience function that creates a new client and gets the latest release
func GetLatestRelease(repo string) (*Release, error) {
	client, err := NewGhCliClient()
	if err != nil {
		return nil, err
	}
	return client.GetLatestRelease(repo)
}
