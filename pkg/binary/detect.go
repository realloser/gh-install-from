package binary

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"time"
)

// detectQuarantine runs the installed binary with a safe no-op flag and reports
// whether it looks like a macOS Gatekeeper/quarantine kill. It never blocks the
// install and never errors on "binary doesn't support the flag".
//
// The sequence tries --help, then -h, then --version. The first flag that lets
// the binary run to completion (exit 0, or exit non-zero because the binary
// executed its own argument-parsing code) means the binary is not blocked by
// Gatekeeper. Only a process killed by a signal before producing any output is
// classified as quarantined.
func detectQuarantine(binPath string) (quarantined bool, detail string) {
	for _, arg := range []string{"--help", "-h", "--version"} {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		cmd := exec.CommandContext(ctx, binPath, arg)
		var out, errBuf bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &errBuf
		err := cmd.Run()
		if ctx.Err() == context.DeadlineExceeded {
			// Hung past the timeout — a binary that blocks on --help is not a
			// quarantine kill. Stop here rather than trying the remaining flags.
			cancel()
			return false, ""
		}
		cancel()
		if err == nil {
			return false, "" // binary ran fine
		}
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			if isQuarantineKill(ee.ExitCode(), out.Len() > 0) {
				return true, errBuf.String()
			}
			// Non-zero exit but the binary ran (usage/flag error) => not quarantine.
			return false, ""
		}
		// Not an ExitError (e.g. "exec format error" for a text file, or a
		// missing binary) => the binary did not get far enough to be killed by
		// Gatekeeper; treat as not quarantined.
		return false, ""
	}
	return false, ""
}

// isQuarantineKill is the pure classifier for a Gatekeeper kill: the process
// was terminated by a signal (ExitCode() < 0) and produced no stdout before
// being killed. This is the strongest available signal that macOS killed the
// binary before it could run.
func isQuarantineKill(exitCode int, hadStdout bool) bool {
	return exitCode < 0 && !hadStdout
}
