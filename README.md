# gh-install-from

A GitHub CLI extension to install binaries from GitHub releases.

## Features

- Install binaries from GitHub releases
- List available versions of binaries
- Interactive version selection using terminal UI
- Update functionality for installed binaries
- Automatic OS/Architecture detection
- Symlink management for installed binaries

## Installation

```bash
gh extension install realloser/gh-install-from
```

## Usage

```bash
# List all commands
gh install-from --help

# Install a binary from a repository
gh install-from install owner/repo

# List available versions
gh install-from versions owner/repo

# Update all installed binaries
gh install-from update

# Update specific binary
gh install-from update owner/repo
```

## Development

Requirements:
- Go 1.21 or higher
- GitHub CLI

```bash
# Clone the repository
git clone https://github.com/realloser/gh-install-from
cd gh-install-from

# Build
go build

# Install locally for development
gh extension remove gh-install-from || true
gh extension install .
```

## License

MIT 