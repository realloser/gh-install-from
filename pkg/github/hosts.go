package github

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
)

// authHostsResponse mirrors the JSON shape of `gh auth status --json hosts`.
type authHostsResponse struct {
	Hosts map[string][]struct {
		Active bool   `json:"active"`
		Host   string `json:"host"`
		Login  string `json:"login"`
	} `json:"hosts"`
}

// authenticatedHosts runs `gh auth status --json hosts` and returns the list of
// hostnames the user is authenticated with, ordered with the active host first.
// An empty list (no output / parse failure) is a non-fatal signal that we
// should fall back to the default github.com behavior.
func authenticatedHosts() ([]string, error) {
	cmd := exec.Command("gh", "auth", "status", "--json", "hosts")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list authenticated hosts: %w", err)
	}

	var parsed authHostsResponse
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse gh auth status: %w", err)
	}

	hosts := make([]string, 0, len(parsed.Hosts))
	for h := range parsed.Hosts {
		hosts = append(hosts, h)
	}
	sort.Strings(hosts)

	// Promote the active host (if any) to the front so it is probed first.
	for _, h := range hosts {
		for _, entry := range parsed.Hosts[h] {
			if entry.Active {
				// Move h to front.
				rest := make([]string, 0, len(hosts))
				for _, x := range hosts {
					if x != h {
						rest = append(rest, x)
					}
				}
				return append([]string{h}, rest...), nil
			}
		}
	}

	return hosts, nil
}
