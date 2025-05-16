/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cli/go-gh"
	"github.com/spf13/cobra"
)

// downloadMsg represents a download progress message
type downloadMsg struct {
	progress float64
}

// downloadModel represents the download progress UI model
type downloadModel struct {
	progress progress.Model
	err      error
	done     bool
}

func (m downloadModel) Init() tea.Cmd {
	return nil
}

func (m downloadModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m, tea.Quit
	case downloadMsg:
		var cmd tea.Cmd
		m.progress.SetPercent(msg.progress)
		if msg.progress >= 1.0 {
			m.done = true
			return m, tea.Quit
		}
		return m, cmd
	case error:
		m.err = msg
		return m, tea.Quit
	}
	return m, nil
}

func (m downloadModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}
	if m.done {
		return "Download complete!\n"
	}
	return fmt.Sprintf("\nDownloading... %s\n", m.progress.View())
}

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install [owner/repo]",
	Short: "Install a binary from a GitHub repository",
	Long: `Install a binary from a GitHub repository's releases.
The command will automatically detect the appropriate binary for your OS and architecture.`,
	Args: cobra.ExactArgs(1),
	RunE: runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// installCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// installCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
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

	// Create a temporary file for download
	tmpFile, err := os.CreateTemp("", "gh-install-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Initialize progress bar
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)

	model := downloadModel{
		progress: p,
	}

	// Start download program
	prog := tea.NewProgram(&model)
	go func() {
		resp, err := http.Get(matchingAsset.URL)
		if err != nil {
			prog.Send(fmt.Errorf("failed to download binary: %w", err))
			return
		}
		defer resp.Body.Close()

		size := resp.ContentLength
		var downloaded int64

		for {
			buffer := make([]byte, 32*1024)
			n, err := resp.Body.Read(buffer)
			if n > 0 {
				_, writeErr := tmpFile.Write(buffer[:n])
				if writeErr != nil {
					prog.Send(fmt.Errorf("failed to write to temporary file: %w", writeErr))
					return
				}
				downloaded += int64(n)
				prog.Send(downloadMsg{progress: float64(downloaded) / float64(size)})
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				prog.Send(fmt.Errorf("error during download: %w", err))
				return
			}
		}
	}()

	// Wait for download to complete
	if _, err := prog.Run(); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Close the temporary file
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Create the final binary path
	binaryName := filepath.Base(matchingAsset.Name)
	if !strings.Contains(binaryName, ".") {
		// If no extension, use the repo name
		parts := strings.Split(args[0], "/")
		binaryName = parts[len(parts)-1]
	}
	binaryPath := filepath.Join(installDir, binaryName)

	// Make the temporary file executable
	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	// Move the binary to the installation directory
	if err := os.Rename(tmpFile.Name(), binaryPath); err != nil {
		return fmt.Errorf("failed to move binary to installation directory: %w", err)
	}

	fmt.Printf("Successfully installed %s to %s\n", binaryName, binaryPath)
	return nil
}

// matchBinary checks if the binary name matches the current OS and architecture
func matchBinary(name, os, arch string) bool {
	name = strings.ToLower(name)
	os = strings.ToLower(os)
	arch = strings.ToLower(arch)

	// Map common architecture names
	archMap := map[string][]string{
		"amd64": {"x86_64", "x64", "amd64"},
		"386":   {"x86", "i386", "i686", "32"},
		"arm64": {"aarch64", "arm64"},
		"arm":   {"armv7", "armhf", "arm"},
	}

	// Map common OS names
	osMap := map[string][]string{
		"darwin":  {"darwin", "macos", "osx"},
		"linux":   {"linux", "gnu"},
		"windows": {"windows", "win"},
	}

	// Check if the binary name contains both OS and architecture
	matchesOS := false
	for _, variant := range osMap[os] {
		if strings.Contains(name, variant) {
			matchesOS = true
			break
		}
	}

	matchesArch := false
	for _, variant := range archMap[arch] {
		if strings.Contains(name, variant) {
			matchesArch = true
			break
		}
	}

	// Special case for Windows executables
	if os == "windows" && strings.HasSuffix(name, ".exe") {
		matchesOS = true
	}

	return matchesOS && matchesArch
}
