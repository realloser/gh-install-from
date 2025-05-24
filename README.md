# gh-install-from

A GitHub CLI extension to install binaries from GitHub releases. It automatically detects the appropriate binary for your OS and architecture, handles compressed files, and manages updates.

> Maintained by [Martyn Messerli](https://github.com/realloser)

## ⚠️ Security Notice

This tool helps you download and install binaries from GitHub releases. Please note:

- **No Binary Verification**: While this tool itself undergoes security scanning, it **does not** verify the security or authenticity of the binaries you install
- **Trust Required**: You should only install binaries from repositories and authors you trust
- **Your Responsibility**: Always verify the source and reputation of repositories before installing their binaries
- **Recommended Practices**:
  - Check the repository's security practices
  - Verify release signatures if available
  - Review the repository's security advisories
  - Consider using package managers for well-known software

## Why Use This Tool?

### For Individual Users
- 🔍 **Automatic Detection**: Automatically finds the right binary for your system
- 📦 **Compression Support**: Handles .tar.gz, .tgz, and .zip files
- 🔄 **Version Management**: Easy updates and version tracking
- 📊 **User Experience**: Progress bars and detailed logging
- 🚀 **Cross-Platform**: Works on macOS, Linux, and Windows

### For Enterprises
- 🔒 **GitHub Enterprise Support**: Works seamlessly with [GitHub Enterprise](https://github.com/enterprise)
- 📝 **Audit Trail**: Maintains detailed installation records
- 🔐 **Security**: Implements enterprise-grade security measures
- 🏢 **Internal Tools**: Perfect for distributing internal tools via GitHub releases
- 📋 **Compliance**: Supports enterprise security policies and version control

### Why Not Use Package Managers?
While package managers like [Homebrew](https://brew.sh), [winget](https://learn.microsoft.com/en-us/windows/package-manager/winget/), or [apt](https://wiki.debian.org/Apt) are great for public software, they have limitations:
- Internal tools can't be published to public repositories
- Enterprise tools often can't be shared publicly
- Custom builds need to be distributed securely
- Version control is more complex
- Approval processes can be lengthy

`gh-install-from` solves these issues by:
- Installing directly from GitHub releases
- Supporting private repositories
- Working with GitHub Enterprise
- Maintaining version history
- Providing audit trails

## Features

- 🔍 Automatic OS and architecture detection
- 📦 Support for compressed files (.tar.gz, .tgz, .zip)
- 🔄 Version management and updates
- 📊 Progress bar for downloads
- 🚀 Multi-platform build support
- 🔒 Security scanning of gh-install-from itself
- 🛠️ Parallel builds for faster releases
- 📝 Detailed logging with verbose mode
- 🔑 Uses GitHub CLI's authentication

## Prerequisites

- [GitHub CLI](https://cli.github.com/) (`gh`) installed and authenticated
- Go 1.21 or later (for development only)

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
- `--verbose, -V`: Enable verbose output with detailed logging

### Verbose Mode

When using the `--verbose` flag, the tool will output detailed information about:
- Binary detection and selection
- Download progress and file operations
- Version checking and updates
- Installation paths and file operations

Example with verbose output:
```bash
gh install-from --verbose BurntSushi/ripgrep
```

## Development

### Prerequisites

- Go 1.21 or later
- GNU Make
- Git

Optional tools (automatically installed when needed):
- [golangci-lint](https://golangci-lint.run/)
- [gosec](https://github.com/securego/gosec)
- [goimports](https://pkg.go.dev/golang.org/x/tools/cmd/goimports)

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

Run tests (with race detection and coverage):
```bash
make test
```

Run all linters:
```bash
make lint
```

Available linting commands:
```bash
make lint-golangci  # Run comprehensive linting
make lint-go        # Run go vet and verify modules
make lint-sec       # Run security checks
make lint-imports   # Fix imports formatting
make lint-fmt       # Check code formatting
```

Fix common linting issues automatically:
```bash
make fix
```

See all available make targets:
```bash
make help
```

### Release Build

Build for all supported platforms with parallel execution:
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
- Run comprehensive tests and linting
- Build binaries for all supported platforms
- Generate SHA256 checksums
- Create a GitHub release
- Upload the binaries and checksums
- Generate release notes

## CI/CD

### Security Measures

The following security measures apply to the `gh-install-from` tool itself:

1. **Static Analysis**:
   - Code security scanning with [gosec](https://github.com/securego/gosec)
   - Dependency vulnerability checking with [nancy](https://github.com/sonatype-nexus-community/nancy)
   - Regular automated security updates

2. **Build Security**:
   - Reproducible builds
   - SHA256 checksums for verification
   - Automated binary size limits

3. **Runtime Security**:
   - Minimal required permissions
   - Safe archive extraction
   - Proper error handling

Note: These security measures only apply to the `gh-install-from` tool itself, not to the binaries you install using it.

### Pull Request Checks

All pull requests undergo automated checks:
- Code validation (formatting, linting)
- Cross-platform builds (Linux, macOS, Windows)
- Binary size verification (10MB limit)
- Security scanning (gosec, nancy)
- Dependency verification
- Test coverage

### Release Process

Releases are automated and triggered by version tags:
- Comprehensive validation
- Parallel multi-platform builds
- Checksum generation
- Release notes generation
- Binary uploads

## Supported Platforms

- macOS (amd64, arm64)
- Linux (386, amd64, arm, arm64)
- Windows (386, amd64)

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Run tests and linting (`make test && make lint`)
4. Commit your changes (`git commit -m 'feat: add amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

## License

MIT License - see LICENSE file for details 