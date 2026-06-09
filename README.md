# bootconf

Bootconf is a little Go CLI utility
that configures a Linux system at boot time from a single YAML file.

It runs before any other service starts, placing configuration files,
SSH host keys, user accounts, and service sentinels into place.

Because it executes on every boot (not just the first), configuration
changes take effect on the next reboot without reimaging the device.


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

### Cross-compile for arm64 Linux

From an arm64 macOS or any other host:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
  -ldflags="-s -w -X github.com/offline-lab/bootconf/internal/version.Version=v0.1.0" \
  -o bootconf cmd/bootconf/main.go
```

### Integration into a Buildroot / Yocto image

Copy the binary to `/usr/local/bin/bootconf` and add a systemd service that runs
it before networking and user services start:

```ini
# /etc/systemd/system/bootconf.service
[Unit]
Description=Apply boot configuration
Before=network-pre.target
Wants=network-pre.target
DefaultDependencies=no
After=local-fs.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/bootconf run
RemainAfterExit=yes

[Install]
WantedBy=sysinit.target
```

Enable it:

```bash
systemctl enable bootconf.service
```

### Uninstall

```bash
make uninstall                  # removes /usr/local/bin/bootconf
make PREFIX=/usr uninstall      # removes /usr/bin/bootconf
```

## Configure

Bootconf reads `/boot/firmware/bootconf.yaml` by default. Override with
`--config` / `-c` on any subcommand.

See [`bootconf.yaml`](bootconf.yaml) for a fully commented example config.

### Minimal config

```yaml
bootconf:
  enabled: true
  directory: /data/bootconf

system:
  enabled: true
  hostname: mydevice
  timezone: UTC

ssh:
  enabled: true
  directory: /data/config/ssh
  daemon: dropbear
  keytype: ed25519
  generate_host_keys: true

wifi:
  enabled: true
  directory: /data/config/wifi
  ssid: MyNetwork
  password_hash: <64-char hex PSK hash>
  country: NL

services:
  enabled: true
  directory: /data/config/services
  services: []

users:
  enabled: true
  directory: /data/config/users
  users:
    - name: admin
      enabled: true
      sudo: true
      home: /data/home/admin
      authorized_keys:
        - "ssh-ed25519 AAAA... admin@host"

files:
  enabled: true
  files: []
```

### Generating a WiFi PSK hash

Bootconf expects the WiFi password as a pre-computed WPA2 PSK hash (64 hex
characters), not the plaintext password. This avoids storing plaintext
passwords in the config file.

Generate it with `wpa_passphrase`:

```bash
wpa_passphrase MyNetwork
# then type your password and press Enter
```

Output:

```
network={
        ssid="MyNetwork"
        psk=614b0b8c3b6c5e8a7d9f2a1c4e3f5d7b9a8c6e4f2d1b3a5c7e9f0d2b4a6c8e0
}
```

Copy the `psk=` value (the 64-character hex string) into `password_hash`:

```yaml
wifi:
  ssid: MyNetwork
  password_hash: 614b0b8c3b6c5e8a7d9f2a1c4e3f5d7b9a8c6e4f2d1b3a5c7e9f0d2b4a6c8e0
  country: NL
```

If `wpa_passphrase` is not installed on your system:

```bash
# Debian/Ubuntu
sudo apt install wpasupplicant

# Alpine
sudo apk add wpa_supplicant

# macOS (not available via brew — use a Linux host or Docker)
docker run --rm -it debian:bookworm-slim bash -c \
  'apt update && apt install -y wpasupplicant && wpa_passphrase MyNetwork'
```

### Sections reference

| Section | Purpose | Key files created |
|---------|---------|-------------------|
| `bootconf` | Master switch, status directory | `<directory>/.bootconf/status.json` |
| `system` | Sets hostname and timezone | N/A (calls `hostnamectl`/`timedatectl`) |
| `ssh` | Host key generation, daemon sentinel | `<ssh.directory>/hostkey`, `<services.directory>/ssh` |
| `wifi` | `wpa_supplicant.conf` generation | `<wifi.directory>/wpa_supplicant.conf`, `<services.directory>/wifi` |
| `services` | Sentinel files, optional config copy | `<services.directory>/<name>`, `<destination>` |
| `users` | User accounts, SSH keys, sudo group | `<users.directory>/<user>.conf`, `<home>/.ssh/authorized_keys` |
| `files` | Arbitrary file copy | As specified per entry |

### Per-module enable/disable

Every section has an `enabled` field. Set `bootconf.enabled: false` to disable
the entire tool. Individual modules can be disabled independently:

```yaml
wifi:
  enabled: false
```

When a module is disabled, bootconf reverses its effects where possible (removes
sentinel files, removes sysusers config). When the whole tool is disabled, it
exits immediately without touching anything.

### Users and sudo

Users are created via `systemd-sysusers`. For each user with `sudo: true`,
bootconf adds a group membership line:

```
u admin 2000 "admin" /data/home/admin /bin/bash
m admin sudo
```

The `sudo` group must be configured in `/etc/sudoers` by the OS image (typically
`%sudo ALL=(ALL) ALL`). Bootconf does not manage sudoers rules — that belongs
in the OS builder.

### SSH host keys

When `ssh.generate_host_keys: true` and no host key exists, bootconf generates
one:

- **dropbear**: `dropbearkey -t <keytype> -f <ssh.directory>/hostkey`
- **openssh**: `ssh-keygen -t <keytype> -f <ssh.directory>/hostkey -N ""`

Existing host keys are never overwritten.

### Services and sentinel files

Each service entry creates an empty sentinel file at
`<services.directory>/<name>` when enabled, and removes it when disabled.
Service scripts check for this file to decide whether to start.

Optional `default_config` copies a default config file to a writable location
on first boot. If the destination already exists, the copy is placed alongside
with a `.new` suffix instead.

### Files module

Copies arbitrary files from source to destination. If the destination exists,
writes to `<destination>.new` instead (never overwrites). All copied files are
owned by `root:root` with configurable `chmod` (default `640`).

## Usage

### run

Apply the full configuration:

```bash
bootconf run                    # apply all sections
bootconf run --dry-run          # preview without changes
bootconf run --section wifi     # only run the wifi module
bootconf run -c /path/to.yaml  # use a different config file
```

If the config file does not exist, `run` exits silently with code 0 (nothing to do).

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

Verify runtime state against the config file. Checks that configured services
are active and users exist:

```bash
bootconf check
```

### version

Print build version, commit, and timestamp:

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
make test-e2e         # Run Docker-based end-to-end tests
```

## Project structure

```
cmd/bootconf/             CLI entry point
  commands/               Cobra subcommands (run, validate, status, check)
internal/
  config/                 YAML config loading, defaults, and validation
  logging/                Structured section-aware logging
  module/                 Module interface and sequential runner
    system/               Hostname and timezone
    ssh/                  Host key generation (dropbear / openssh)
    wifi/                 wpa_supplicant.conf generation
    services/             Service sentinel files and config copy
    users/                User accounts, SSH keys, home directories, sudo group
    files/                Arbitrary file copy
  status/                 Run status persistence (JSON)
  version/                Build version variables (injected via LDFLAGS)
test/
  integration_test.go     Full pipeline tests with real filesystem
  e2e/                    Docker-based end-to-end tests
```

## Architecture

Each configuration section is implemented as a `Module` that satisfies a
two-method interface:

```go
type Module interface {
    Name() string
    Run(ctx context.Context, dryRun bool) Result
}
```

The `Runner` executes all registered modules sequentially. If a section fails,
the remaining sections continue. Results are written to
`<bootconf.directory>/.bootconf/status.json` after every run, which
`bootconf status` reads back.

Modules can be targeted individually with `--section`.

## Security

Bootconf runs as root at boot time. The following hardening measures are in place:

- **Path traversal prevention**: All directory and file paths are validated to be
  absolute, clean (no `.`/`..` components), and free of traversal sequences.
- **Service name validation**: Names must match `^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`.
- **Username validation**: Lowercase letters, digits, underscores, hyphens only.
  Leading hyphens are rejected to prevent flag injection.
- **File permissions**: Secrets (wpa_supplicant.conf, authorized_keys) use 0600.
  WiFi config directory uses 0700.
- **Chmod restrictions**: The files module rejects setuid/setgid/sticky bits.
- **WiFi PSK validation**: Password hashes must be exactly 64 hex characters.
- **SSID validation**: SSIDs are validated for printable characters, max 32 bytes.
- **Country code validation**: Must be 2 uppercase letters (ISO 3166-1).

### Threat model

The config file lives at `/boot/firmware/bootconf.yaml`. If an attacker can
modify this file (e.g., physical access to the firmware partition), they control
what bootconf does with root privileges. To mitigate:

- Mount the firmware partition with restrictive permissions (`fmask=0177`).
- Consider integrity verification (e.g., signed config) for production deployments.
- Users with `sudo: true` are added to the `sudo` group via systemd-sysusers.
  The `sudo` group must be configured in `/etc/sudoers` (handled by the OS builder).
