/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cli/go-gh"
	"github.com/spf13/cobra"
	"github.com/realloser/gh-install-from/pkg/log"
	"github.com/realloser/gh-install-from/pkg/version"
)

var (
	versionFlag bool
	noVersionCheck bool
	verboseFlag bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gh-install-from",
	Short: "Install binaries from GitHub releases",
	Long: `gh-install-from is a GitHub CLI extension that helps you install
binaries from GitHub releases. It automatically detects the appropriate
binary for your OS and architecture, handles compressed files, and manages
updates.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize logging with verbose flag
		log.Init(verboseFlag)
		log.Debug("verbose mode enabled")
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if versionFlag {
			fmt.Println(version.DetailedInfo())
			return nil
		}

		if !noVersionCheck {
			if err := checkForUpdates(); err != nil {
				log.Warn("failed to check for updates", "error", err)
			}
		}

		return cmd.Help()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gh-install-from.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "V", false, "Enable verbose output")
	rootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "Print version information")
	rootCmd.Flags().BoolVar(&noVersionCheck, "no-version-check", false, "Disable version check")
}

func checkForUpdates() error {
	// Only check once per day
	if shouldSkipVersionCheck() {
		log.Debug("skipping version check - last check was less than 24h ago")
		return nil
	}

	log.Debug("checking for updates")
	client, err := gh.RESTClient(nil)
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}

	err = client.Get("repos/realloser/gh-install-from/releases/latest", &release)
	if err != nil {
		return fmt.Errorf("failed to get latest release: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentVersion := version.Info()

	log.Debug("version check",
		"current", currentVersion,
		"latest", latestVersion,
	)

	if currentVersion != "dev" && latestVersion > currentVersion {
		log.Info("new version available",
			"current", currentVersion,
			"latest", latestVersion,
		)
		fmt.Fprintf(os.Stderr, "\nA new version of gh-install-from is available: %s → %s\n", currentVersion, latestVersion)
		fmt.Fprintf(os.Stderr, "Run 'gh extension upgrade gh-install-from' to update\n\n")
	}

	// Update last check time
	updateLastCheckTime()
	return nil
}

func shouldSkipVersionCheck() bool {
	lastCheck := getLastCheckTime()
	if lastCheck.IsZero() {
		return false
	}
	return time.Since(lastCheck) < 24*time.Hour
}

func getLastCheckTime() time.Time {
	data, err := os.ReadFile(getVersionCheckFile())
	if err != nil {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, string(data))
	if err != nil {
		return time.Time{}
	}
	return t
}

func updateLastCheckTime() {
	file := getVersionCheckFile()
	os.MkdirAll(getCacheDir(), 0755)
	os.WriteFile(file, []byte(time.Now().Format(time.RFC3339)), 0644)
}

func getVersionCheckFile() string {
	return fmt.Sprintf("%s/last_version_check", getCacheDir())
}

func getCacheDir() string {
	if dir, err := os.UserCacheDir(); err == nil {
		return fmt.Sprintf("%s/gh-install-from", dir)
	}
	home, _ := os.UserHomeDir()
	return fmt.Sprintf("%s/.cache/gh-install-from", home)
}


