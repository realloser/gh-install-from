/*
Copyright © 2025 Martyn Messerli
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/realloser/gh-install-from/pkg/path"
	"github.com/spf13/cobra"
)

// doctorCmd checks PATH configuration
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check PATH configuration",
	Long: `Check if the gh-install-from bin directory is on your PATH.
Prints instructions if not configured.`,
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	pathMgr, err := path.New()
	if err != nil {
		return fmt.Errorf("failed to get path: %w", err)
	}

	binDir := pathMgr.GetBinDir()
	pathEnv := os.Getenv("PATH")
	if runtime.GOOS == "windows" && pathEnv == "" {
		pathEnv = os.Getenv("Path")
	}

	paths := filepath.SplitList(pathEnv)
	for _, p := range paths {
		if strings.TrimSpace(p) == binDir {
			fmt.Printf("PATH is correctly configured. %s is on PATH.\n", binDir)
			return nil
		}
	}

	fmt.Printf("PATH is not configured. Add the following to your shell config:\n\n")
	if runtime.GOOS == "windows" {
		fmt.Printf("  $env:Path = \"%s;\" + $env:Path\n\n", binDir)
		fmt.Printf("Or run 'gh install-from init' to add it automatically.\n")
	} else {
		fmt.Printf("  export PATH=\"%s:$PATH\"\n\n", binDir)
		fmt.Printf("Or run 'gh install-from init' to add it automatically.\n")
	}
	return nil
}
