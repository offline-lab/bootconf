# Bootconf Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build bootconf — a Go CLI tool that configures a readonly Linux OS at boot time from a YAML config file.

**Architecture:** Single Go binary with cobra CLI, module-per-section design under `internal/module/`, parallel execution via goroutines. Follows disco project conventions (cmd/internal structure, Makefile pattern, YAML config with Validate/SetDefaults).

**Tech Stack:** Go 1.24, cobra, gopkg.in/yaml.v3, no external filesystem abstraction (use real fs + temp dirs in tests).

---

### Task 1: Project Scaffold

**Files:**
- Create: `go.mod`
- Create: `cmd/bootconf/main.go`
- Create: `internal/version/version.go`
- Create: `Makefile`
- Create: `AGENTS.md`
- Create: `.gitignore`

**Step 1: Initialize Go module**

Run: `go mod init github.com/offline-lab/bootconf`

**Step 2: Create main.go entry point**

```go
package main

import (
	"os"

	"github.com/offline-lab/bootconf/cmd/bootconf/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
```

**Step 3: Create version package**

```go
package version

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)
```

**Step 4: Create minimal cobra root (placeholder)**

```go
package commands

import (
	"fmt"

	"github.com/offline-lab/bootconf/internal/version"
	"github.com/spf13/cobra"
)

var (
	configPath string
	verbose    bool
	dryRun     bool
	section    string
)

var rootCmd = &cobra.Command{
	Use:   "bootconf",
	Short: "Configure a readonly OS at boot time",
	Long: `Bootconf reads a YAML configuration file and prepares the system
before other services start. It runs every boot, not just first boot.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "/boot/firmware/bootconf.yaml", "Path to configuration file")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show build version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("bootconf %s (commit: %s, built: %s)\n", version.Version, version.Commit, version.BuildTime)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
```

**Step 5: Add cobra + yaml dependency**

Run: `go get github.com/spf13/cobra gopkg.in/yaml.v3`

**Step 6: Create Makefile (matching disco conventions)**

```makefile
.PHONY: all clean install test lint fmt vet help

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags="-s -w -X github.com/offline-lab/bootconf/internal/version.Version=$(VERSION) -X github.com/offline-lab/bootconf/internal/version.Commit=$(COMMIT) -X github.com/offline-lab/bootconf/internal/version.BuildTime=$(BUILD_TIME)"

BUILDDIR := build
BINDIR := $(BUILDDIR)/bin

PREFIX ?= /usr/local
INSTALL_BINDIR := $(PREFIX)/bin

all: bootconf

bootconf:
	go build $(LDFLAGS) -o $(BINDIR)/$@ cmd/bootconf/main.go

clean:
	rm -rf $(BUILDDIR)

install: bootconf
	install -d $(INSTALL_BINDIR)
	install -m 755 $(BINDIR)/bootconf $(INSTALL_BINDIR)/

uninstall:
	rm -f $(INSTALL_BINDIR)/bootconf

test:
	go test -v -race ./...

test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

help:
	@echo "Bootconf - Boot Configuration Utility"
	@echo ""
	@echo "Targets:"
	@echo "  all              Build binary (default)"
	@echo "  clean            Remove build artifacts"
	@echo "  install          Install binary"
	@echo "  uninstall        Remove installed binary"
	@echo "  test             Run tests with race detection"
	@echo "  test-coverage    Run tests with coverage report"
	@echo "  lint             Run golangci-lint"
	@echo "  fmt              Format Go code"
	@echo "  vet              Run go vet"
	@echo "  help             Show this help"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION    Build version (default: git tag or 'dev')"
	@echo "  PREFIX     Installation prefix (default: /usr/local)"
```

**Step 7: Create AGENTS.md**

Build/lint/test commands and code style guidelines matching disco conventions (import ordering, error handling with `%w`, naming conventions, `Validate()` / `SetDefaults()` pattern on config structs).

**Step 8: Create .gitignore**

```
build/
coverage.out
coverage.html
*.swp
.DS_Store
```

**Step 9: Verify build**

Run: `make clean && make`
Expected: Binary at `build/bin/bootconf`

Run: `./build/bin/bootconf version`
Expected: `bootconf dev (commit: unknown, built: ...)`

**Step 10: Commit**

```
feat: project scaffold with cobra CLI and Makefile
```

---

### Task 2: Config Struct + YAML Parsing

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`
- Create: `internal/config/validation.go`
- Create: `internal/config/validation_test.go`

**Step 1: Write config struct test**

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	cfgContent := `
bootconf:
  basedir: /data/config
  exclude: []

system:
  timezone: Europe/Amsterdam
  hostname: offline-lab

ssh:
  enabled: true
  keytype: ed25519
  generate_host_keys: true
  daemon: dropbear

wifi:
  enabled: true
  ssid: mynetwork
  password_hash: "hash123"
  country: NL

services:
  - name: disco
    enabled: true
    default_config:
      copy: true
      source: /etc/disco/disco.conf
      destination: disco/disco.conf

users:
  - name: admin
    enabled: true
    sudo: true
    home: /data/home/admin
    authorized_keys:
      - "ssh-ed25519 AAAA test@host"

files:
  - source: /boot/firmware/config/app.conf
    dest: app/app.conf
    chmod: 640
`
	dir := t.TempDir()
	path := filepath.Join(dir, "bootconf.yaml")
	if err := os.WriteFile(path, []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Bootconf.Basedir != "/data/config" {
		t.Errorf("basedir = %q, want /data/config", cfg.Bootconf.Basedir)
	}
	if cfg.System.Timezone != "Europe/Amsterdam" {
		t.Errorf("timezone = %q", cfg.System.Timezone)
	}
	if cfg.SSH.Daemon != "dropbear" {
		t.Errorf("daemon = %q", cfg.SSH.Daemon)
	}
	if len(cfg.Users) != 1 || cfg.Users[0].Name != "admin" {
		t.Errorf("users = %+v", cfg.Users)
	}
	if len(cfg.Files) != 1 || cfg.Files[0].Chmod != "640" {
		t.Errorf("files = %+v", cfg.Files)
	}
}

func TestLoadMissing(t *testing.T) {
	_, err := Load("/nonexistent/bootconf.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bootconf.yaml")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Bootconf.Basedir != "" {
		t.Errorf("expected empty config")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestLoad ./internal/config/`
Expected: FAIL (Load not defined)

**Step 3: Write config struct + Load function**

```go
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Bootconf BootconfConfig `yaml:"bootconf"`
	System   SystemConfig   `yaml:"system"`
	SSH      SSHConfig      `yaml:"ssh"`
	Wifi     WifiConfig     `yaml:"wifi"`
	Services []ServiceEntry `yaml:"services"`
	Users    []UserEntry    `yaml:"users"`
	Files    []FileEntry    `yaml:"files"`
}

type BootconfConfig struct {
	Basedir string   `yaml:"basedir"`
	Exclude []string `yaml:"exclude"`
}

type SystemConfig struct {
	Timezone string `yaml:"timezone"`
	Hostname string `yaml:"hostname"`
}

type SSHConfig struct {
	Enabled          bool   `yaml:"enabled"`
	Keytype          string `yaml:"keytype"`
	GenerateHostKeys bool   `yaml:"generate_host_keys"`
	Daemon           string `yaml:"daemon"`
}

type WifiConfig struct {
	Enabled      bool   `yaml:"enabled"`
	SSID         string `yaml:"ssid"`
	PasswordHash string `yaml:"password_hash"`
	Country      string `yaml:"country"`
}

type ServiceEntry struct {
	Name          string          `yaml:"name"`
	Enabled       bool            `yaml:"enabled"`
	DefaultConfig DefaultConfig   `yaml:"default_config"`
}

type DefaultConfig struct {
	Copy        bool   `yaml:"copy"`
	Source      string `yaml:"source"`
	Destination string `yaml:"destination"`
}

type UserEntry struct {
	Name           string   `yaml:"name"`
	Enabled        bool     `yaml:"enabled"`
	Sudo           bool     `yaml:"sudo"`
	Home           string   `yaml:"home"`
	AuthorizedKeys []string `yaml:"authorized_keys"`
}

type FileEntry struct {
	Source string `yaml:"source"`
	Dest   string `yaml:"dest"`
	Chmod  string `yaml:"chmod"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

func (c *Config) SetDefaults() {
	if c.Bootconf.Basedir == "" {
		c.Bootconf.Basedir = "/data/config"
	}
	if c.SSH.Keytype == "" {
		c.SSH.Keytype = "ed25519"
	}
	if c.SSH.Daemon == "" {
		c.SSH.Daemon = "dropbear"
	}
	for i := range c.Files {
		if c.Files[i].Chmod == "" {
			c.Files[i].Chmod = "640"
		}
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v -run TestLoad ./internal/config/`
Expected: PASS

**Step 5: Write validation tests**

```go
package config

import "testing"

func TestValidateValid(t *testing.T) {
	cfg := &Config{
		Bootconf: BootconfConfig{Basedir: "/data/config"},
		System:   SystemConfig{Timezone: "Europe/Amsterdam", Hostname: "test"},
		SSH:      SSHConfig{Enabled: true, Keytype: "ed25519", Daemon: "dropbear"},
		Wifi:     WifiConfig{Enabled: true, SSID: "net", PasswordHash: "hash", Country: "NL"},
		Users: []UserEntry{
			{Name: "admin", Enabled: true, Home: "/data/home/admin"},
		},
	}
	cfg.SetDefaults()

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateInvalidSSHDaemon(t *testing.T) {
	cfg := &Config{
		Bootconf: BootconfConfig{Basedir: "/data/config"},
		SSH:      SSHConfig{Enabled: true, Daemon: "badvalue"},
	}
	cfg.SetDefaults()

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for bad daemon")
	}
}

func TestValidateUserNoName(t *testing.T) {
	cfg := &Config{
		Bootconf: BootconfConfig{Basedir: "/data/config"},
		Users:    []UserEntry{{Enabled: true}},
	}
	cfg.SetDefaults()

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for user without name")
	}
}

func TestValidateServiceNoName(t *testing.T) {
	cfg := &Config{
		Bootconf: BootconfConfig{Basedir: "/data/config"},
		Services: []ServiceEntry{{Enabled: true}},
	}
	cfg.SetDefaults()

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for service without name")
	}
}

func TestValidateWifiNoSSID(t *testing.T) {
	cfg := &Config{
		Bootconf: BootconfConfig{Basedir: "/data/config"},
		Wifi:     WifiConfig{Enabled: true},
	}
	cfg.SetDefaults()

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for wifi without ssid")
	}
}
```

**Step 6: Run test to verify it fails**

Run: `go test -v -run TestValidate ./internal/config/`
Expected: FAIL

**Step 7: Write validation logic**

```go
package config

import (
	"fmt"
	"path/filepath"
)

func (c *Config) Validate() error {
	if c.Bootconf.Basedir == "" {
		return fmt.Errorf("bootconf.basedir is required")
	}
	if !filepath.IsAbs(c.Bootconf.Basedir) {
		return fmt.Errorf("bootconf.basedir must be an absolute path")
	}

	if err := c.validateSSH(); err != nil {
		return err
	}
	if err := c.validateWifi(); err != nil {
		return err
	}
	if err := c.validateUsers(); err != nil {
		return err
	}
	if err := c.validateServices(); err != nil {
		return err
	}
	if err := c.validateFiles(); err != nil {
		return err
	}
	return nil
}

func (c *Config) validateSSH() error {
	if !c.SSH.Enabled {
		return nil
	}
	validDaemons := map[string]bool{"dropbear": true, "openssh": true}
	if !validDaemons[c.SSH.Daemon] {
		return fmt.Errorf("ssh.daemon must be 'dropbear' or 'openssh', got %q", c.SSH.Daemon)
	}
	validKeytypes := map[string]bool{"ed25519": true, "rsa": true, "ecdsa": true}
	if !validKeytypes[c.SSH.Keytype] {
		return fmt.Errorf("ssh.keytype must be ed25519, rsa, or ecdsa, got %q", c.SSH.Keytype)
	}
	return nil
}

func (c *Config) validateWifi() error {
	if !c.Wifi.Enabled {
		return nil
	}
	if c.Wifi.SSID == "" {
		return fmt.Errorf("wifi.ssid is required when wifi is enabled")
	}
	if c.Wifi.PasswordHash == "" {
		return fmt.Errorf("wifi.password_hash is required when wifi is enabled")
	}
	if c.Wifi.Country == "" {
		return fmt.Errorf("wifi.country is required when wifi is enabled")
	}
	return nil
}

func (c *Config) validateUsers() error {
	for i, u := range c.Users {
		if !u.Enabled {
			continue
		}
		if u.Name == "" {
			return fmt.Errorf("users[%d].name is required", i)
		}
		if u.Home == "" {
			return fmt.Errorf("users[%d].home is required for enabled user %q", i, u.Name)
		}
	}
	return nil
}

func (c *Config) validateServices() error {
	for i, s := range c.Services {
		if !s.Enabled {
			continue
		}
		if s.Name == "" {
			return fmt.Errorf("services[%d].name is required", i)
		}
		if s.DefaultConfig.Copy {
			if s.DefaultConfig.Source == "" {
				return fmt.Errorf("services[%d].default_config.source is required when copy is true", i)
			}
			if s.DefaultConfig.Destination == "" {
				return fmt.Errorf("services[%d].default_config.destination is required when copy is true", i)
			}
		}
	}
	return nil
}

func (c *Config) validateFiles() error {
	for i, f := range c.Files {
		if f.Source == "" {
			return fmt.Errorf("files[%d].source is required", i)
		}
		if f.Dest == "" {
			return fmt.Errorf("files[%d].dest is required", i)
		}
	}
	return nil
}
```

**Step 8: Run tests**

Run: `go test -v ./internal/config/`
Expected: ALL PASS

**Step 9: Commit**

```
feat: config struct with YAML parsing and validation
```

---

### Task 3: Logging Package

**Files:**
- Create: `internal/logging/logging.go`
- Create: `internal/logging/logging_test.go`

**Step 1: Write logging test**

```go
package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestFormatWithSection(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, DEBUG)
	l.Info("wifi", "configuring SSID %s", "mynet")

	output := buf.String()
	if !strings.Contains(output, "bootconf:") {
		t.Errorf("missing prefix, got: %s", output)
	}
	if !strings.Contains(output, "INFO") {
		t.Errorf("missing level, got: %s", output)
	}
	if !strings.Contains(output, "section::wifi") {
		t.Errorf("missing section, got: %s", output)
	}
	if !strings.Contains(output, "configuring SSID mynet") {
		t.Errorf("missing message, got: %s", output)
	}
}

func TestLogLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, WARN)
	l.Info("system", "should not appear")
	l.Warn("system", "should appear")

	if strings.Contains(buf.String(), "should not appear") {
		t.Error("info should be filtered at warn level")
	}
	if !strings.Contains(buf.String(), "should appear") {
		t.Error("warn should pass at warn level")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestLog ./internal/logging/`
Expected: FAIL

**Step 3: Write logging implementation**

Format: `bootconf: <time> <severity> <section::name> <message>`

Simple leveled logger writing to an `io.Writer` (stdout in production, buffer in tests). No file logging — stdout only (journald captures it).

```go
package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

type Logger struct {
	logger *log.Logger
	level  LogLevel
}

func New(w io.Writer, level LogLevel) *Logger {
	return &Logger{
		logger: log.New(w, "", 0),
		level:  level,
	}
}

var std = New(os.Stdout, INFO)

func SetLevel(level LogLevel) {
	std.level = level
}

func (l *Logger) logf(level LogLevel, section, format string, args ...interface{}) {
	if l.level > level {
		return
	}
	ts := time.Now().UTC().Format(time.RFC3339)
	msg := fmt.Sprintf(format, args...)
	l.logger.Printf("bootconf: %s %s section::%s %s", ts, level, section, msg)
}

func (l *Logger) Debug(section, format string, args ...interface{}) {
	l.logf(DEBUG, section, format, args...)
}

func (l *Logger) Info(section, format string, args ...interface{}) {
	l.logf(INFO, section, format, args...)
}

func (l *Logger) Warn(section, format string, args ...interface{}) {
	l.logf(WARN, section, format, args...)
}

func (l *Logger) Error(section, format string, args ...interface{}) {
	l.logf(ERROR, section, format, args...)
}

func Debug(section, format string, args ...interface{}) { std.Debug(section, format, args...) }
func Info(section, format string, args ...interface{})  { std.Info(section, format, args...) }
func Warn(section, format string, args ...interface{})  { std.Warn(section, format, args...) }
func Error(section, format string, args ...interface{}) { std.Error(section, format, args...) }
```

**Step 4: Run tests**

Run: `go test -v ./internal/logging/`
Expected: ALL PASS

**Step 5: Commit**

```
feat: logging package with section-scoped format
```

---

### Task 4: Module Interface + Runner

**Files:**
- Create: `internal/module/module.go`
- Create: `internal/module/runner.go`
- Create: `internal/module/runner_test.go`
- Create: `internal/status/status.go`
- Create: `internal/status/status_test.go`

**Step 1: Write module interface + result types**

```go
package module

import "context"

type Result struct {
	Section string
	Success bool
	Message string
	Error   string
}

type Module interface {
	Name() string
	Run(ctx context.Context, basedir string, dryRun bool) Result
}
```

**Step 2: Write status types**

```go
package status

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type SectionStatus struct {
	Section  string `json:"section"`
	Success  bool   `json:"success"`
	Message  string `json:"message,omitempty"`
	Error    string `json:"error,omitempty"`
	Duration string `json:"duration,omitempty"`
}

type RunStatus struct {
	Timestamp time.Time     `json:"timestamp"`
	Overall   bool          `json:"overall"`
	Sections  []SectionStatus `json:"sections"`
}

func Write(dir string, s *RunStatus) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create status dir: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal status: %w", err)
	}
	path := filepath.Join(dir, "status.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write status: %w", err)
	}
	return nil
}

func Read(dir string) (*RunStatus, error) {
	path := filepath.Join(dir, "status.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read status: %w", err)
	}
	var s RunStatus
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("failed to parse status: %w", err)
	}
	return &s, nil
}
```

**Step 3: Write runner test**

```go
package module

import (
	"context"
	"testing"
)

type mockModule struct {
	name    string
	success bool
	msg     string
}

func (m *mockModule) Name() string { return m.name }
func (m *mockModule) Run(_ context.Context, _ string, _ bool) Result {
	return Result{Section: m.name, Success: m.success, Message: m.msg}
}

func TestRunnerAllSuccess(t *testing.T) {
	mods := []Module{
		&mockModule{name: "system", success: true, msg: "ok"},
		&mockModule{name: "ssh", success: true, msg: "ok"},
	}
	r := NewRunner(mods)
	results := r.Run(context.Background(), "/tmp", false)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if !r.Success {
			t.Errorf("section %s failed", r.Section)
		}
	}
}

func TestRunnerPartialFailure(t *testing.T) {
	mods := []Module{
		&mockModule{name: "system", success: true, msg: "ok"},
		&mockModule{name: "wifi", success: false, msg: "", Error: "bad ssid"},
		&mockModule{name: "ssh", success: true, msg: "ok"},
	}
	r := NewRunner(mods)
	results := r.Run(context.Background(), "/tmp", false)

	wifiFailed := false
	for _, r := range results {
		if r.Section == "wifi" && !r.Success {
			wifiFailed = true
		}
	}
	if !wifiFailed {
		t.Error("wifi should have failed")
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestRunnerSingleSection(t *testing.T) {
	mods := []Module{
		&mockModule{name: "system", success: true},
		&mockModule{name: "ssh", success: true},
	}
	r := NewRunner(mods)
	r.SetSection("ssh")
	results := r.Run(context.Background(), "/tmp", false)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Section != "ssh" {
		t.Errorf("expected ssh, got %s", results[0].Section)
	}
}
```

**Step 4: Run test to verify it fails**

Run: `go test -v -run TestRunner ./internal/module/`
Expected: FAIL

**Step 5: Write runner implementation**

```go
package module

import (
	"context"
	"sync"
	"time"

	"github.com/offline-lab/bootconf/internal/logging"
)

type Runner struct {
	modules []Module
	section string
	exclude map[string]bool
}

func NewRunner(modules []Module) *Runner {
	return &Runner{
		modules: modules,
		exclude: make(map[string]bool),
	}
}

func (r *Runner) SetSection(name string) {
	r.section = name
}

func (r *Runner) SetExclude(names []string) {
	for _, n := range names {
		r.exclude[n] = true
	}
}

func (r *Runner) Run(ctx context.Context, basedir string, dryRun bool) []Result {
	var wg sync.WaitGroup
	results := make([]Result, len(r.modules))

	for i, m := range r.modules {
		if r.section != "" && m.Name() != r.section {
			continue
		}
		if r.exclude[m.Name()] {
			logging.Debug(m.Name(), "section excluded, skipping")
			continue
		}

		wg.Add(1)
		go func(idx int, mod Module) {
			defer wg.Done()
			start := time.Now()
			logging.Info(mod.Name(), "starting section")
			result := mod.Run(ctx, basedir, dryRun)
			result.Duration = time.Since(start).String()
			logging.Info(mod.Name(), "section completed in %s: success=%v", result.Duration, result.Success)
			results[idx] = result
		}(i, m)
	}

	wg.Wait()

	var active []Result
	for _, r := range results {
		if r.Section != "" {
			active = append(active, r)
		}
	}
	return active
}
```

**Step 6: Run tests**

Run: `go test -v ./internal/module/`
Expected: ALL PASS

**Step 7: Commit**

```
feat: module interface, parallel runner, and status tracking
```

---

### Task 5: System Module

**Files:**
- Create: `internal/module/system/system.go`
- Create: `internal/module/system/system_test.go`

**Step 1: Write test**

```go
package system

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSystemHostnameAndTimezone(t *testing.T) {
	dir := t.TempDir()

	mod := &SystemModule{Hostname: "test-host", Timezone: "UTC"}
	result := mod.Run(context.Background(), dir, true)

	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}
}

func TestSystemDisabled(t *testing.T) {
	mod := &SystemModule{}
	result := mod.Run(context.Background(), t.TempDir(), true)
	if !result.Success {
		t.Fatalf("disabled module should succeed")
	}
}

func TestSystemWritableCheck(t *testing.T) {
	readOnlyDir := filepath.Join(t.TempDir(), "readonly")
	os.MkdirAll(readOnlyDir, 0555)
	os.Chmod(readOnlyDir, 0555)
	defer os.Chmod(readOnlyDir, 0755)

	mod := &SystemModule{Hostname: "test"}
	result := mod.Run(context.Background(), readOnlyDir, false)
	if result.Success {
		t.Error("should fail on read-only dir")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./internal/module/system/`

**Step 3: Write implementation**

```go
package system

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/offline-lab/bootconf/internal/module"
)

type SystemModule struct {
	Hostname string
	Timezone string
}

func (m *SystemModule) Name() string { return "system" }

func (m *SystemModule) Run(_ context.Context, basedir string, dryRun bool) module.Result {
	if m.Hostname == "" && m.Timezone == "" {
		return module.Result{Section: m.Name(), Success: true, Message: "nothing to configure"}
	}

	if err := m.checkWritable(basedir); err != nil {
		return module.Result{Section: m.Name(), Success: false, Error: err.Error()}
	}

	if m.Hostname != "" {
		if !dryRun {
			cmd := exec.Command("hostnamectl", "set-hostname", m.Hostname)
			if err := cmd.Run(); err != nil {
				return module.Result{Section: m.Name(), Success: false, Error: fmt.Sprintf("hostnamectl failed: %v", err)}
			}
		}
	}

	if m.Timezone != "" {
		if !dryRun {
			cmd := exec.Command("timedatectl", "set-timezone", m.Timezone)
			if err := cmd.Run(); err != nil {
				return module.Result{Section: m.Name(), Success: false, Error: fmt.Sprintf("timedatectl failed: %v", err)}
			}
		}
	}

	return module.Result{Section: m.Name(), Success: true, Message: "configured"}
}

func (m *SystemModule) checkWritable(dir string) error {
	testFile := dir + "/.bootconf-write-test"
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("basedir %s is not writable: %w", dir, err)
	}
	os.Remove(testFile)
	return nil
}
```

**Step 4: Run tests**

Run: `go test -v ./internal/module/system/`
Expected: ALL PASS (dry-run skips exec, writable check uses temp dir)

**Step 5: Commit**

```
feat: system module (hostname, timezone)
```

---

### Task 6: SSH Module

**Files:**
- Create: `internal/module/ssh/ssh.go`
- Create: `internal/module/ssh/ssh_test.go`

**Step 1: Write test**

```go
package ssh

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/offline-lab/bootconf/internal/module"
)

func TestSSHCreatesSentinel(t *testing.T) {
	dir := t.TempDir()
	mod := New("dropbear", "ed25519", true)
	result := mod.Run(context.Background(), dir, true)
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}
}

func TestSSHDisabledRemovesSentinel(t *testing.T) {
	dir := t.TempDir()
	servicesDir := filepath.Join(dir, "services")
	os.MkdirAll(servicesDir, 0755)
	sentinel := filepath.Join(servicesDir, "ssh")
	os.WriteFile(sentinel, []byte{}, 0644)

	mod := New("dropbear", "ed25519", false)
	result := mod.Run(context.Background(), dir, false)
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Error("sentinel should be removed when disabled")
	}
}

func TestSSHDryRunNoWrite(t *testing.T) {
	dir := t.TempDir()
	mod := New("dropbear", "ed25519", true)
	result := mod.Run(context.Background(), dir, true)
	if !result.Success {
		t.Fatalf("dry run should succeed")
	}
	sentinel := filepath.Join(dir, "services", "ssh")
	if _, err := os.Stat(sentinel); err == nil {
		t.Error("dry run should not create sentinel file")
	}
}

func TestSSHInvalidDaemon(t *testing.T) {
	mod := New("badvalue", "ed25519", true)
	result := mod.Run(context.Background(), t.TempDir(), true)
	if result.Success {
		t.Error("should fail with invalid daemon")
	}
}
```

**Step 2: Write implementation**

```go
package ssh

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/offline-lab/bootconf/internal/module"
)

type SSHModule struct {
	daemon           string
	keytype          string
	generateHostKeys bool
	enabled          bool
}

func New(daemon, keytype string, generateHostKeys bool) *SSHModule {
	return &SSHModule{
		daemon:           daemon,
		keytype:          keytype,
		generateHostKeys: generateHostKeys,
		enabled:          true,
	}
}

func (m *SSHModule) Name() string { return "ssh" }

func (m *SSHModule) Run(_ context.Context, basedir string, dryRun bool) module.Result {
	if !m.enabled {
		return m.removeSentinel(basedir, dryRun)
	}

	if m.daemon != "dropbear" && m.daemon != "openssh" {
		return module.Result{Section: m.Name(), Success: false, Error: fmt.Sprintf("unsupported daemon: %s", m.daemon)}
	}

	if m.generateHostKeys {
		sshDir := filepath.Join(basedir, "ssh")
		hostkey := filepath.Join(sshDir, "hostkey")

		if _, err := os.Stat(hostkey); os.IsNotExist(err) {
			if !dryRun {
				if err := os.MkdirAll(sshDir, 0700); err != nil {
					return module.Result{Section: m.Name(), Success: false, Error: fmt.Sprintf("failed to create ssh dir: %v", err)}
				}
				var cmd *exec.Cmd
				if m.daemon == "dropbear" {
					cmd = exec.Command("dropbearkey", "-t", m.keytype, "-f", hostkey)
				} else {
					cmd = exec.Command("ssh-keygen", "-t", m.keytype, "-f", hostkey, "-N", "")
				}
				if err := cmd.Run(); err != nil {
					return module.Result{Section: m.Name(), Success: false, Error: fmt.Sprintf("key generation failed: %v", err)}
				}
			}
		}
	}

	if !dryRun {
		if err := m.createSentinel(basedir); err != nil {
			return module.Result{Section: m.Name(), Success: false, Error: err.Error()}
		}
	}

	return module.Result{Section: m.Name(), Success: true, Message: "ssh configured"}
}

func (m *SSHModule) createSentinel(basedir string) error {
	servicesDir := filepath.Join(basedir, "services")
	if err := os.MkdirAll(servicesDir, 0755); err != nil {
		return fmt.Errorf("failed to create services dir: %w", err)
	}
	return os.WriteFile(filepath.Join(servicesDir, "ssh"), []byte{}, 0644)
}

func (m *SSHModule) removeSentinel(basedir string, dryRun bool) module.Result {
	if dryRun {
		return module.Result{Section: m.Name(), Success: true, Message: "would remove ssh sentinel"}
	}
	sentinel := filepath.Join(basedir, "services", "ssh")
	os.Remove(sentinel)
	return module.Result{Section: m.Name(), Success: true, Message: "ssh sentinel removed"}
}
```

**Step 3: Run tests**

Run: `go test -v ./internal/module/ssh/`
Expected: ALL PASS

**Step 4: Commit**

```
feat: ssh module with dropbear and openssh support
```

---

### Task 7: Wifi Module

**Files:**
- Create: `internal/module/wifi/wifi.go`
- Create: `internal/module/wifi/wifi_test.go`

**Step 1: Write test**

```go
package wifi

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/offline-lab/bootconf/internal/module"
)

func TestWifiCreatesConfig(t *testing.T) {
	dir := t.TempDir()
	mod := New(true, "mynetwork", "hash123", "NL")
	result := mod.Run(context.Background(), dir, false)
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}

	config := filepath.Join(dir, "wifi", "wpa_supplicant.conf")
	data, err := os.ReadFile(config)
	if err != nil {
		t.Fatalf("config file not found: %v", err)
	}
	content := string(data)
	if content == "" {
		t.Error("config should not be empty")
	}
}

func TestWifiDisabledRemovesSentinel(t *testing.T) {
	dir := t.TempDir()
	servicesDir := filepath.Join(dir, "services")
	os.MkdirAll(servicesDir, 0755)
	sentinel := filepath.Join(servicesDir, "wifi")
	os.WriteFile(sentinel, []byte{}, 0644)

	mod := New(false, "", "", "")
	result := mod.Run(context.Background(), dir, false)
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Error("sentinel should be removed")
	}
}

func TestWifiDryRunNoWrite(t *testing.T) {
	dir := t.TempDir()
	mod := New(true, "net", "hash", "NL")
	result := mod.Run(context.Background(), dir, true)
	if !result.Success {
		t.Fatalf("dry run should succeed")
	}
	config := filepath.Join(dir, "wifi", "wpa_supplicant.conf")
	if _, err := os.Stat(config); err == nil {
		t.Error("dry run should not create config file")
	}
}
```

**Step 2: Write implementation**

Generates `wpa_supplicant.conf` from `password_hash` directly. Template-based, no external tooling needed.

```go
package wifi

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/offline-lab/bootconf/internal/module"
)

type WifiModule struct {
	enabled      bool
	ssid         string
	passwordHash string
	country      string
}

func New(enabled bool, ssid, passwordHash, country string) *WifiModule {
	return &WifiModule{
		enabled:      enabled,
		ssid:         ssid,
		passwordHash: passwordHash,
		country:      country,
	}
}

func (m *WifiModule) Name() string { return "wifi" }

func (m *WifiModule) Run(_ context.Context, basedir string, dryRun bool) module.Result {
	if !m.enabled {
		return m.removeSentinel(basedir, dryRun)
	}

	if !dryRun {
		wifiDir := filepath.Join(basedir, "wifi")
		if err := os.MkdirAll(wifiDir, 0755); err != nil {
			return module.Result{Section: m.Name(), Success: false, Error: fmt.Sprintf("failed to create wifi dir: %v", err)}
		}

		config := m.generateConfig()
		configPath := filepath.Join(wifiDir, "wpa_supplicant.conf")
		if err := os.WriteFile(configPath, []byte(config), 0600); err != nil {
			return module.Result{Section: m.Name(), Success: false, Error: fmt.Sprintf("failed to write config: %v", err)}
		}

		if err := m.createSentinel(basedir); err != nil {
			return module.Result{Section: m.Name(), Success: false, Error: err.Error()}
		}
	}

	return module.Result{Section: m.Name(), Success: true, Message: "wifi configured"}
}

func (m *WifiModule) generateConfig() string {
	return fmt.Sprintf(`country=%s
ctrl_interface=DIR=/var/run/wpa_supplicant GROUP=netdev
update_config=1

network={
    ssid="%s"
    psk=%s
}
`, m.country, m.ssid, m.passwordHash)
}

func (m *WifiModule) createSentinel(basedir string) error {
	servicesDir := filepath.Join(basedir, "services")
	if err := os.MkdirAll(servicesDir, 0755); err != nil {
		return fmt.Errorf("failed to create services dir: %w", err)
	}
	return os.WriteFile(filepath.Join(servicesDir, "wifi"), []byte{}, 0644)
}

func (m *WifiModule) removeSentinel(basedir string, dryRun bool) module.Result {
	if dryRun {
		return module.Result{Section: m.Name(), Success: true, Message: "would remove wifi sentinel"}
	}
	os.Remove(filepath.Join(basedir, "services", "wifi"))
	return module.Result{Section: m.Name(), Success: true, Message: "wifi sentinel removed"}
}
```

**Step 3: Run tests**

Run: `go test -v ./internal/module/wifi/`
Expected: ALL PASS

**Step 4: Commit**

```
feat: wifi module with wpa_supplicant.conf generation
```

---

### Task 8: Services Module

**Files:**
- Create: `internal/module/services/services.go`
- Create: `internal/module/services/services_test.go`

**Step 1: Write test**

```go
package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/offline-lab/bootconf/internal/config"
	"github.com/offline-lab/bootconf/internal/module"
)

func TestServicesEnable(t *testing.T) {
	dir := t.TempDir()
	entries := []config.ServiceEntry{
		{Name: "disco", Enabled: true},
	}
	mod := New(entries)
	result := mod.Run(context.Background(), dir, false)
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}
	sentinel := filepath.Join(dir, "services", "disco")
	if _, err := os.Stat(sentinel); os.IsNotExist(err) {
		t.Error("sentinel file should exist")
	}
}

func TestServicesDisableRemovesSentinel(t *testing.T) {
	dir := t.TempDir()
	servicesDir := filepath.Join(dir, "services")
	os.MkdirAll(servicesDir, 0755)
	os.WriteFile(filepath.Join(servicesDir, "oldservice"), []byte{}, 0644)

	entries := []config.ServiceEntry{
		{Name: "oldservice", Enabled: false},
	}
	mod := New(entries)
	result := mod.Run(context.Background(), dir, false)
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}
	if _, err := os.Stat(filepath.Join(servicesDir, "oldservice")); !os.IsNotExist(err) {
		t.Error("disabled service sentinel should be removed")
	}
}

func TestServicesCopyDefaultConfig(t *testing.T) {
	dir := t.TempDir()

	sourceDir := filepath.Join(dir, "etc", "disco")
	os.MkdirAll(sourceDir, 0755)
	sourceFile := filepath.Join(sourceDir, "disco.conf")
	os.WriteFile(sourceFile, []byte("key=value"), 0644)

	entries := []config.ServiceEntry{
		{
			Name:    "disco",
			Enabled: true,
			DefaultConfig: config.DefaultConfig{
				Copy:        true,
				Source:      sourceFile,
				Destination: "disco/disco.conf",
			},
		},
	}
	mod := New(entries)
	result := mod.Run(context.Background(), dir, false)
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}

	copied := filepath.Join(dir, "disco", "disco.conf")
	data, err := os.ReadFile(copied)
	if err != nil {
		t.Fatalf("copied config not found: %v", err)
	}
	if string(data) != "key=value" {
		t.Errorf("unexpected content: %s", data)
	}
}

func TestServicesExistingFileGetsNewSuffix(t *testing.T) {
	dir := t.TempDir()

	sourceDir := filepath.Join(dir, "etc", "app")
	os.MkdirAll(sourceDir, 0755)
	os.WriteFile(filepath.Join(sourceDir, "app.conf"), []byte("new"), 0644)

	destDir := filepath.Join(dir, "app")
	os.MkdirAll(destDir, 0755)
	os.WriteFile(filepath.Join(destDir, "app.conf"), []byte("old"), 0644)

	entries := []config.ServiceEntry{
		{
			Name:    "app",
			Enabled: true,
			DefaultConfig: config.DefaultConfig{
				Copy:        true,
				Source:      filepath.Join(sourceDir, "app.conf"),
				Destination: "app/app.conf",
			},
		},
	}
	mod := New(entries)
	result := mod.Run(context.Background(), dir, false)
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}

	original := filepath.Join(destDir, "app.conf")
	data, _ := os.ReadFile(original)
	if string(data) != "old" {
		t.Error("original file should not be overwritten")
	}

	newFile := filepath.Join(destDir, "app.conf.new")
	data, _ = os.ReadFile(newFile)
	if string(data) != "new" {
		t.Error(".new file should have new content")
	}
}
```

**Step 2: Write implementation**

```go
package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/offline-lab/bootconf/internal/config"
	"github.com/offline-lab/bootconf/internal/module"
)

type ServicesModule struct {
	entries []config.ServiceEntry
}

func New(entries []config.ServiceEntry) *ServicesModule {
	return &ServicesModule{entries: entries}
}

func (m *ServicesModule) Name() string { return "services" }

func (m *ServicesModule) Run(_ context.Context, basedir string, dryRun bool) module.Result {
	var errors []string

	for _, entry := range m.entries {
		sentinel := filepath.Join(basedir, "services", entry.Name)

		if entry.Enabled {
			if !dryRun {
				if err := os.MkdirAll(filepath.Join(basedir, "services"), 0755); err != nil {
					errors = append(errors, fmt.Sprintf("%s: %v", entry.Name, err))
					continue
				}
				if err := os.WriteFile(sentinel, []byte{}, 0644); err != nil {
					errors = append(errors, fmt.Sprintf("%s: %v", entry.Name, err))
					continue
				}
			}

			if entry.DefaultConfig.Copy && !dryRun {
				if err := copyConfig(basedir, entry.DefaultConfig); err != nil {
					errors = append(errors, fmt.Sprintf("%s config: %v", entry.Name, err))
				}
			}
		} else {
			if !dryRun {
				os.Remove(sentinel)
			}
		}
	}

	if len(errors) > 0 {
		return module.Result{Section: m.Name(), Success: false, Error: fmt.Sprintf("%d errors: %v", len(errors), errors)}
	}
	return module.Result{Section: m.Name(), Success: true, Message: fmt.Sprintf("%d services processed", len(m.entries))}
}

func copyConfig(basedir string, dc config.DefaultConfig) error {
	destPath := filepath.Join(basedir, dc.Destination)

	if _, err := os.Stat(destPath); err == nil {
		destPath = destPath + ".new"
	}

	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0750); err != nil {
		return fmt.Errorf("failed to create dest dir: %w", err)
	}

	src, err := os.Open(dc.Source)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer src.Close()

	dst, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0640)
	if err != nil {
		return fmt.Errorf("failed to create dest: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy: %w", err)
	}

	return os.Chown(destPath, 0, 0)
}
```

**Step 3: Run tests**

Run: `go test -v ./internal/module/services/`
Expected: ALL PASS

**Step 4: Commit**

```
feat: services module with sentinel files and config copy
```

---

### Task 9: Users Module

**Files:**
- Create: `internal/module/users/users.go`
- Create: `internal/module/users/users_test.go`

**Step 1: Write test**

```go
package users

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/offline-lab/bootconf/internal/config"
	"github.com/offline-lab/bootconf/internal/module"
)

func TestUsersCreateHomeAndKeys(t *testing.T) {
	dir := t.TempDir()

	entries := []config.UserEntry{
		{
			Name:           "admin",
			Enabled:        true,
			Sudo:           true,
			Home:           filepath.Join(dir, "home", "admin"),
			AuthorizedKeys: []string{"ssh-ed25519 AAAA test@host"},
		},
	}
	mod := New(entries, 2000)
	result := mod.Run(context.Background(), dir, false)
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}

	sshDir := filepath.Join(dir, "home", "admin", ".ssh")
	keys := filepath.Join(sshDir, "authorized_keys")
	data, err := os.ReadFile(keys)
	if err != nil {
		t.Fatalf("authorized_keys not found: %v", err)
	}
	if string(data) != "ssh-ed25519 AAAA test@host\n" {
		t.Errorf("unexpected keys content: %q", data)
	}

	sudoers := filepath.Join(dir, "sudoers.d", "admin")
	if _, err := os.Stat(sudoers); os.IsNotExist(err) {
		t.Error("sudoers file should exist")
	}
}

func TestUsersDisabledRemovesSentinel(t *testing.T) {
	dir := t.TempDir()
	servicesDir := filepath.Join(dir, "services")
	os.MkdirAll(servicesDir, 0755)
	os.WriteFile(filepath.Join(servicesDir, "olduser"), []byte{}, 0644)

	entries := []config.UserEntry{
		{Name: "olduser", Enabled: false, Home: "/data/home/olduser"},
	}
	mod := New(entries, 2000)
	result := mod.Run(context.Background(), dir, false)
	if !result.Success {
		t.Fatalf("expected success")
	}
	if _, err := os.Stat(filepath.Join(servicesDir, "olduser")); !os.IsNotExist(err) {
		t.Error("sentinel should be removed for disabled user")
	}
}
```

**Step 2: Write implementation**

```go
package users

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/offline-lab/bootconf/internal/config"
	"github.com/offline-lab/bootconf/internal/module"
)

type UsersModule struct {
	entries  []config.UserEntry
	uidStart int
}

func New(entries []config.UserEntry, uidStart int) *UsersModule {
	return &UsersModule{entries: entries, uidStart: uidStart}
}

func (m *UsersModule) Name() string { return "users" }

func (m *UsersModule) Run(_ context.Context, basedir string, dryRun bool) module.Result {
	var errors []string

	for i, entry := range m.entries {
		if !entry.Enabled {
			if !dryRun {
				m.disableUser(basedir, entry.Name)
			}
			continue
		}

		uid := m.uidStart + i

		if !dryRun {
			if err := m.createUser(basedir, entry, uid); err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", entry.Name, err))
			}
		}
	}

	if len(errors) > 0 {
		return module.Result{Section: m.Name(), Success: false, Error: strings.Join(errors, "; ")}
	}
	return module.Result{Section: m.Name(), Success: true, Message: fmt.Sprintf("%d users processed", len(m.entries))}
}

func (m *UsersModule) createUser(basedir string, entry config.UserEntry, uid int) error {
	sysusersDir := filepath.Join(basedir, "sysusers.d")
	if err := os.MkdirAll(sysusersDir, 0755); err != nil {
		return fmt.Errorf("failed to create sysusers dir: %w", err)
	}

	sysusersContent := fmt.Sprintf("u %s %d \"%s\" %s /bin/bash\n", entry.Name, uid, entry.Name, entry.Home)
	if err := os.WriteFile(filepath.Join(sysusersDir, entry.Name+".conf"), []byte(sysusersContent), 0644); err != nil {
		return fmt.Errorf("failed to write sysusers config: %w", err)
	}

	if err := os.MkdirAll(entry.Home, 0755); err != nil {
		return fmt.Errorf("failed to create home dir: %w", err)
	}
	os.Chown(entry.Home, uid, uid)

	sshDir := filepath.Join(entry.Home, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to create .ssh dir: %w", err)
	}
	os.Chown(sshDir, uid, uid)

	if len(entry.AuthorizedKeys) > 0 {
		content := strings.Join(entry.AuthorizedKeys, "\n") + "\n"
		keysFile := filepath.Join(sshDir, "authorized_keys")
		if err := os.WriteFile(keysFile, []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to write authorized_keys: %w", err)
		}
		os.Chown(keysFile, uid, uid)
	}

	if entry.Sudo {
		sudoersDir := filepath.Join(basedir, "sudoers.d")
		if err := os.MkdirAll(sudoersDir, 0750); err != nil {
			return fmt.Errorf("failed to create sudoers dir: %w", err)
		}
		sudoContent := fmt.Sprintf("%s ALL=(ALL) ALL\n", entry.Name)
		if err := os.WriteFile(filepath.Join(sudoersDir, entry.Name), []byte(sudoContent), 0440); err != nil {
			return fmt.Errorf("failed to write sudoers: %w", err)
		}
	}

	servicesDir := filepath.Join(basedir, "services")
	if err := os.MkdirAll(servicesDir, 0755); err != nil {
		return fmt.Errorf("failed to create services dir: %w", err)
	}
	return os.WriteFile(filepath.Join(servicesDir, "user-"+entry.Name), []byte{}, 0644)
}

func (m *UsersModule) disableUser(basedir, name string) {
	os.Remove(filepath.Join(basedir, "services", "user-"+name))
	os.Remove(filepath.Join(basedir, "sysusers.d", name+".conf"))
	os.Remove(filepath.Join(basedir, "sudoers.d", name))
	exec.Command("userdel", name).Run()
}
```

**Step 3: Run tests**

Run: `go test -v ./internal/module/users/`
Expected: ALL PASS

**Step 4: Commit**

```
feat: users module with sysusers, homedir, authorized_keys, sudoers
```

---

### Task 10: Files Module

**Files:**
- Create: `internal/module/files/files.go`
- Create: `internal/module/files/files_test.go`

**Step 1: Write test**

```go
package files

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/offline-lab/bootconf/internal/config"
	"github.com/offline-lab/bootconf/internal/module"
)

func TestFilesCopyFile(t *testing.T) {
	dir := t.TempDir()

	source := filepath.Join(dir, "source", "app.conf")
	os.MkdirAll(filepath.Dir(source), 0755)
	os.WriteFile(source, []byte("key=value"), 0644)

	entries := []config.FileEntry{
		{Source: source, Dest: "app/app.conf", Chmod: "640"},
	}
	mod := New(entries)
	result := mod.Run(dir, false)
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}

	copied := filepath.Join(dir, "app", "app.conf")
	data, err := os.ReadFile(copied)
	if err != nil {
		t.Fatalf("file not found: %v", err)
	}
	if string(data) != "key=value" {
		t.Errorf("unexpected content: %s", data)
	}
}

func TestFilesExistingFileGetsNewSuffix(t *testing.T) {
	dir := t.TempDir()

	source := filepath.Join(dir, "src", "new.conf")
	os.MkdirAll(filepath.Dir(source), 0755)
	os.WriteFile(source, []byte("new content"), 0644)

	destDir := filepath.Join(dir, "app")
	os.MkdirAll(destDir, 0755)
	os.WriteFile(filepath.Join(destDir, "app.conf"), []byte("old content"), 0644)

	entries := []config.FileEntry{
		{Source: source, Dest: "app/app.conf", Chmod: "640"},
	}
	mod := New(entries)
	result := mod.Run(dir, false)
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}

	original := filepath.Join(destDir, "app.conf")
	data, _ := os.ReadFile(original)
	if string(data) != "old content" {
		t.Error("original should not be overwritten")
	}

	newFile := filepath.Join(destDir, "app.conf.new")
	data, _ = os.ReadFile(newFile)
	if string(data) != "new content" {
		t.Error(".new file should have new content")
	}
}
```

**Step 2: Write implementation**

```go
package files

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/offline-lab/bootconf/internal/config"
	"github.com/offline-lab/bootconf/internal/module"
)

type FilesModule struct {
	entries []config.FileEntry
}

func New(entries []config.FileEntry) *FilesModule {
	return &FilesModule{entries: entries}
}

func (m *FilesModule) Name() string { return "files" }

func (m *FilesModule) Run(basedir string, dryRun bool) module.Result {
	if dryRun {
		return module.Result{Section: m.Name(), Success: true, Message: "dry run"}
	}

	var errors []string
	for _, entry := range m.entries {
		if err := m.copyEntry(basedir, entry); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", entry.Dest, err))
		}
	}

	if len(errors) > 0 {
		return module.Result{Section: m.Name(), Success: false, Error: fmt.Sprintf("%d errors", len(errors))}
	}
	return module.Result{Section: m.Name(), Success: true, Message: fmt.Sprintf("%d files processed", len(m.entries))}
}
```

Wait — the Module interface expects `Run(ctx context.Context, basedir string, dryRun bool) Result`. Let me fix the signature:

```go
func (m *FilesModule) Run(_ context.Context, basedir string, dryRun bool) module.Result {
```

**Step 3: Run tests**

Run: `go test -v ./internal/module/files/`
Expected: ALL PASS

**Step 4: Commit**

```
feat: files module with .new suffix for existing files
```

---

### Task 11: CLI Subcommands (run, status, validate, check)

**Files:**
- Create: `cmd/bootconf/commands/run.go`
- Create: `cmd/bootconf/commands/status.go`
- Create: `cmd/bootconf/commands/validate.go`
- Create: `cmd/bootconf/commands/check.go`
- Modify: `cmd/bootconf/commands/root.go`

**Step 1: Wire modules into CLI run command**

The `run` command loads config, validates, constructs modules from config, runs the runner, writes status.

**Step 2: Write status command**

Reads `/data/.bootconf/status.json` (or basedir-derived path) and displays results.

**Step 3: Write validate command**

Loads config, calls `SetDefaults()`, calls `Validate()`, reports errors. Works fully offline.

**Step 4: Write check command**

Checks runtime state: is wifi up? is ssh running? do users exist? (calls `systemctl is-active`, `id <user>`, etc.)

**Step 5: Wire all commands into root**

**Step 6: Verify all commands**

Run:
```bash
./build/bin/bootconf validate --config bootconf.yaml
./build/bin/bootconf run --dry-run --config bootconf.yaml --verbose
./build/bin/bootconf status
./build/bin/bootconf version
./build/bin/bootconf help
```

**Step 7: Commit**

```
feat: wire all CLI subcommands (run, status, validate, check)
```

---

### Task 12: Full Integration Test

**Files:**
- Create: `test/integration_test.go`

**Step 1: Write end-to-end test**

Uses a temp directory as basedir. Loads the example bootconf.yaml (with paths adjusted). Runs all modules. Verifies:
- Sentinel files created for enabled services
- wpa_supplicant.conf generated
- User home dirs + authorized_keys created
- Sysusers.d configs written
- Sudoers configs written
- Status file written and readable

**Step 2: Run full test**

Run: `go test -v ./test/`
Expected: PASS

**Step 3: Commit**

```
test: add integration test for full bootconf run
```

---

### Task 13: README + Final Cleanup

**Files:**
- Create: `README.md`
- Modify: `planning/tasks.md` (mark Phase 1-2 as done)

**Step 1: Write README with build instructions, usage examples, config reference.**

**Step 2: Run full test suite**

Run: `make test && make lint && make vet`
Expected: ALL PASS

**Step 3: Verify binary**

Run: `make clean && make && ./build/bin/bootconf help`
Expected: Help output

**Step 4: Update planning docs**

Mark all Phase 1-3 tasks as done in tasks.md.

**Step 5: Commit**

```
docs: add README with build and usage instructions
```
