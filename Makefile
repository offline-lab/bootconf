.PHONY: all clean install uninstall test test-coverage test-e2e lint fmt vet help

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags="-s -w -X github.com/offline-lab/bootconf/internal/version.Version=$(VERSION) -X github.com/offline-lab/bootconf/internal/version.Commit=$(COMMIT) -X github.com/offline-lab/bootconf/internal/version.BuildTime=$(BUILD_TIME)"

BUILDDIR := build
BINDIR := $(BUILDDIR)/bin

PREFIX ?= /usr/local
INSTALL_BINDIR := $(PREFIX)/bin

all: bootconf

bootconf: | $(BINDIR)
	go build $(LDFLAGS) -o $(BINDIR)/bootconf cmd/bootconf/main.go

$(BINDIR):
	@mkdir -p $(BINDIR)

clean:
	rm -rf $(BUILDDIR)

install: bootconf
	@echo "Installing bootconf to $(INSTALL_BINDIR)..."
	@mkdir -p $(INSTALL_BINDIR)
	install -m 755 $(BINDIR)/bootconf $(INSTALL_BINDIR)/
	@echo "  installed bootconf"

uninstall:
	@echo "Removing bootconf from $(INSTALL_BINDIR)..."
	@rm -f $(INSTALL_BINDIR)/bootconf
	@echo "  removed bootconf"

test:
	go test -v -race ./...

test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint:
	@command -v golangci-lint >/dev/null 2>&1 || (echo "golangci-lint not found" && exit 1)
	golangci-lint run ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

test-e2e:
	docker build -t bootconf-e2e .
	docker run --rm bootconf-e2e

help:
	@echo "Bootconf - Readonly OS boot-time configuration"
	@echo ""
	@echo "Targets:"
	@echo "  all              Build bootconf binary (default)"
	@echo "  clean            Remove build artifacts"
	@echo "  install          Install bootconf to PREFIX"
	@echo "  uninstall        Remove installed bootconf"
	@echo "  test             Run Go tests with race detection"
	@echo "  test-coverage    Run tests and generate coverage report"
	@echo "  lint             Run golangci-lint"
	@echo "  fmt              Format Go code"
	@echo "  vet              Run go vet"
	@echo "  test-e2e         Run end-to-end test in Docker (requires Docker)"
	@echo "  help             Show this help"
	@echo ""
	@echo "Build output:"
	@echo "  Binary: $(BINDIR)/bootconf"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION    Build version (default: git tag or 'dev')"
	@echo "  COMMIT     Git commit (default: git HEAD)"
	@echo "  PREFIX     Installation prefix (default: /usr/local)"
	@echo ""
	@echo "Examples:"
	@echo "  make                           # Build bootconf"
	@echo "  make VERSION=v1.0.0 all        # Build with specific version"
	@echo "  make PREFIX=/usr install       # Install to /usr/bin"
	@echo "  ./build/bin/bootconf version   # Print version info"
