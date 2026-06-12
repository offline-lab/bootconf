# AGENTS.md: Bootconf Developer Guide

## Quick Reference

```bash
make                          # Build binary to build/bin/bootconf
make test                     # go test -v -race ./...
go test -v -run TestFoo ./... # Run single test
gofmt -s -w .                 # Format all Go files
golangci-lint run             # Lint
```

---

## Project Layout

```
cmd/bootconf/
  main.go               — Thin entry point; calls commands.Execute()
  commands/
    root.go             — Root cobra command and flag setup
    run.go              — "bootconf run" — executes all modules
    status.go           — "bootconf status" — reads last run status
    validate.go         — "bootconf validate" — validates config file
    check.go            — "bootconf check" — checks service/user state
    version.go          — Build vars (Version, Commit, BuildTime via LDFLAGS)

internal/
  config/               — YAML config loading, SetDefaults, and validation
  logging/              — Leveled structured logger (DEBUG/INFO/WARN/ERROR)
  module/               — Module interface, Runner, per-module packages
    module.go           — Module interface and Result type
    runner.go           — Sequential runner (order set by bootconf.order)
    files/              — Copy files from source to destination
    services/           — Manage service sentinel files and default configs
    ssh/                — Generate SSH host keys, manage ssh sentinel
    system/             — Apply hostname and timezone
    users/              — Provision users via systemd-sysusers
    wifi/               — Write wpa_supplicant.conf and wifi sentinel
  output/               — Table rendering for CLI output
  registry/             — Single source of truth for module names and constructors
  run/                  — Thin exec.CommandContext wrapper (internal/run.Command)
  status/               — Read/write status.json in bootconf.directory
```

---

## Adding a New Module

This section is the authoritative guide for implementing a new bootconf module.
Read the existing modules alongside this guide; they are the reference implementation.

### 1. Create the package

```
internal/module/<name>/<name>.go
internal/module/<name>/<name>_test.go
```

Where `<name>` is a lowercase single word (e.g. `ntp`, `dns`, `firewall`).

### 2. Add a config struct in `internal/config/config.go`

```go
type NtpConfig struct {
    Enabled bool     `yaml:"enabled"`
    Servers []string `yaml:"servers"`
}
```

Add it to the root `Config` struct and to `SetDefaults()` if defaults are needed.
Add validation in `internal/config/validation.go`; the `Validate()` method calls per-section validators.

### 3. Implement the module

The module type must be named `<Name>Module` and must implement `module.Module`:

```go
package ntp

import (
    "context"
    "fmt"

    "github.com/offline-lab/bootconf/internal/config"
    "github.com/offline-lab/bootconf/internal/logging"
    "github.com/offline-lab/bootconf/internal/module"
    "github.com/offline-lab/bootconf/internal/run"
)

// NtpModule configures NTP servers at boot.
type NtpModule struct {
    enabled bool
    servers []string
}

// New creates an NtpModule from the given NTP config.
func New(cfg config.NtpConfig) *NtpModule {
    return &NtpModule{
        enabled: cfg.Enabled,
        servers: cfg.Servers,
    }
}

// Name returns the module identifier "ntp".
func (ntpModule *NtpModule) Name() string { return "ntp" }

// Run configures NTP servers. In dry-run mode each I/O action is logged
// but not executed.
func (ntpModule *NtpModule) Run(ctx context.Context, dryRun bool) module.Result {
    if !ntpModule.enabled {
        return module.Result{Section: ntpModule.Name(), Success: true, Message: "ntp disabled"}
    }

    for _, server := range ntpModule.servers {
        if dryRun {
            logging.Info(ntpModule.Name(), "would configure NTP server %s (dry-run)", server)
        } else {
            logging.Info(ntpModule.Name(), "configuring NTP server %s", server)
            if err := run.Command(ctx, "timedatectl", "ntp-server", server); err != nil {
                logging.Error(ntpModule.Name(), "failed to set NTP server %s: %v", server, err)
                errMsg := fmt.Sprintf("failed to set NTP server %s: %v", server, err)
                return module.Result{Section: ntpModule.Name(), Success: false, Error: errMsg}
            }
        }
    }

    if dryRun {
        return module.Result{Section: ntpModule.Name(), Success: true, Message: "ntp configured (dry-run)"}
    }
    return module.Result{Section: ntpModule.Name(), Success: true, Message: fmt.Sprintf("configured %d NTP server(s)", len(ntpModule.servers))}
}
```

#### Constructor rules

- Always named `New`. Never `NewNtpModule`; the package name provides the context.
- First argument is always the config type for this module.
- Extra arguments (e.g., a shared services directory) follow the config arg.
- Pull all needed fields from config into the struct in `New`; `Run` must not read config directly.

#### Receiver naming

Receiver must be the full type name, lowercase: `ntpModule *NtpModule`, not `m *NtpModule`.

### 4. Dry-run pattern

**Never bail out early on dry-run.** The whole point of dry-run is to exercise every code path
except the actual I/O. A module that returns immediately on `dryRun == true` tells the user
nothing useful.

The pattern is: guard each I/O action individually.

```go
// CORRECT — full code path runs, only I/O is skipped
if dryRun {
    logging.Info(ntpModule.Name(), "would write /etc/ntp.conf (dry-run)")
} else {
    if err := os.WriteFile("/etc/ntp.conf", content, 0644); err != nil {
        ...
    }
}

// WRONG — bails before any logic runs
if dryRun {
    return module.Result{..., Message: "dry-run: skipped"}
}
```

### 5. Logging pattern

Use `internal/logging` for all module output. Never write to stdout or use `fmt.Print`.

| Level   | When to use |
|---------|-------------|
| `Info`  | Each meaningful action: "writing sentinel /srv/bootconf/ssh", "setting hostname to myhost" |
| `Info`  | Each dry-run action: "would write sentinel /srv/bootconf/ssh (dry-run)" |
| `Debug` | Skipped work: "host key already exists at /etc/dropbear/hostkey, skipping" |
| `Warn`  | Non-fatal failures: chown failures, cleanup errors that don't fail the module |
| `Error` | Before returning any error result |

Always log the specific path, name, or value in the message. "failed to write config" is not useful.
"failed to write /etc/ntp.conf: permission denied" is.

### 6. Error handling

- Return the first hard error via `module.Result{Success: false, Error: errMsg}`.
- For modules processing a list (files, services, users), collect all errors and report the count,
  similar to how `files` and `services` do it.
- Non-fatal side effects (chown failures, cleanup errors) use `logging.Warn` and continue.
- Never discard errors silently. Use `_ = expr` only for `defer f.Close()`.

### 7. External commands

Always use `internal/run.Command(ctx, name, args...)`. Never call `exec.Command` directly in a module.
This ensures the context is threaded through, errors include the full command line, and the pattern
is consistent across modules.

```go
if err := run.Command(ctx, "hostnamectl", "set-hostname", ntpModule.hostname); err != nil {
    // err already contains the full command line
    logging.Error(ntpModule.Name(), "failed to set hostname: %v", err)
    ...
}
```

### 8. Register the module in `internal/registry/registry.go`

This is the only file that needs editing to wire in a new module. Add one `Entry` to `Modules`:

```go
{"ntp", func(cfg *config.Config) module.Module { return ntp.New(cfg.Ntp) }},
```

Also add the import at the top of that file:

```go
"github.com/offline-lab/bootconf/internal/module/ntp"
```

The position in the slice sets the default execution order. Modules run **sequentially** in the
order specified by `bootconf.order` (defaulting to the slice order in `registry.Modules`).

### 9. Write tests

Tests live alongside the module: `internal/module/ntp/ntp_test.go`.
Use `t.TempDir()` for all filesystem work. Never write to fixed paths.

Minimum test coverage:
- Disabled module returns success with "ntp disabled" message.
- Dry-run returns success, writes nothing to disk.
- Happy path: actual action applied correctly.
- Error path: non-existent source / unwritable destination returns failure.

```go
func TestNtpDisabled(t *testing.T) {
    result := New(config.NtpConfig{Enabled: false}).Run(context.Background(), false)
    if !result.Success {
        t.Fatalf("expected success, got: %s", result.Error)
    }
    if result.Message != "ntp disabled" {
        t.Errorf("unexpected message: %q", result.Message)
    }
}
```

---

## Code Style

### Naming

- No one- or two-letter variable names anywhere, including receivers and loop variables.
- No abbreviations: `sourceFile` not `src`, `destinationPath` not `dst`, `character` not `char`.
- Receiver names match the type: `(ntpModule *NtpModule)`, not `(m *NtpModule)`.
- Unexported functions: camelCase. Exported: PascalCase. Never mixed-case packages.
- Booleans: prefix `is`, `has`, `can` (e.g. `isEnabled`, `hasKeys`).

### Comments

Write no comments by default. Add one only when the **why** is non-obvious: a hidden constraint,
a subtle invariant, a workaround for a known external bug. A comment that says what the code
does is noise; the identifiers already say that.

```go
// WRONG — states the obvious
// Write the sentinel file to enable the service.
if err := os.WriteFile(sentinelPath, []byte{}, 0640); err != nil { ... }

// RIGHT — explains why this specific check exists
// isWritable probes the directory with a throwaway file because hostnamectl
// and timedatectl both fail with a confusing error if /run is read-only.
func isWritable(dir string) bool { ... }
```

### Imports

Three groups, each alphabetically sorted, separated by a blank line:
1. Standard library
2. Third-party
3. Internal (`github.com/offline-lab/bootconf/...`)

### Error messages

Use `fmt.Errorf("verb noun: %w", err)` for error wrapping. Include the specific path or value.
Do not start error strings with a capital letter or end with a period.

```go
// CORRECT
return fmt.Errorf("write sysusers config %s: %w", path, err)

// WRONG
return fmt.Errorf("Failed to write sysusers config. %w", err)
```

---

## Systemd Integration

Bootconf sends `sd_notify` signals when run as a systemd service:

- `STATUS=Applying boot configuration`: emitted before modules run
- `READY=1`: emitted after all modules complete successfully

Use `Type=notify` in the unit file. The notify call is a no-op outside systemd,
so there is no conditional wrapping needed.

---

## Key Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `gopkg.in/yaml.v3` | YAML config parsing |
| `github.com/coreos/go-systemd/v22/daemon` | systemd sd_notify |

---

## Important Notes

- Go 1.24.0, target arm64 Linux, dev on arm64 macOS.
- Version info is injected at build time via LDFLAGS into `cmd/bootconf/commands`.
- Status file is written to `<bootconf.directory>/status.json` with no subdirectory.
- `make test` runs `go test -v -race ./...`; all tests must pass with the race detector.
