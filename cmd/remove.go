/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/realloser/gh-install-from/pkg/binary"
	"github.com/realloser/gh-install-from/pkg/github"
	"github.com/spf13/cobra"
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove [binary-name|owner/repo]",
	Short: "Remove an installed binary",
	Long: `Remove an installed binary and its metadata.
You can specify either the binary name or the repository name (owner/repo).
The binary will be removed from ~/.local/bin and its metadata will be cleaned up.

Examples:
  gh install-from remove ripgrep      # Remove by binary name
  gh install-from remove cli/cli      # Remove by repository name`,
	Args: cobra.ExactArgs(1),
	RunE: runRemove,
}

func init() {
	rootCmd.AddCommand(removeCmd)
}

func runRemove(cmd *cobra.Command, args []string) error {
	nameOrRepo := args[0]

	// Create a GitHub client
	client, err := github.NewGhCliClient()
	if err != nil {
		return err
	}

	// Create a binary manager
	manager, err := binary.New(client)
	if err != nil {
		return err
	}

	// Use the manager to remove the binary
	return manager.Remove(nameOrRepo)
}
