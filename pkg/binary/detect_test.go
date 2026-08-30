package binary

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// writeScript creates an executable shell script at path returning path.
func writeScript(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body), 0755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestIsQuarantineKill(t *testing.T) {
	tests := []struct {
		name      string
		exitCode  int
		hadStdout bool
		want      bool
	}{
		{"killed by signal, no output", -1, false, true},
		{"killed by signal with output", -1, true, false},
		{"exit 0 (ran fine)", 0, true, false},
		{"exit 2 usage error, no output", 2, false, false},
		{"exit 2 usage error, with output", 2, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isQuarantineKill(tt.exitCode, tt.hadStdout); got != tt.want {
				t.Errorf("isQuarantineKill(%d, %v) = %v, want %v", tt.exitCode, tt.hadStdout, got, tt.want)
			}
		})
	}
}

func TestDetectQuarantine(t *testing.T) {
	// Each case is a fake binary that mimics an exit mode. Real Gatekeeper
	// kills aren't available in unit tests, so we simulate them.
	tests := []struct {
		name string
		body string
		want bool
	}{
		{
			name: "exits 0 (ran fine, not quarantined)",
			body: "exit 0\n",
			want: false,
		},
		{
			name: "prints usage and exits 2 (ran its own code, not quarantined)",
			body: "echo 'flag provided but not defined: -help'\nexit 2\n",
			want: false,
		},
		{
			name: "self-SIGKILLs (simulated Gatekeeper kill, quarantined)",
			body: "kill -9 $$\n",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip the SIGKILL simulation on Windows (no /bin/sh, and the
			// platform is irrelevant to darwin quarantine anyway).
			if runtime.GOOS == "windows" {
				t.Skip("shell-script fake binaries not portable to windows")
			}
			bin := writeScript(t, t.TempDir(), "fakebin", tt.body)
			got, _ := detectQuarantine(bin)
			if got != tt.want {
				t.Errorf("detectQuarantine(%s) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestDetectQuarantine_HangsIsNotQuarantine(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-script fake binaries not portable to windows")
	}
	// A binary that sleeps past the context timeout must be classified as NOT
	// quarantined (hung, not a Gatekeeper kill).
	bin := writeScript(t, t.TempDir(), "hungbin", "sleep 30\n")
	got, _ := detectQuarantine(bin)
	if got {
		t.Error("detectQuarantine() = true for a hung binary, want false")
	}
}
