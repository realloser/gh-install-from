/*
Copyright © 2025 Martyn Messerli
*/
package cmd

import (
	"fmt"

	"github.com/realloser/gh-install-from/pkg/path"
	"github.com/realloser/gh-install-from/pkg/shell"
	"github.com/spf13/cobra"
)

// initCmd adds the gh-install-from bin directory to the shell PATH
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Add bin directory to PATH",
	Long: `Add the gh-install-from bin directory to your shell configuration.
Detects your shell and appends the PATH export to the appropriate rc file.
Idempotent - safe to run multiple times.`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	pathMgr, err := path.New()
	if err != nil {
		return fmt.Errorf("failed to get path: %w", err)
	}

	adapter, err := shell.NewShellAdapter(nil)
	if err != nil {
		return fmt.Errorf("failed to detect shell: %w", err)
	}

	binPath := pathMgr.GetBinDir()
	if err := adapter.AddToPATH(binPath); err != nil {
		return fmt.Errorf("failed to add to PATH: %w", err)
	}

	fmt.Print(adapter.PostInitMessage(binPath))
	return nil
}
