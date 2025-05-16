/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/realloser/gh-install-from/pkg/github"
	"github.com/realloser/gh-install-from/pkg/log"
	"github.com/realloser/gh-install-from/pkg/version"
	"github.com/spf13/cobra"
)

var (
	verbose        bool
	noVersionCheck bool
	showVersion    bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gh-install-from",
	Short: "Install binaries from GitHub releases",
	Long: `A GitHub CLI extension to install binaries from GitHub releases.
It automatically detects the appropriate binary for your OS and architecture,
handles compressed files, and manages updates.`,
	SilenceUsage: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if verbose {
			log.Init(true)
		}
		if err := checkForUpdates(); err != nil {
			log.Debug("failed to check for updates:", err)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			fmt.Print(version.DetailedInfo())
			return nil
		}
		return cmd.Usage()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "V", false, "enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&noVersionCheck, "no-version-check", false, "disable automatic version check")
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "print version information")
}

func checkForUpdates() error {
	if noVersionCheck {
		return nil
	}

	file := filepath.Join(getCacheDir(), "last_check")
	if shouldCheck(file) {
		if err := os.MkdirAll(getCacheDir(), 0750); err != nil {
			return fmt.Errorf("failed to create cache directory: %w", err)
		}
		if err := os.WriteFile(file, []byte(time.Now().Format(time.RFC3339)), 0600); err != nil {
			return fmt.Errorf("failed to write last check time: %w", err)
		}

		// Check for updates
		client, err := github.NewGhCliClient()
		if err != nil {
			return fmt.Errorf("failed to create GitHub client: %w", err)
		}

		_, err = client.GetLatestRelease("realloser/gh-install-from")
		if err != nil {
			return fmt.Errorf("failed to check for updates: %w", err)
		}

		// Compare versions and notify if update available
		// ... rest of the function unchanged ...
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

func getLastCheckTime() (time.Time, error) {
	file := filepath.Join(getCacheDir(), "last_check")
	data, err := os.ReadFile(file)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, string(data))
}

func shouldCheck(file string) bool {
	lastCheck, err := getLastCheckTime()
	if err != nil {
		return true // If we can't read the last check time, do check
	}
	return time.Since(lastCheck) >= 24*time.Hour
}
