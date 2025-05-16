# gh-install-from

A GitHub CLI extension to install binaries from GitHub releases. This extension makes it easy to install and manage binaries from GitHub releases, automatically detecting the appropriate version for your operating system and architecture.

## Features

- 🚀 Install binaries from GitHub releases
- 📋 List available versions with interactive selection
- 🔄 Update functionality for all or specific binaries
- 🎯 Automatic OS/Architecture detection
- 📦 Smart binary name matching
- 🔗 Symlink management for installed binaries
- 💻 Beautiful terminal UI with progress bars

## Installation

```bash
gh extension install realloser/gh-install-from
```

## Usage

### Install a Binary

```bash
# Install the latest version
gh install-from install owner/repo

# Example: Install ripgrep
gh install-from install BurntSushi/ripgrep
```

### List Available Versions

```bash
# Browse and select from available versions
gh install-from versions owner/repo

# Example: List ripgrep versions
gh install-from versions BurntSushi/ripgrep
```

### Update Binaries

```bash
# Update all installed binaries
gh install-from update

# Update a specific binary
gh install-from update owner/repo

# Example: Update ripgrep
gh install-from update BurntSushi/ripgrep
```

## Binary Name Detection

The extension automatically detects appropriate binaries for your system by looking for common patterns in release asset names:

- OS patterns: darwin, macos, osx, linux, windows, win
- Architecture patterns: x86_64, amd64, x64, i386, arm64, aarch64, etc.
- Special handling for Windows .exe files

## Installation Directory

Binaries are installed to `~/.local/bin` by default. Make sure this directory is in your PATH:

```bash
# Add to your shell's configuration file (.bashrc, .zshrc, etc.)
export PATH="$HOME/.local/bin:$PATH"
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

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT 