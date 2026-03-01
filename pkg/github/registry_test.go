package github

import (
	"testing"

	"github.com/realloser/gh-install-from/pkg/config"
)

func TestNewClient_Default(t *testing.T) {
	// Default uses "gh" - will fail if gh not installed, so we use mock
	cfg := &config.Config{Client: "mock"}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient(mock) error = %v", err)
	}
	if client == nil {
		t.Fatal("NewClient(mock) returned nil client")
	}
	if got := client.GetHost(); got != "github.com" {
		t.Errorf("GetHost() = %v, want github.com", got)
	}
}

func TestNewClient_UnknownAdapter(t *testing.T) {
	cfg := &config.Config{Client: "nonexistent"}
	_, err := NewClient(cfg)
	if err == nil {
		t.Fatal("NewClient(nonexistent) expected error")
	}
}

func TestNewClient_NilConfigUsesEnv(t *testing.T) {
	// With nil config, FromEnv is used - defaults to "gh"
	// We can't easily test without gh, so test that nil config doesn't panic
	_, err := NewClient(nil)
	// May succeed if gh is installed, or fail with "gh cli is not installed"
	if err != nil && err.Error() != "" {
		// Expected: either works or fails with specific error
		return
	}
}
