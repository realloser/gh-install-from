# gh-install-from

A GitHub CLI extension to install binaries from GitHub releases. It automatically detects the appropriate binary for your OS and architecture, handles compressed files, and manages updates.

## Features

- 🔍 Automatic OS and architecture detection
- 📦 Support for compressed files (.tar.gz, .tgz, .zip)
- 🔄 Version management and updates
- 📊 Progress bar for downloads
- 🚀 Multi-platform build support

## Installation

```bash
gh extension install realloser/gh-install-from
```

## Usage

Install a binary from a GitHub repository:
```bash
gh install-from owner/repo
```

Example using ripgrep:
```bash
gh install-from BurntSushi/ripgrep
```

### Options

- `--version, -v`: Print version information
- `--no-version-check`: Disable automatic version check

## Development

### Prerequisites

- Go 1.21 or later
- GNU Make
- Git

### Building

Build for your current platform:
```bash
make build
```

Install to your local bin directory:
```bash
make install
```

### Testing and Linting

Run tests:
```bash
make test
```

Run linters:
```bash
make lint
```

### Release Build

Build for all supported platforms (with parallel execution):
```bash
# Build with 4 parallel jobs
make -j4 release

# Build with number of CPU cores
make -j$(nproc) release      # Linux
make -j$(sysctl -n hw.ncpu) release  # macOS
```

### Creating a Release

1. Create a new version tag:
```bash
make tag TAG=X.Y.Z
```

2. Push the tag to trigger the release workflow:
```bash
git push origin vX.Y.Z
```

The GitHub Actions workflow will automatically:
- Build binaries for all supported platforms
- Create a GitHub release
- Upload the binaries as release assets

## Supported Platforms

- macOS (amd64, arm64)
- Linux (386, amd64, arm, arm64)
- Windows (386, amd64)

## License

MIT License - see LICENSE file for details 