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
# Windows binaries get an .exe suffix (required by `gh extension install`)
BINARY_NAMES=$(foreach PLATFORM,$(PLATFORMS),$(BINARY_NAME)-$(word 1,$(subst /, ,$(PLATFORM)))-$(word 2,$(subst /, ,$(PLATFORM)))$(if $(filter windows,$(word 1,$(subst /, ,$(PLATFORM)))),.exe))

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
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0; \
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
	gosec -quiet -exclude G302,G306,G301,G304,G204,G115,G110,G104 ./...

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

# $* is "<os>-<arch>[.exe]". GOOS/GOARCH must be on the same shell line as
# `go build`, otherwise they are not exported and the host platform is built.
dist/$(BINARY_NAME)-%: tidy-up
	@echo "Building binary: $@"
	CGO_ENABLED=0 GOOS=$(word 1,$(subst -, ,$*)) GOARCH=$(basename $(word 2,$(subst -, ,$*))) \
		go build $(LDFLAGS) -o $@

# Fail if any binary in dist/ is not built for its advertised OS/arch
.PHONY: verify-binaries
verify-binaries:
	@cd dist && fail=0; \
	check() { file "$$1" | grep -q "$$2" || { echo "::error::$$1 is not $$2"; file "$$1"; fail=1; }; }; \
	check $(BINARY_NAME)-darwin-amd64 "Mach-O 64-bit.*x86_64"; \
	check $(BINARY_NAME)-darwin-arm64 "Mach-O 64-bit.*arm64"; \
	check $(BINARY_NAME)-linux-386 "ELF 32-bit.*80386"; \
	check $(BINARY_NAME)-linux-amd64 "ELF 64-bit.*x86-64"; \
	check $(BINARY_NAME)-linux-arm "ELF 32-bit.*ARM"; \
	check $(BINARY_NAME)-linux-arm64 "ELF 64-bit.*ARM aarch64"; \
	check $(BINARY_NAME)-windows-386.exe "PE32 executable.*80386"; \
	check $(BINARY_NAME)-windows-amd64.exe "PE32+ executable.*x86-64"; \
	exit $$fail

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
	@echo "  verify-binaries - Check dist/ binaries match their OS/arch"
	@echo "  tag         - Create a new version tag (TAG=X.Y.Z)"
	@echo "  jobs        - Show parallel job count"
	@echo ""
	@echo "Linting targets:"
	@echo "  lint-golangci  - Run golangci-lint"
	@echo "  lint-go        - Run go vet and verify modules"
	@echo "  lint-sec       - Run security checks"
	@echo "  lint-imports   - Fix imports formatting"
	@echo "  lint-fmt       - Check code formatting" 