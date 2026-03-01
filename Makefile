BINARY_NAME=gh-install-from
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u '+%Y-%m-%d-%H%M UTC')

# Build flags
LDFLAGS=-ldflags="-X 'github.com/realloser/gh-install-from/pkg/version.Version=$(VERSION)' \
                  -X 'github.com/realloser/gh-install-from/pkg/version.Commit=$(COMMIT)' \
                  -X 'github.com/realloser/gh-install-from/pkg/version.Date=$(DATE)'"

# Supported platforms
PLATFORMS=darwin/amd64 darwin/arm64 linux/386 linux/amd64 linux/arm linux/arm64 windows/386 windows/amd64
BINARY_NAMES=$(foreach PLATFORM,$(PLATFORMS),$(BINARY_NAME)-$(word 1,$(subst /, ,$(PLATFORM)))-$(word 2,$(subst /, ,$(PLATFORM))))

# Generate platform-specific targets
PLATFORM_TARGETS=$(foreach PLATFORM,$(PLATFORMS),dist/$(BINARY_NAME)_$(word 1,$(subst /, ,$(PLATFORM)))_$(word 2,$(subst /, ,$(PLATFORM))).tar.gz)

# Tools
GOLANGCI_LINT = $(shell command -v golangci-lint 2> /dev/null)
GOSEC = $(shell command -v gosec 2> /dev/null)
GOIMPORTS = $(shell command -v goimports 2> /dev/null)

.PHONY: all
all: build

.PHONY: build
build:
	go build $(LDFLAGS) -o $(BINARY_NAME)

.PHONY: install
install: build
	mkdir -p ~/.local/bin
	cp $(BINARY_NAME) ~/.local/bin/

.PHONY: clean
clean:
	rm -rf dist/
	rm -f $(BINARY_NAME)
	go clean -cache -testcache -modcache

.PHONY: test
test:
	go test -v -race -cover ./...

.PHONY: lint
lint: lint-tools lint-golangci lint-go lint-sec lint-imports lint-fmt

.PHONY: lint-tools
lint-tools:
	@if [ -z "$(GOLANGCI_LINT)" ]; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@if [ -z "$(GOSEC)" ]; then \
		echo "Installing gosec..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
	fi
	@if [ -z "$(GOIMPORTS)" ]; then \
		echo "Installing goimports..."; \
		go install golang.org/x/tools/cmd/goimports@latest; \
	fi

.PHONY: lint-golangci
lint-golangci:
	golangci-lint run --timeout=5m

.PHONY: lint-go
lint-go:
	go vet ./...
	go mod verify

.PHONY: lint-sec
lint-sec:
	gosec -quiet ./...

.PHONY: lint-imports
lint-imports:
	goimports -w .

.PHONY: lint-fmt
lint-fmt:
	@echo "Checking formatting..."
	@test -z $(shell gofmt -l .)

.PHONY: fix
fix: lint-tools
	golangci-lint run --fix
	goimports -w .
	go fmt ./...

.PHONY: tidy-up
tidy-up:
	go mod tidy
	go mod verify

.PHONY: binaries
binaries: $(foreach BINARY,$(BINARY_NAMES),dist/$(BINARY))
	@echo "Build binaries: $^"

dist/$(BINARY_NAME)%: tidy-up
	GOOS=$(word 4,$(subst -, ,$(notdir $(basename $@)))) \
	GOARCH=$(word 5,$(subst -, ,$(notdir $(basename $@)))) \
	EXT=$(if $(findstring windows,$(word 4,$(subst -, ,$(notdir $(basename $@))))),.exe,)
	@echo "Building binary: $@$(EXT)"
	go build $(LDFLAGS) -o $@$(EXT)

# Pattern rule for creating tarballs
dist/%.tar.gz: dist/%/$(BINARY_NAME)%
	cd dist/$(notdir $(basename $@)) && \
	tar -czf ../$(notdir $@) \
		$(BINARY_NAME)$(if $(findstring windows,$(word 1,$(subst _, ,$(notdir $(basename $@))))),.exe,)

.PHONY: tag
tag:
	@if [ "$(TAG)" = "" ]; then echo "Please specify TAG=X.Y.Z"; exit 1; fi
	@if ! echo "$(TAG)" | grep -q "^[0-9]\+\.[0-9]\+\.[0-9]\+$$"; then echo "TAG must be in semver format X.Y.Z"; exit 1; fi
	git tag -a v$(TAG) -m "Release v$(TAG)"
	@echo "Created tag v$(TAG)"
	@echo "Now run: git push origin v$(TAG)"

# Print the number of jobs Make will run in parallel
.PHONY: jobs
jobs:
	@echo "Running with $(MAKEFLAGS) jobs"

.PHONY: verbose
verbose:
	@echo "Running with $(MAKEFLAGS) jobs"
	@echo "Platforms: $(PLATFORMS)"
	@echo "Platform targets: $(PLATFORM_TARGETS)"
	@echo "BINARY_NAMES: $(BINARY_NAMES)"

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build       - Build for current platform"
	@echo "  tidy-up     - Tidy up Go modules"
	@echo "  install     - Install to ~/.local/bin"
	@echo "  clean       - Clean build artifacts and caches"
	@echo "  test        - Run tests with race detection and coverage"
	@echo "  lint        - Run all linters"
	@echo "  fix         - Fix common linting issues automatically"
	@echo "  binaries    - Build release artifacts for all platforms"
	@echo "  tag         - Create a new version tag (TAG=X.Y.Z)"
	@echo "  jobs        - Show parallel job count"
	@echo ""
	@echo "Linting targets:"
	@echo "  lint-golangci  - Run golangci-lint"
	@echo "  lint-go        - Run go vet and verify modules"
	@echo "  lint-sec       - Run security checks"
	@echo "  lint-imports   - Fix imports formatting"
	@echo "  lint-fmt       - Check code formatting" 