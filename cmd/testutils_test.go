package cmd

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/cli/go-gh"
	"github.com/spf13/cobra"
)

// testServer creates a test HTTP server for mocking GitHub API responses
func testServer(t *testing.T, handler http.HandlerFunc) (string, func()) {
	t.Helper()
	server := httptest.NewServer(handler)
	return server.URL, server.Close
}

// captureOutput captures stdout and stderr during test execution
type captureOutput struct {
	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

func newCaptureOutput() *captureOutput {
	return &captureOutput{
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
	}
}

func (c *captureOutput) capture() func() {
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()

	os.Stdout = wOut
	os.Stderr = wErr

	go func() {
		io.Copy(c.stdout, rOut)
	}()
	go func() {
		io.Copy(c.stderr, rErr)
	}()

	return func() {
		wOut.Close()
		wErr.Close()
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}
}

// setupTestEnv creates a temporary directory and sets it up for testing
func setupTestEnv(t *testing.T) (string, func()) {
	t.Helper()

	// Create temp directory
	tmpDir := t.TempDir()

	// Create .local/bin directory
	binDir := filepath.Join(tmpDir, ".local", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Store original home directory
	oldHome := os.Getenv("HOME")

	// Set new home directory
	if err := os.Setenv("HOME", tmpDir); err != nil {
		t.Fatal(err)
	}

	// Return cleanup function
	return tmpDir, func() {
		os.Setenv("HOME", oldHome)
	}
}

// executeCommand executes a cobra command for testing
func executeCommand(t *testing.T, cmd *cobra.Command, args ...string) (string, string, error) {
	t.Helper()

	output := newCaptureOutput()
	cleanup := output.capture()
	defer cleanup()

	cmd.SetArgs(args)
	err := cmd.Execute()

	return output.stdout.String(), output.stderr.String(), err
}

// mockGitHubClient creates a mock GitHub client for testing
type mockGitHubClient struct {
	responses map[string]interface{}
	errors    map[string]error
}

func newMockGitHubClient() *mockGitHubClient {
	return &mockGitHubClient{
		responses: make(map[string]interface{}),
		errors:    make(map[string]error),
	}
}

func (m *mockGitHubClient) Get(path string, response interface{}) error {
	if err, ok := m.errors[path]; ok {
		return err
	}
	if resp, ok := m.responses[path]; ok {
		// Copy the mock response to the provided interface
		switch v := resp.(type) {
		case []byte:
			if err := json.Unmarshal(v, response); err != nil {
				return err
			}
		default:
			// Add more type handling as needed
			return fmt.Errorf("unsupported response type")
		}
	}
	return nil
}

// mockGH replaces the gh.RESTClient function with a mock for testing
func mockGH(t *testing.T, mock *mockGitHubClient) func() {
	original := gh.RESTClient
	gh.RESTClient = func(opts ...gh.ClientOption) (*gh.Client, error) {
		return mock, nil
	}
	return func() {
		gh.RESTClient = original
	}
} 