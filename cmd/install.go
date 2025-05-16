/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/realloser/gh-install-from/pkg/binary"
	"github.com/realloser/gh-install-from/pkg/github"
	"github.com/realloser/gh-install-from/pkg/log"
	"github.com/spf13/cobra"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install [repo]",
	Short: "Install a binary from a GitHub repository",
	Long: `Install a binary from a GitHub repository's latest release.
The binary will be installed to ~/.local/bin.`,
	Args: cobra.ExactArgs(1),
	RunE: runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
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
}
