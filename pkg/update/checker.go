// Package update handles version checking and update management
package update

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/realloser/gh-install-from/pkg/github"
	"github.com/realloser/gh-install-from/pkg/log"
	"github.com/realloser/gh-install-from/pkg/semver"
	"github.com/realloser/gh-install-from/pkg/version"
)

// Checker handles version checking operations
type Checker struct {
	client     github.Client
	cacheDir   string
	repository string
}

// NewChecker creates a new version checker
func NewChecker(client github.Client, repository string) (*Checker, error) {
	cacheDir := getCacheDir()
	if err := os.MkdirAll(cacheDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &Checker{
		client:     client,
		cacheDir:   cacheDir,
		repository: repository,
	}, nil
}

// Check checks for available updates
func (c *Checker) Check(noVersionCheck bool) error {
	if noVersionCheck {
		return nil
	}

	file := filepath.Join(c.cacheDir, "last_check")
	if !c.shouldCheck(file) {
		return nil
	}

	if err := os.WriteFile(file, []byte(time.Now().Format(time.RFC3339)), 0600); err != nil {
		return fmt.Errorf("failed to write last check time: %w", err)
	}

	release, err := c.client.GetLatestRelease(c.repository)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	currentVersion := version.Version
	if currentVersion == "dev" {
		log.Debug("skipping version check for development version")
		return nil
	}

	latestVersion := release.TagName
	if latestVersion == "" {
		return fmt.Errorf("latest version is empty")
	}

	// Strip 'v' prefix if present for consistent comparison
	currentVersion = strings.TrimPrefix(currentVersion, "v")
	latestVersion = strings.TrimPrefix(latestVersion, "v")

	// Parse versions for comparison
	current, err := semver.Parse(currentVersion)
	if err != nil {
		return fmt.Errorf("failed to parse current version: %w", err)
	}

	latest, err := semver.Parse(latestVersion)
	if err != nil {
		return fmt.Errorf("failed to parse latest version: %w", err)
	}

	// Compare versions
	if latest.GT(current) {
		fmt.Printf("\nA new version of gh-install-from is available: %s → %s\n", currentVersion, latestVersion)
		fmt.Printf("Run 'gh extension upgrade gh-install-from' to upgrade\n\n")
	}

	return nil
}

func getCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return os.TempDir()
	}
	return filepath.Join(home, ".cache", "gh-install-from")
}

func (c *Checker) getLastCheckTime() (time.Time, error) {
	file := filepath.Join(c.cacheDir, "last_check")
	data, err := os.ReadFile(file)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, string(data))
}

func (c *Checker) shouldCheck(file string) bool {
	lastCheck, err := c.getLastCheckTime()
	if err != nil {
		return true // If we can't read the last check time, do check
	}
	return time.Since(lastCheck) >= 24*time.Hour
}
