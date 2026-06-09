# Bootconf — Design Document

> Living document. Update as decisions are made.
> Keep ready for handover.

---

## Overview

Bootconf is a single Go binary that configures a readonly Linux OS at boot
time. It reads a YAML configuration file (default `/boot/firmware/bootconf.yaml`)
and creates/updates configuration files on a writable data partition before
any other service starts.

- Runs **every boot**, not just first boot.
- **Not** a service manager — it prepares config so systemd services can start.
- Portable: not tied to a specific distro. Uses a configurable `basedir`.
- Target: arm64 (Buildroot), but architecture-agnostic code.

---

## Architecture

```
bootconf.yaml (FAT32 /boot/firmware or --config path)
        |
        v
   +-----------+
   | bootconf  |  (Go binary, runs as early systemd unit)
   +-----------+
        |
        | reads config, writes to <basedir>/*
        |
        v
   systemd services start (ConditionPathExists gates)
```

### Module structure

Each YAML section maps to a Go module. Modules are independent and
self-contained.

```
internal/
  module/
    system/     -- hostname, timezone
    ssh/        -- host key generation (dropbear + openssh), service enable
    wifi/       -- wpa_supplicant.conf from password_hash, service enable
    services/   -- sentinel files, default config copy
    users/      -- sysusers.d, homedirs, authorized_keys, sudoers
    files/      -- arbitrary file/directory copy
```

Each module implements a common interface:

```go
type Module interface {
    Name() string
    Run(ctx context.Context, cfg Config, dryRun bool) Result
}
```

---

## Filesystem Layout

All paths are relative to `<basedir>` (default: `/data/config`).
The `basedir` is configured in `bootconf.basedir` in the YAML.

```
/boot/firmware/
  bootconf.yaml                  # Input config (FAT32, user-editable)

<basedir>/  (default /data/config)
  services/                      # Sentinel files for systemd
    <name>                       #   exists = service enabled
  wifi/
    wpa_supplicant.conf          # Generated wifi config
  ssh/
    hostkey                      # SSH host key (dropbear format)
    hostkey.pub                  # SSH host public key
  sysusers.d/                    # systemd-sysusers configs
    <name>.conf                  #   builder configures systemd-sysusers to read from here
  sudoers.d/                     # sudo configs
    <name>                       #   builder creates /etc/sudoers.d/include.conf pointing here
  <service>/                     # Per-service config dirs
    <service>.conf
    <service>.conf.new           # If file exists, bootconf writes .new suffix

/data/
  .bootconf/                     # Status from last run (directory)
    status.json                  #   JSON format

  home/                          # User home directories
    <name>/
      .ssh/
        authorized_keys          # Fully managed by bootconf
```

---

## Configuration Format

See `bootconf.yaml` in repo root for the full example.

### Top-level sections

| Section | Required | Purpose |
|---------|----------|---------|
| `bootconf` | yes | Tool behavior: basedir, exclude list |
| `system` | no | Hostname, timezone |
| `ssh` | no | SSH host key generation, daemon type, enable/disable |
| `wifi` | no | WiFi SSID, password_hash, country |
| `services` | no | List of services to enable/configure |
| `users` | no | User accounts, SSH keys, sudo access |
| `files` | no | Arbitrary file copies |

### Default behavior

- All sections default to `enabled: false` unless explicitly set.
- If a section is absent from the YAML, it is treated as disabled.
- With `--verbose`, bootconf logs each disabled section.
- If the config file doesn't exist: silent exit 0 (unit file has ConditionPathExists).

### Execution model

- All enabled sections run **in parallel** (goroutines).
- Sections listed in `bootconf.exclude` are not run.
- No ordering — no `order` field.
- A failing section does not block other sections.

### Key config changes (from Q&A)

| Change | Before | After |
|--------|--------|-------|
| Execution order | `bootconf.order` array | `bootconf.exclude` array; all others run in parallel |
| Wifi password | `password` + `password_hash` | `password_hash` only |
| SSH daemon | Implicit dropbear | Explicit `ssh.daemon: dropbear\|openssh` |
| Users | No enable/disable | `users[].enabled: true\|false` |
| Service config copy | `copy_default_config: true` | `default_config: {copy, source, destination}` |
| File destinations | Absolute paths | Relative to `basedir` |
| Existing file conflict | Skip | Write as `.new` suffix |

---

## CLI

```
bootconf help                        Show usage
bootconf version                     Show build version
bootconf validate                    Validate config (offline, no network)
bootconf validate --config <path>    Validate a specific config file
bootconf check                       Check runtime status of configured services
bootconf run                         Execute all enabled sections
bootconf run --config <path>         Use a specific config file
bootconf run --dry-run               Test everything, write nothing
bootconf run --section <name>        Run only one section
bootconf run --verbose               Log disabled sections and detailed info
bootconf status                      Show last run status
bootconf status --section <name>     Show status for one section
bootconf status --failed             Show only failures
bootconf status --full               Show detailed trace per section
```

### Default config path

- Default: `/boot/firmware/bootconf.yaml`
- Override with `--config <path>` on any subcommand.

### Missing config file

- Silent exit 0 — the systemd unit uses `ConditionPathExists` to guard.

### Empty / minimal config

- All sections default to `enabled: false`.
- With `--verbose`: log each section as "not enabled".

---

## Module Details

### system

- Sets hostname via `hostnamectl set-hostname`
- Sets timezone via `timedatectl set-timezone`
- Requires: `hostnamectl`, `timedatectl`

### ssh

- Supports `daemon: dropbear` and `daemon: openssh`
- If `generate_host_keys: true`:
  - dropbear: `dropbearkey -t <keytype> -f <basedir>/ssh/hostkey`
  - openssh: `ssh-keygen -t <keytype> -f <basedir>/ssh/hostkey`
- Only generates if host key doesn't already exist
- Creates/removes sentinel file `<basedir>/services/ssh`

### wifi

- Creates `<basedir>/wifi/wpa_supplicant.conf` using `password_hash`
- No plaintext passwords — no `wpa_passphrase` dependency
- Creates/removes sentinel file `<basedir>/services/wifi`

### services

- For each service:
  - `enabled: true` → create `<basedir>/services/<name>`
  - `enabled: false` → remove `<basedir>/services/<name>`
  - If `default_config.copy: true` → copy `source` to `destination` (relative to basedir)
    - If destination exists → write as `<destination>.new` (never overwrite)

### users

- For each user:
  - `enabled: true`:
    - Write sysusers.d config to `<basedir>/sysusers.d/<name>.conf`
      - UID starts at 2000, auto-increment
      - Shell: `/bin/bash`
    - If `sudo: true` → write `<basedir>/sudoers.d/<name>` (password required, no NOPASSWD)
    - Create homedir `/data/home/<name>` with correct ownership
    - Create `/data/home/<name>/.ssh/` (perms 700, owned by user)
    - Write `/data/home/<name>/.ssh/authorized_keys` (perms 600, owned by user)
      - Full file ownership — bootconf overwrites entire file
  - `enabled: false`:
    - Remove sentinel files
    - Run `userdel <name>` to remove user
- Static UID/GID required because overlay is ephemeral (users recreated each boot)

### files

- For each entry:
  - Source can be file or directory
  - Destination is relative to `basedir`
  - If destination exists → write as `<dest>.new`
  - Default permissions: 750 (dir), 640 (file), root:root

---

## Error Handling

- A failing section does **not** stop other sections from running.
- Within a section that loops over a list (services, users), a failing item
  is recorded but the loop continues.
- Within a section that is **not** a loop, failure moves to the next section.
- Every section's result (success/fail/detail) is recorded to the status file.
- Before writing: test if filesystem is writable, bail with clear error if not.
- No wait loops for volumes — handled by systemd unit ordering.

---

## Logging

- Plain stdout/stderr (captured by journald via systemd).
- Format: `bootconf: <time> <severity> <section::name> <message>`
- Example: `bootconf: 2026-06-09T08:00:00Z info section::wifi configuring SSID 'mynetwork'`
- Each section's log lines are distinguishable even under parallel execution.

---

## Dry-Run Mode

- `--dry-run` on `bootconf run`: runs all checks, validates paths and permissions,
  but does **not** create or modify any files.
- A real dry-run: test everything that would happen, report what would change.

---

## Security Considerations

- Config on FAT32 `/boot/firmware` — no Unix permissions, no encryption.
  Anyone with physical access can read/modify.
- Only `password_hash` accepted for wifi — no plaintext passwords in config.
- SSH host keys only generated if they don't already exist.
- authorized_keys is fully managed by bootconf — manual edits will be overwritten.
- sudoers configs use password-required (no NOPASSWD).
- sudoers files written to `<basedir>/sudoers.d/`, not `/etc/sudoers.d/`
  (read-only fs). Builder creates an include bridge.

---

## Build

- Module: `github.com/offline-lab/bootconf`
- Go: latest stable LTS (keep switching easy via go.mod)
- Target: arm64 (native on dev mac, cross-compile not needed currently)
- Build: Makefile + `go build` (reference disco project structure)
- Docs: build instructions in README + agent docs

---

## Testing

- Table-driven unit tests with fake filesystems (afero or similar).
- Docker for integration tests where real filesystem behavior matters.
- No QEMU — overkill for this scope.
- Install real dropbear/openssh in test env to validate key generation syntax.

---

## Open Design Questions

None — all Q&A questions answered. Decisions recorded in `decisions.md`.
