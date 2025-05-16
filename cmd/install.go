package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cli/go-gh"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install [owner/repo]",
	Short: "Install a binary from a GitHub repository",
	Args:  cobra.ExactArgs(1),
	RunE:  runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	repo := args[0]
	client, err := gh.RESTClient(nil)
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Get latest release
	var release struct {
		Assets []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	err = client.Get(fmt.Sprintf("repos/%s/releases/latest", repo), &release)
	if err != nil {
		return fmt.Errorf("failed to get latest release: %w", err)
	}

	// Find matching binary for current OS/Arch
	osName := runtime.GOOS
	archName := runtime.GOARCH
	var matchingAsset struct {
		Name string
		URL  string
	}

	for _, asset := range release.Assets {
		if matchBinary(asset.Name, osName, archName) {
			matchingAsset.Name = asset.Name
			matchingAsset.URL = asset.BrowserDownloadURL
			break
		}
	}

	if matchingAsset.URL == "" {
		return fmt.Errorf("no matching binary found for %s/%s", osName, archName)
	}

	// Create installation directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	installDir := filepath.Join(homeDir, ".local", "bin")
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}

	// TODO: Implement download and installation with progress bar
	return nil
}

func matchBinary(name, os, arch string) bool {
	// TODO: Implement proper binary name matching
	return true
} 