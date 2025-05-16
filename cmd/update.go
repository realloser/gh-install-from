/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

		var updateErrors []string
		for _, entry := range entries {
			if entry.Type()&os.ModeSymlink != 0 {
				target, err := os.Readlink(filepath.Join(installDir, entry.Name()))
				if err != nil {
					updateErrors = append(updateErrors, fmt.Sprintf("failed to read symlink %s: %v", entry.Name(), err))
					continue
				}

				// Extract repo from target path
				repoPath := extractRepoFromPath(target)
				if repoPath == "" {
					continue
				}

				fmt.Printf("Updating %s...\n", repoPath)
				if err := installLatestVersion(repoPath); err != nil {
					updateErrors = append(updateErrors, fmt.Sprintf("failed to update %s: %v", repoPath, err))
				}
			}
		}

		if len(updateErrors) > 0 {
			return fmt.Errorf("update completed with errors:\n%s", strings.Join(updateErrors, "\n"))
		}
	} else {
		// Update specific binary
		repo := args[0]
		fmt.Printf("Updating %s...\n", repo)
		if err := installLatestVersion(repo); err != nil {
			return fmt.Errorf("failed to update %s: %v", repo, err)
		}
	}

	return nil
}

func extractRepoFromPath(path string) string {
	// This is a simple implementation. You might want to store metadata
	// about installed binaries to make this more reliable.
	parts := strings.Split(filepath.Base(path), "-")
	if len(parts) >= 2 {
		return fmt.Sprintf("%s/%s", parts[0], parts[1])
	}
	return ""
}

func installLatestVersion(repo string) error {
	// Reuse the install command
	return runInstall(nil, []string{repo})
}
