# bootconf

Bootconf is a declarative boot configuration tool for Linux. It reads a single
YAML file and applies system settings during early boot — before any other
service starts.

It runs on **every boot**, so configuration changes take effect on the next
reboot without reinstalling or reimaging the system.

```
hostname · timezone · SSH host keys · WiFi · users · service sentinels · files
```

No cloud dependency. No agent. No runtime daemon. Just a binary, a YAML file,
and a systemd unit.

## What it manages

| Section | What it does |
|---------|-------------|
| `system` | Sets hostname and timezone via `hostnamectl` / `timedatectl` |
| `ssh` | Generates SSH host keys (dropbear or openssh), manages service sentinel |
| `wifi` | Writes `wpa_supplicant.conf` from a pre-hashed WPA2 PSK |
| `services` | Creates/removes sentinel files; optionally copies a default config on first boot |
| `users` | Provisions user accounts via `systemd-sysusers`, sets up SSH authorized keys and sudo group membership |
| `files` | Copies arbitrary files from source to destination (never overwrites existing content) |
| `templates` | Renders Go `text/template` files with config-supplied variables |
| `shell` | Executes shell commands at boot, capturing stdout/stderr/exit code to log files |
| `unitrun` | Writes shell scripts and systemd units, enables them via `systemctl` |

## Install

### From source

Requires Go 1.24+.

```bash
make
make install                  # installs to /usr/local/bin/bootconf
make PREFIX=/usr install      # installs to /usr/bin/bootconf
```

Build a release binary with version info baked in:

```bash
make VERSION=v0.1.0 all
```

### Cross-compile

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
  -ldflags="-s -w -X github.com/offline-lab/bootconf/cmd/bootconf/commands.Version=v0.1.0" \
  -o bootconf cmd/bootconf/main.go
```

### Uninstall

```bash
make uninstall                  # removes /usr/local/bin/bootconf
make PREFIX=/usr uninstall      # removes /usr/bin/bootconf
```

## Quick start

### 1. Install the binary

Place `bootconf` in `/usr/local/bin/` on the target system.

### 2. Create a config file

Bootconf reads `/boot/firmware/bootconf.yaml` by default. Override with
`--config` / `-c` on any subcommand.

A minimal config:

```yaml
bootconf:
  enabled: true
  directory: /var/lib/bootconf

system:
  enabled: true
  hostname: myhost
  timezone: UTC

ssh:
  enabled: true
  directory: /etc/bootconf/ssh
  daemon: openssh
  keytype: ed25519
  generate_host_keys: true

users:
  enabled: true
  directory: /etc/bootconf/users
  users:
    - name: admin
      enabled: true
      sudo: true
      authorized_keys:
        - "ssh-ed25519 AAAA... admin@workstation"
```

See [`bootconf.yaml`](bootconf.yaml) for a fully commented reference config.

### 3. Add a systemd unit

```ini
# /etc/systemd/system/bootconf.service
[Unit]
Description=Apply boot configuration
Before=network-pre.target
Wants=network-pre.target
DefaultDependencies=no
After=local-fs.target

[Service]
Type=notify
ExecStart=/usr/local/bin/bootconf run
RemainAfterExit=yes

[Install]
WantedBy=sysinit.target
```

```bash
systemctl enable bootconf.service
```

`Type=notify` lets systemd track exact completion — bootconf sends `READY=1`
via sd_notify after all modules finish.

## Configure

### Sections reference

| Section | Key fields | Files written |
|---------|-----------|---------------|
| `bootconf` | `enabled`, `directory` | `<directory>/status.json` |
| `system` | `hostname`, `timezone` | N/A (calls system tools) |
| `ssh` | `daemon`, `keytype`, `generate_host_keys`, `directory` | `<ssh.directory>/hostkey`, `<services.directory>/ssh` |
| `wifi` | `ssid`, `password_hash`, `country`, `directory` | `<wifi.directory>/wpa_supplicant.conf`, `<services.directory>/wifi` |
| `services` | `directory`, `services[]` | `<services.directory>/<name>` |
| `users` | `directory`, `users[]` | `<users.directory>/<user>.conf`, `<home>/.ssh/authorized_keys` |
| `files` | `files[]` | As configured per entry |
| `templates` | `templates[]` | As configured per entry (`.new` suffix if destination exists) |
| `shell` | `directory`, `commands[]` | `<directory>/<name>.log`, `<directory>/<name>.firstboot` |
| `unitrun` | `directory`, `firstboot`, `units[]` | `<directory>/<name>.sh`, `/etc/systemd/system/bootconf-<name>.service` |

### Per-module enable/disable

Every section has an `enabled` field. Set `bootconf.enabled: false` to disable
the entire tool. Individual modules can be disabled independently:

```yaml
wifi:
  enabled: false
```

When a module is disabled, bootconf reverses its effects where possible —
removing sentinel files, removing sysusers config. When the whole tool is
disabled, it exits immediately without touching anything.

### Users and sudo

Users are provisioned via `systemd-sysusers`. For each user with `sudo: true`,
bootconf adds a group membership directive:

```
u admin 2000 "admin" /home/admin /bin/bash
m admin sudo
```

The `sudo` group must exist and be configured in `/etc/sudoers` (typically
`%sudo ALL=(ALL) ALL`). Bootconf manages group membership, not sudoers rules.

If `home` is not set, it defaults to `/home/<username>`.

### SSH host keys

When `ssh.generate_host_keys: true` and no host key exists yet, bootconf
generates one on the first boot:

- **dropbear**: `dropbearkey -t <keytype> -f <ssh.directory>/hostkey`
- **openssh**: `ssh-keygen -t <keytype> -f <ssh.directory>/hostkey -N ""`

Existing host keys are never overwritten.

### Service sentinel files

Each service entry creates an empty file at `<services.directory>/<name>` when
enabled, and removes it when disabled. Your init scripts or systemd units use
`ConditionPathExists=` to decide whether to start.

The optional `default_config` block copies a default config file to a writable
location. If the destination already exists, the new content is placed alongside
it with a `.new` suffix — existing config is never overwritten.

### Generating a WiFi PSK hash

Bootconf expects the WiFi password as a pre-computed WPA2 PSK hash (64 hex
characters), not plaintext. Generate it with `wpa_passphrase`:

```bash
wpa_passphrase MyNetwork
# type your password and press Enter
```

Copy the `psk=` value (64 hex characters) into `password_hash`:

```yaml
wifi:
  ssid: MyNetwork
  password_hash: 614b0b8c3b6c5e8a7d9f2a1c4e3f5d7b9a8c6e4f2d1b3a5c7e9f0d2b4a6c8e0
  country: NL
```

## Usage

### run

Apply the full configuration:

```bash
bootconf run                    # apply all sections
bootconf run --dry-run          # preview without changes
bootconf run --section wifi     # only run the wifi module
bootconf run -c /path/to.yaml   # use a different config file
```

If the config file does not exist, `run` exits silently with code 0.

### validate

Check config syntax and schema without touching the system:

```bash
bootconf validate
```

### status

Show results from the last `run`:

```bash
bootconf status              # summary
bootconf status --failed     # only failed sections
bootconf status --full       # all details
bootconf status --section ssh
```

### check

Verify runtime state against the config file — checks that configured services
are active and users exist:

```bash
bootconf check
```

### version

```bash
bootconf version
```

## Build

```bash
make                  # Build binary to build/bin/bootconf
make test             # Run tests with race detection
make lint             # Run golangci-lint
make fmt              # Format Go code
make clean            # Remove build artifacts
```

## Project structure

```
cmd/bootconf/
  main.go               Entry point
  commands/             Cobra subcommands (run, validate, status, check)
    version.go          Build vars injected via LDFLAGS
internal/
  config/               YAML config loading, defaults, and validation
  logging/              Structured leveled logger (INFO/WARN/ERROR/DEBUG)
  module/               Module interface, concurrent runner, per-module packages
    system/             Hostname and timezone
    ssh/                SSH host key generation (dropbear / openssh)
    wifi/               wpa_supplicant.conf generation
    services/           Service sentinel files and config copy
    users/              User accounts, SSH keys, home directories, sudo group
    files/              Arbitrary file copy
  run/                  exec.CommandContext wrapper used by all modules
  status/               Run status read/write (status.json)
test/
  integration_test.go   Full pipeline tests with real filesystem
```

## Architecture

Each configuration section is implemented as a `Module`:

```go
type Module interface {
    Name() string
    Run(ctx context.Context, dryRun bool) Result
}
```

The `Runner` executes all registered modules **concurrently** and collects
results in declaration order. If a section fails, the remaining sections
continue running. Results are written to `<bootconf.directory>/status.json`
after every run, which `bootconf status` reads back.

Modules can be targeted individually with `--section`.

## Security

Bootconf runs as root at boot time. The following hardening measures are in place:

- **Path traversal prevention**: All paths are validated to be absolute, clean,
  and free of traversal sequences.
- **Service name validation**: Names must match `^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`.
- **Username validation**: Lowercase letters, digits, underscores, hyphens only.
  Leading hyphens are rejected to prevent flag injection.
- **File permissions**: Secrets (`wpa_supplicant.conf`, `authorized_keys`) use 0600.
  WiFi config directory uses 0700.
- **Chmod restrictions**: The files module rejects setuid/setgid/sticky bits.
- **WiFi PSK validation**: Password hashes must be exactly 64 hex characters.
- **SSID validation**: Printable characters only, max 32 bytes.
- **Country code validation**: Must be 2 uppercase letters (ISO 3166-1).

### Threat model

The config file lives at `/boot/firmware/bootconf.yaml` by default. An attacker
with write access to that file controls what bootconf does with root privileges.
To mitigate:

- Mount the config partition with restrictive permissions.
- Consider integrity verification (e.g., signed config) for production deployments.
