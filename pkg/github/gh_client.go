package github

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
)

// Release represents a GitHub release
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// Asset represents a GitHub release asset
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Client defines the interface for GitHub API operations
type Client interface {
	// GetLatestRelease fetches the latest release for a repository
	GetLatestRelease(repo string) (*Release, error)
	// DownloadAsset downloads a release asset to a specified path
	DownloadAsset(url, destPath string) error
	// GetHost returns the GitHub host being used (e.g., "github.com")
	GetHost() string
}

// Ensure ghCliClient implements Client
var _ Client = (*ghCliClient)(nil)

// ghCliClient implements the Client interface using gh cli commands
type ghCliClient struct {
	host         string
	explicitHost bool // true when the host was pinned via --host (skip probing)
}

// newGhCliClient creates a new GitHub client using gh cli. The host is
// auto-resolved per-repo in GetLatestRelease (which accepts either owner/repo
// or a full https://host/owner/repo URL).
func newGhCliClient() (Client, error) {
	// Check if gh is installed
	if _, err := exec.LookPath("gh"); err != nil {
		return nil, fmt.Errorf("gh cli is not installed: %w\n\nInstall it from https://cli.github.com/ or via 'brew install gh'", err)
	}

	// Default to github.com; GetLatestRelease probes all authenticated hosts
	// per-repo and remembers the winner.
	host := "github.com"
	if hosts, err := authenticatedHosts(); err == nil && len(hosts) > 0 {
		host = hosts[0]
	}

	return &ghCliClient{host: host}, nil
}

// NewGhCliClient is a variable so it can be overridden in tests
var NewGhCliClient = newGhCliClient

// parseRepoArg accepts either "owner/repo" or a full URL
// "https://host/owner/repo" (optionally with a trailing path like /releases).
// It returns the host (empty for owner/repo) and the owner/repo string.
func parseRepoArg(arg string) (host, repo string, err error) {
	if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
		u, perr := url.Parse(arg)
		if perr != nil {
			return "", "", fmt.Errorf("invalid repository URL %q: %w", arg, perr)
		}
		// Path is like /owner/repo or /owner/repo/...
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
			return "", "", fmt.Errorf("invalid repository URL %q: expected https://<host>/<owner>/<repo>", arg)
		}
		return u.Host, parts[0] + "/" + parts[1], nil
	}
	// Plain owner/repo — no host, will be resolved by probing.
	if !isValidRepo(arg) {
		return "", "", fmt.Errorf("invalid repository format: %s (use owner/repo or https://host/owner/repo)", arg)
	}
	return "", arg, nil
}

// GetLatestRelease implements Client.GetLatestRelease using gh api. The repo
// argument may be either "owner/repo" (auto-resolved across authenticated
// hosts) or a full URL "https://host/owner/repo" (pins the host directly).
//
// For owner/repo: probes each authenticated host (with --hostname) until one
// returns a non-404 result. When the repo exists on exactly one host, that
// host is used. When it exists on multiple hosts, the install is aborted with
// a message listing each host's URL — the tool never silently guesses.
//
// For a full URL: the host is used directly, no probing.
//
// The winning host is stored on the client for subsequent DownloadAsset calls.
func (c *ghCliClient) GetLatestRelease(repoArg string) (*Release, error) {
	host, repo, err := parseRepoArg(repoArg)
	if err != nil {
		return nil, err
	}

	// A full URL pins the host: probe only that host, no multi-host resolution.
	if host != "" {
		release, ok, err := c.probeRelease(host, repo)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("repository %s not found on host %s", repo, host)
		}
		c.host = host
		return release, nil
	}

	hosts, err := authenticatedHosts()
	if err != nil || len(hosts) == 0 {
		// No enumerable hosts — fall back to the single current host.
		hosts = []string{c.host}
	}

	// Probe every host concurrently and collect matches. A 404 means "not on
	// this host"; any other error fails immediately. We must probe ALL hosts
	// (not short-circuit on first match) to detect ambiguity, so concurrency
	// keeps latency at one round-trip regardless of host count.
	type match struct {
		host    string
		release *Release
	}
	type result struct {
		host    string
		release *Release
		ok      bool
		err     error
	}
	results := make(chan result, len(hosts))
	var wg sync.WaitGroup
	for _, host := range hosts {
		wg.Add(1)
		go func(h string) {
			defer wg.Done()
			rel, ok, err := c.probeRelease(h, repo)
			results <- result{host: h, release: rel, ok: ok, err: err}
		}(host)
	}
	wg.Wait()
	close(results)

	var matches []match
	var firstErr error
	for r := range results {
		if r.err != nil {
			// non-404 real error — fail immediately (but collect the first).
			if firstErr == nil {
				firstErr = r.err
			}
			continue
		}
		if r.ok {
			matches = append(matches, match{host: r.host, release: r.release})
		}
	}
	if firstErr != nil && len(matches) == 0 {
		// A real error occurred and no host matched — surface it.
		return nil, firstErr
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("repository %s not found on any authenticated host (%s)\n\nCheck that:\n  - the repository name is correct (owner/repo)\n  - you are authenticated to the right host: 'gh auth status'\n  - the repository is not private or you have access", repo, strings.Join(hosts, ", "))
	case 1:
		c.host = matches[0].host // remember the winning host for download + metadata
		slog.Info("resolved to host", "host", c.host)
		return matches[0].release, nil
	default:
		// Ambiguous: the repo exists on multiple hosts. Abort with a
		// disambiguation message listing each URL rather than silently guessing.
		var urls []string
		for _, m := range matches {
			urls = append(urls, fmt.Sprintf("  - https://%s/%s", m.host, repo))
		}
		return nil, fmt.Errorf(
			"repository %s found on multiple authenticated hosts:\n%s\n"+
				"Specify the full URL to disambiguate, e.g.:\n  gh install-from https://%s/%s",
			repo, strings.Join(urls, "\n"), matches[0].host, repo)
	}
}

// probeRelease runs `gh api --hostname <host> repos/<repo>/releases/latest` and
// reports (release, true, nil) on success, (nil, false, nil) on a 404 (not
// found on this host), and (nil, false, err) on any other error.
func (c *ghCliClient) probeRelease(host, repo string) (*Release, bool, error) {
	slog.Info("checking host", "host", host)
	args := []string{"api"}
	if host != "" && host != "github.com" {
		args = append(args, "--hostname", host)
	}
	args = append(args, fmt.Sprintf("repos/%s/releases/latest", repo))

	cmd := exec.Command("gh", args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		if isNotFound(stderrStr) {
			slog.Debug("release not found on host", "host", host, "repo", repo)
			return nil, false, nil
		}
		if stderrStr != "" {
			return nil, false, fmt.Errorf("failed to get latest release from %s: %w\nCommand output: %s", host, err, stderrStr)
		}
		return nil, false, fmt.Errorf("failed to get latest release from %s: %w", host, err)
	}

	var release Release
	if err := json.Unmarshal([]byte(stdout.String()), &release); err != nil {
		return nil, false, fmt.Errorf("failed to decode response: %w", err)
	}

	slog.Info("found on host", "host", host, "tag", release.TagName)
	return &release, true, nil
}

// isNotFound reports whether gh stderr output indicates an HTTP 404.
func isNotFound(stderr string) bool {
	return strings.Contains(stderr, "Not Found") || strings.Contains(stderr, "HTTP 404")
}

// GetHost implements Client.GetHost
func (c *ghCliClient) GetHost() string {
	return c.host
}

// DownloadAsset implements Client.DownloadAsset using gh api
func (c *ghCliClient) DownloadAsset(downloadURL, destPath string) error {
	// Create the destination directory if it doesn't exist
	if err := os.MkdirAll(path.Dir(destPath), 0750); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Use gh api with --method GET and write output to file
	// Note: We don't use --raw flag as it's not supported in all gh versions.
	// Pass --hostname so the download uses the resolved host's auth (asset URLs
	// are host-specific).
	args := []string{"api"}
	if c.host != "" && c.host != "github.com" {
		args = append(args, "--hostname", c.host)
	}
	args = append(args, downloadURL)
	cmd := exec.Command("gh", args...)

	// Capture stderr
	var stderr strings.Builder
	cmd.Stderr = &stderr

	// Open the destination file
	out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() {
		if cerr := out.Close(); cerr != nil {
			if err == nil {
				err = fmt.Errorf("failed to close output file: %w", cerr)
			}
		}
	}()

	// Set the output to the file
	cmd.Stdout = out

	// Run the command
	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		if stderrStr != "" {
			// Check for specific error messages and provide clearer errors
			if strings.Contains(stderrStr, "Not Found") {
				return fmt.Errorf("asset not found at %s", downloadURL)
			}
			if strings.Contains(stderrStr, "no such file") {
				return fmt.Errorf("failed to create file at %s: directory may not exist", destPath)
			}
			if strings.Contains(stderrStr, "permission denied") {
				return fmt.Errorf("permission denied when writing to %s", destPath)
			}
			if strings.Contains(stderrStr, "unknown flag") {
				return fmt.Errorf("incompatible gh CLI version. Please update gh CLI with 'brew upgrade gh' or your system's package manager")
			}
			return fmt.Errorf("failed to download asset: %w\nCommand output: %s", err, stderrStr)
		}
		return fmt.Errorf("failed to download asset: %w", err)
	}

	// Make the file executable
	if err := os.Chmod(destPath, 0755); err != nil {
		return fmt.Errorf("failed to make file executable: %w", err)
	}

	return nil
}

// isValidRepo checks if a repository string is in the correct format (owner/repo)
func isValidRepo(repo string) bool {
	// Must be owner/repo: non-empty, contains a slash, doesn't start/end with one.
	return len(repo) > 0 && repo[0] != '/' && repo[len(repo)-1] != '/' && len(repo) < 256 && strings.Contains(repo, "/")
}

// GetLatestRelease is a convenience function that creates a new client and gets the latest release
func GetLatestRelease(repo string) (*Release, error) {
	client, err := NewGhCliClient()
	if err != nil {
		return nil, err
	}
	return client.GetLatestRelease(repo)
}
