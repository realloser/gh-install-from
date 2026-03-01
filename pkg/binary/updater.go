package binary

import (
	"log/slog"
	"runtime"
	"strings"
	"sync"

	"github.com/realloser/gh-install-from/pkg/github"
	"github.com/realloser/gh-install-from/pkg/semver"
)

// CheckUpdates checks which installed binaries have updates available (concurrent)
func CheckUpdates(installed []InstalledBinary, client github.Client, workers int) ([]UpdateCandidate, error) {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	if workers > len(installed) {
		workers = len(installed)
	}
	if workers < 1 {
		workers = 1
	}

	jobs := make(chan InstalledBinary, len(installed))
	results := make(chan UpdateCandidate, len(installed))

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for b := range jobs {
				if b.Repository == "" {
					continue
				}
				release, err := client.GetLatestRelease(b.Repository)
				if err != nil || release == nil {
					slog.Debug("failed to get latest release", "repo", b.Repository, "error", err)
					continue
				}
				latestVersion := strings.TrimPrefix(release.TagName, "v")
				installedVersion := strings.TrimPrefix(b.Version, "v")

				latest, err := semver.Parse(latestVersion)
				if err != nil {
					slog.Debug("failed to parse latest version", "repo", b.Repository, "version", latestVersion, "error", err)
					continue
				}
				cur, err := semver.Parse(installedVersion)
				if err != nil {
					slog.Debug("failed to parse installed version", "repo", b.Repository, "version", installedVersion, "error", err)
					continue
				}

				if latest.GT(cur) {
					results <- UpdateCandidate{
						InstalledBinary: b,
						LatestVersion:   release.TagName,
					}
				}
			}
		}()
	}

	go func() {
		for _, b := range installed {
			jobs <- b
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	candidates := make([]UpdateCandidate, 0, len(installed))
	for c := range results {
		candidates = append(candidates, c)
	}

	return candidates, nil
}
