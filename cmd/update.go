/*
Copyright © 2025 Martyn Messerli
*/
package cmd

import (
	"fmt"

	"github.com/realloser/gh-install-from/pkg/binary"
	"github.com/realloser/gh-install-from/pkg/github"
	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update [owner/repo]",
	Short: "Update installed binaries",
	Long: `Update installed binaries from GitHub releases.
If no repository is specified, all installed binaries will be updated.`,
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// updateCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// updateCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	client, err := github.NewGhCliClient()
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	manager, err := binary.New(client)
	if err != nil {
		return fmt.Errorf("failed to create binary manager: %w", err)
	}

	if len(args) == 0 {
		return manager.UpdateAll()
	}

	return manager.Update(args[0])
}
