BINARY_NAME=gh-install-from
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u '+%Y-%m-%d-%H%M UTC')

# Build flags
LDFLAGS=-ldflags "-X github.com/realloser/gh-install-from/pkg/version.Version=$(VERSION) \
                  -X github.com/realloser/gh-install-from/pkg/version.Commit=$(COMMIT) \
                  -X github.com/realloser/gh-install-from/pkg/version.Date=$(DATE)"

# Supported platforms
PLATFORMS=darwin/amd64 darwin/arm64 linux/386 linux/amd64 linux/arm linux/arm64 windows/386 windows/amd64

# Generate platform-specific targets
PLATFORM_TARGETS=$(foreach PLATFORM,$(PLATFORMS),dist/$(BINARY_NAME)_$(word 1,$(subst /, ,$(PLATFORM)))_$(word 2,$(subst /, ,$(PLATFORM))).tar.gz)

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

.PHONY: test
test:
	go test ./...

.PHONY: lint
lint:
	go vet ./...
	go fmt ./...

# Pattern rule for building platform-specific binaries
dist/%/$(BINARY_NAME)%: | dist/%
	GOOS=$(word 1,$(subst _, ,$(notdir $(basename $@)))) \
	GOARCH=$(word 2,$(subst _, ,$(notdir $(basename $@)))) \
	FILENAME=$(BINARY_NAME)$(if $(findstring windows,$(word 1,$(subst _, ,$(notdir $(basename $@))))),.exe,) \
	go build $(LDFLAGS) -o $@/$${FILENAME}

# Pattern rule for creating directories
dist/%:
	mkdir -p $@

# Pattern rule for creating tarballs
dist/%.tar.gz: dist/%/$(BINARY_NAME)%
	cd dist/$(notdir $(basename $@)) && \
	tar -czf ../$(notdir $@) \
		$(BINARY_NAME)$(if $(findstring windows,$(word 1,$(subst _, ,$(notdir $(basename $@))))),.exe,)

.PHONY: release
release: clean $(PLATFORM_TARGETS)

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