package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update [owner/repo]",
	Short: "Update installed binaries",
	Long: `Update installed binaries from GitHub releases.
If no repository is specified, all installed binaries will be updated.`,
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	installDir := filepath.Join(homeDir, ".local", "bin")
	if len(args) == 0 {
		// Update all installed binaries
		entries, err := os.ReadDir(installDir)
		if err != nil {
			return fmt.Errorf("failed to read install directory: %w", err)
		}

		for _, entry := range entries {
			if entry.Type()&os.ModeSymlink != 0 {
				// TODO: Implement update logic for each binary
				fmt.Printf("Updating %s...\n", entry.Name())
			}
		}
	} else {
		// Update specific binary
		repo := args[0]
		fmt.Printf("Updating %s...\n", repo)
		// TODO: Implement update logic for specific binary
	}

	return nil
} 