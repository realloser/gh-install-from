package fs

import (
	"testing"

	"github.com/realloser/gh-install-from/pkg/config"
)

// TestNewOSService_UnknownAdapter runs on all platforms - tests error path
func TestNewOSService_UnknownAdapter(t *testing.T) {
	cfg := &config.Config{OS: "unknown-fake-os"}
	_, err := NewOSService(cfg)
	if err == nil {
		t.Error("NewOSService(unknown) expected error")
	}
}
