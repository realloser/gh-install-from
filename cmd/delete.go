/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/realloser/gh-install-from/pkg/binary"
	"github.com/realloser/gh-install-from/pkg/github"
	"github.com/spf13/cobra"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete [binary-name|owner/repo]",
	Short: "Delete an installed binary",
	Long: `Delete an installed binary and its metadata.
You can specify either the binary name or the repository name (owner/repo).
The binary will be removed from ~/.local/bin and its metadata will be cleaned up.

Examples:
  gh install-from delete ripgrep      # Delete by binary name
  gh install-from delete cli/cli      # Delete by repository name`,
	Args: cobra.ExactArgs(1),
	RunE: runDelete,
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}

func runDelete(cmd *cobra.Command, args []string) error {
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

	// Use the manager to delete the binary
	return manager.Delete(nameOrRepo)
}
