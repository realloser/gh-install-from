# Contributing to gh-install-from

Thanks for your interest in contributing! This document covers development,
CI/CD, and the release process for `gh-install-from`. For user-facing usage
documentation, see the [README](README.md).

## Development

### Prerequisites

- Go 1.27 or later
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

This copies the binary to `~/.local/bin`. After first use, run `gh install-from init` to add `~/.gh-install-from/bin` to PATH for installed binaries.

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

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Run tests and linting (`make test && make lint`)
4. Commit your changes (`git commit -m 'feat: add amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request
