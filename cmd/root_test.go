package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/realloser/gh-install-from/pkg/version"
)

func TestRootCmd(t *testing.T) {
	// Store original values
	origVersion := version.Version
	origCommit := version.Commit
	origDate := version.Date
	origStdout := os.Stdout
	origStderr := os.Stderr

	// Restore original values after test
	defer func() {
		version.Version = origVersion
		version.Commit = origCommit
		version.Date = origDate
		os.Stdout = origStdout
		os.Stderr = origStderr
	}()

	// Set test values
	version.Version = "1.0.0"
	version.Commit = "abc123"
	version.Date = "2024-03-14"

	tests := []struct {
		name        string
		args        []string
		wantOutput  string
		wantErr     bool
	}{
		{
			name: "version flag",
			args: []string{"--version"},
			wantOutput: "Version: 1.0.0\nCommit: abc123\nBuild Date: 2024-03-14",
			wantErr: false,
		},
		{
			name: "version short flag",
			args: []string{"-v"},
			wantOutput: "Version: 1.0.0\nCommit: abc123\nBuild Date: 2024-03-14",
			wantErr: false,
		},
		{
			name: "no flags shows help",
			args: []string{},
			wantOutput: "Usage:",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create pipes to capture stdout and stderr
			rOut, wOut, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			rErr, wErr, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}

			os.Stdout = wOut
			os.Stderr = wErr

			// Reset command for each test
			rootCmd.SetArgs(tt.args)

			// Create channels to receive the output
			outC := make(chan string)
			errC := make(chan string)

			// Copy stdout
			go func() {
				var buf bytes.Buffer
				io.Copy(&buf, rOut)
				outC <- buf.String()
			}()

			// Copy stderr
			go func() {
				var buf bytes.Buffer
				io.Copy(&buf, rErr)
				errC <- buf.String()
			}()

			err = rootCmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Close the write ends of the pipes
			wOut.Close()
			wErr.Close()

			// Get the captured output
			stdout := <-outC
			stderr := <-errC

			// Close the read ends of the pipes
			rOut.Close()
			rErr.Close()

			// Check if the expected output is in either stdout or stderr
			output := stdout + stderr
			if !strings.Contains(output, tt.wantOutput) {
				t.Errorf("Execute() output = %q, want %q", output, tt.wantOutput)
			}
		})
	}
} 