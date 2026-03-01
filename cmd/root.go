/*
Copyright © 2025 Martyn Messerli
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/realloser/gh-install-from/pkg/binary"
	"github.com/realloser/gh-install-from/pkg/github"
	"github.com/realloser/gh-install-from/pkg/log"
	"github.com/realloser/gh-install-from/pkg/update"
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
	Use:   "gh-install-from [owner/repo]",
	Short: "Install binaries from GitHub releases",
	Long: `A GitHub CLI extension to install binaries from GitHub releases.
It automatically detects the appropriate binary for your OS and architecture,
handles compressed files, and manages updates.

Examples:
  # Install a binary
  gh install-from cli/cli

  # Remove a binary
  gh install-from remove cli/cli

  # Update all installed binaries
  gh install-from update

  # List installed binaries
  gh install-from list`,
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if verbose {
			log.Init(true)
		}

		client, err := github.NewGhCliClient()
		if err != nil {
			log.Debug("failed to create GitHub client:", err)
			return
		}

		checker, err := update.NewChecker(client, "realloser/gh-install-from")
		if err != nil {
			log.Debug("failed to create update checker:", err)
			return
		}

		if err := checker.Check(noVersionCheck); err != nil {
			log.Debug("failed to check for updates:", err)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			fmt.Print(version.DetailedInfo())
			return nil
		}

		if len(args) == 0 {
			return cmd.Usage()
		}

		repo := args[0]
		log.Debug("installing binary", "repo", repo)

		client, err := github.NewGhCliClient()
		if err != nil {
			return fmt.Errorf("failed to create GitHub client: %w", err)
		}

		manager, err := binary.New(client)
		if err != nil {
			return fmt.Errorf("failed to create binary manager: %w", err)
		}

		if err := manager.Install(repo); err != nil {
			return fmt.Errorf("failed to install binary: %w", err)
		}

		return nil
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
