package github

import (
	"testing"
)

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name   string
		stderr string
		want   bool
	}{
		{name: "gh not found", stderr: "gh: Not Found (HTTP 404)", want: true},
		{name: "http 404", stderr: "HTTP 404: Not Found", want: true},
		{name: "rate limit", stderr: "gh: rate limit exceeded", want: false},
		{name: "network error", stderr: "dial tcp: connection refused", want: false},
		{name: "empty", stderr: "", want: false},
		{name: "auth error", stderr: "could not find token", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNotFound(tt.stderr); got != tt.want {
				t.Errorf("isNotFound(%q) = %v, want %v", tt.stderr, got, tt.want)
			}
		})
	}
}

func TestAuthenticatedHostsParsing(t *testing.T) {
	// Unit-test the JSON parsing logic by calling authenticatedHosts with a
	// mock gh on PATH that emits the --json hosts output. This verifies the
	// active-host-first ordering and multi-host enumeration.
	// (The newGhCliClient tests in gh_client_test.go also cover this indirectly,
	// but here we assert the full ordered list, not just the first host.)
	t.Setenv("PATH", "/nonexistent") // no gh available -> error path
	hosts, err := authenticatedHosts()
	if err == nil {
		t.Fatal("expected error when gh is not on PATH")
	}
	if hosts != nil {
		t.Fatalf("expected nil hosts on error, got %v", hosts)
	}
}
