# AGENTS.md

Guidance for AI agents working in this repository.

## What this is

`gh-install-from` is a `gh` CLI extension that installs binaries from GitHub releases. It detects the right asset for the user's OS/arch, extracts it from archives (or copies bare binaries), installs it via symlink (Unix) or `.cmd` shim (Windows), and tracks installs in a JSON metadata store. It supports GitHub Enterprise and private repos by reusing `gh`'s auth.

## Commands

```bash
make build          # build for current platform -> ./gh-install-from
make test           # go test -v -race -cover ./...
make lint           # golangci-lint, go vet, gosec, goimports, gofmt
make fix            # auto-fix lint issues
make help           # list all targets
```

Run a single test: `go test -v -race -run TestName ./pkg/binary/...`

**Before committing:** run `make build && make lint && make test`. Changes must be covered by unit tests. Use conventional commit messages (`feat:`, `fix:`, `docs:`, `test:`, `refactor:`).

## Architecture

**Adapter + registry pattern** — every external concern is behind an interface with a pluggable `map[string]factory` registry, selected at runtime from `config.Config` (reads `GH_INSTALL_FROM_*` env vars). The same shape repeats across `pkg/github` (`Client`), `pkg/metadata` (`MetadataStore`), `pkg/fs` (`OSService`), `pkg/shell` (`ShellAdapter`), `pkg/binary` (`Manager`). To add an adapter: define the factory, `RegisterXxx("name", fn)` in `init()`. Tests inject fakes via `NewWithDeps` or `GH_INSTALL_FROM_CLIENT=mock`.

**Install flow** (`pkg/binary/manager_impl.go` `Install`): `GetLatestRelease` → `findMatchingAsset` (OS/arch synonym matching, skips checksums) → download to temp → `processDownload` (content-detected: archive via `ArchiveProcessor`, bare binary via `BinaryProcessor`) → stage into `downloads/<owner>/<repo>/<tag>/` → symlink/shim into `bin/` → JSON metadata. `Update` re-runs `Install`.

**Multi-host resolution** (`pkg/github/gh_client.go`): enumerates authenticated hosts via `gh auth status --json hosts`, probes each concurrently with `--hostname`, aborts on ambiguous matches with a URL disambiguation message. Accepts `owner/repo` or full `https://host/owner/repo` URLs.

**Cross-platform** — build-tagged: `unix.go` (`//go:build !windows`) vs `windows.go` (`//go:build windows`); `darwin.go` (`//go:build darwin`) vs `darwin_other.go` (`//go:build !darwin`) for quarantine xattr handling.

**Logging** (`pkg/log`) — `slog` with a custom handler: INFO/WARN/ERROR render as clean `→ message (key=value)` lines to stderr; DEBUG stays structured and is verbose-only (`-V`).

## Managed layout

Root: `$GH_INSTALL_FROM_HOME` or `~/.gh-install-from`. Subdirs: `bin/` (symlinks/shims, must be on PATH), `downloads/<owner>/<repo>/<tag>/` (actual binaries), `metadata/` (one `<name>.json` per install).

## Conventions

- **Errors:** return, never panic. Wrap with `fmt.Errorf("...: %w", err)`.
- **Interfaces:** small `interface.go` alongside impl, with compile-time check (`var _ Iface = (*impl)(nil)`).
- **Tests:** `*_test.go` next to code, table-driven.
- **Banned imports:** `unsafe`, `syscall`. Max line length 120.
- **UI styling:** `pkg/ui/format.go` uses `lipgloss`; keep new output consistent.
- **Commits:** explicit `git add <files>` (never `git add .`), detailed messages explaining WHAT and WHY. Only commit files you changed.

## CI

PR checks: `go mod verify`, `go fmt`, golangci-lint, `make test`, cross-platform build (ubuntu/macos/windows), binary size ≤ 10MB, gosec. Go 1.27.
