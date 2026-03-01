package shell

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// detectShell returns the shell name from environment
func detectShell() string {
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	if s := os.Getenv("SHELL"); s != "" {
		base := filepath.Base(s)
		base = strings.TrimSuffix(base, ".exe")
		base = strings.ToLower(base)
		if base == "bash" || base == "zsh" || base == "fish" || base == "pwsh" || base == "powershell" {
			if base == "pwsh" {
				return "powershell"
			}
			return base
		}
	}
	if s := os.Getenv("TERM_PROGRAM"); s != "" {
		s = strings.ToLower(s)
		if s == "apple_terminal" || s == "warp" {
			return "zsh" // default for macOS terminal
		}
	}
	return "bash" // default fallback
}
