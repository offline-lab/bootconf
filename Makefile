.PHONY: all clean install uninstall man-pages install-man-pages uninstall-man-pages test test-coverage test-e2e lint fmt vet cross-compile help

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags="-s -w -X github.com/offline-lab/bootconf/cmd/bootconf/commands.Version=$(VERSION) -X github.com/offline-lab/bootconf/cmd/bootconf/commands.Commit=$(COMMIT) -X github.com/offline-lab/bootconf/cmd/bootconf/commands.BuildTime=$(BUILD_TIME)"

BUILDDIR := build
BINDIR := $(BUILDDIR)/bin

MANPAGES := bootconf.1

PREFIX ?= /usr/local
INSTALL_BINDIR := $(PREFIX)/bin
INSTALL_MANDIR := $(PREFIX)/share/man/man1

all: bootconf

bootconf: | $(BINDIR)
	go build $(LDFLAGS) -o $(BINDIR)/bootconf cmd/bootconf/main.go

$(BINDIR):
	@mkdir -p $(BINDIR)

man-pages: $(MANPAGES)

.1.md.1:
	@command -v pandoc >/dev/null 2>&1 || (echo "pandoc not found" && exit 1)
	pandoc -s -t man $< -o $@

clean:
	rm -rf $(BUILDDIR)
	rm -f $(MANPAGES)

install: bootconf install-man-pages
	@echo "Installing bootconf to $(INSTALL_BINDIR)..."
	@mkdir -p $(INSTALL_BINDIR)
	install -m 755 $(BINDIR)/bootconf $(INSTALL_BINDIR)/
	@echo "  installed bootconf"

uninstall:
	@echo "Removing bootconf from $(INSTALL_BINDIR)..."
	@rm -f $(INSTALL_BINDIR)/bootconf
	@echo "  removed bootconf"
	@$(MAKE) uninstall-man-pages

install-man-pages: man-pages
	@echo "Installing man pages to $(INSTALL_MANDIR)..."
	@mkdir -p $(INSTALL_MANDIR)
	@for man in $(MANPAGES); do \
		if [ -f $$man ]; then \
			install -m 644 $$man $(INSTALL_MANDIR)/ && echo "  $$man"; \
		fi; \
	done

uninstall-man-pages:
	@echo "Removing man pages from $(INSTALL_MANDIR)..."
	@for man in $(MANPAGES); do \
		if [ -f $(INSTALL_MANDIR)/$$man ]; then \
			rm -f $(INSTALL_MANDIR)/$$man && echo "  $$man"; \
		fi; \
	done

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
	docker build -t bootconf-e2e -f test/e2e/Dockerfile .
	docker run --rm bootconf-e2e

cross-compile:
	@echo "Cross-compiling for multiple platforms..."
	@mkdir -p $(BINDIR)
	@echo "  linux/amd64..."
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINDIR)/bootconf-linux-amd64 cmd/bootconf/main.go
	@echo "  linux/arm64 (Pi 4, Pi Zero 2W)..."
	@GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINDIR)/bootconf-linux-arm64 cmd/bootconf/main.go
	@echo "  linux/arm (Pi Zero)..."
	@GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINDIR)/bootconf-linux-arm cmd/bootconf/main.go
	@echo "Cross-compilation complete. Binaries in $(BINDIR)/"

help:
	@echo "Bootconf - Readonly OS boot-time configuration"
	@echo ""
	@echo "Targets:"
	@echo "  all              Build bootconf binary (default)"
	@echo "  man-pages        Generate man pages"
	@echo "  clean            Remove build artifacts"
	@echo "  install          Install all components"
	@echo "  uninstall        Remove all installed components"
	@echo "  test             Run Go tests with race detection"
	@echo "  test-coverage    Run tests and generate coverage report"
	@echo "  lint             Run golangci-lint"
	@echo "  fmt              Format Go code"
	@echo "  vet              Run go vet"
	@echo "  test-e2e         Run end-to-end test in Docker"
	@echo "  cross-compile    Build for multiple platforms"
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
	@echo "  make cross-compile             # Build for all platforms"
	@echo "  ./build/bin/bootconf version   # Print version info"
