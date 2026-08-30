/*
Copyright © 2025 Martyn Messerli
*/
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	ghTerm "github.com/cli/go-gh/pkg/term"
)

// ttyConfirmer is a Confirmer that prompts on stdin. It returns false without
// prompting when stdin is not a TTY (e.g. CI), so non-interactive contexts
// never hang.
type ttyConfirmer struct{}

// NewTTYConfirmer returns a binary.Confirmer backed by an interactive stdin
// prompt. The concrete return is avoided to keep cmd from importing pkg/binary
// circularly; callers use it via the binary.Confirmer interface.
func NewTTYConfirmer() TTYConfirmer { return ttyConfirmer{} }

// TTYConfirmer is the cmd-side interface satisfied by ttyConfirmer. It matches
// binary.Confirmer (Confirm(prompt string) bool) so it can be passed where a
// binary.Confirmer is expected.
type TTYConfirmer interface {
	Confirm(prompt string) bool
}

// Confirm prints the prompt and reads a yes/no answer from stdin. Returns true
// only for "y" or "yes" (case-insensitive). If stdin is not a terminal, it
// returns false without prompting so CI never blocks.
func (ttyConfirmer) Confirm(prompt string) bool {
	if !ghTerm.IsTerminal(os.Stdin) {
		return false
	}
	fmt.Print(prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return false
	}
	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return answer == "y" || answer == "yes"
}
