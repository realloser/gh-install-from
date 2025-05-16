package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "gh-install-from",
	Short: "Install binaries from GitHub releases",
	Long: `A GitHub CLI extension to install binaries from GitHub releases.
It provides functionality to install, update, and manage binaries from GitHub releases.
Automatically handles OS/Architecture detection and binary management.`,
}

func init() {
	// Add subcommands here
} 