// Package shell provides ShellAdapter for PATH persistence
package shell

// ShellAdapter defines the interface for adding a bin path to shell configuration
type ShellAdapter interface {
	// AddToPATH appends binPath to shell rc file (idempotent)
	AddToPATH(binPath string) error
	// PostInitMessage returns the shell-specific message to print after AddToPATH succeeds
	PostInitMessage(binPath string) string
	// ApplyHint returns the shell-specific instruction to apply PATH changes (e.g. "run 'source ~/.bashrc'")
	ApplyHint() string
}
