# Bootconf — Task List

> This file tracks all tasks for the bootconf project.
> Update on every change. Keep ready for handover.

Status: `pending` | `in_progress` | `blocked` | `done`

---

## Phase 0: Planning

| # | Task | Status | Notes |
|---|------|--------|-------|
| P0-1 | Answer design Q&A (`questions.md`) | done | All questions answered |
| P0-2 | Finalize design document | done | See `design.md` |
| P0-3 | Record all decisions in `decisions.md` | done | All decisions recorded |
| P0-4 | Inventory system dependencies | done | See dependency table below |
| P0-5 | Inventory systemd unit file changes | done | See unit file table below |

---

## Phase 1: Project Setup

| # | Task | Status | Notes |
|---|------|--------|-------|
| P1-1 | Initialize Go module `github.com/offline-lab/bootconf` | done | Go 1.24.0 |
| P1-2 | Define CLI structure with cobra | done | Subcommands: status, run, validate, check, version, help |
| P1-3 | Define config struct + YAML parsing | done | All sections have `enabled` field |
| P1-4 | Implement Go struct validation with clear error messages | done | Per-section validators |
| P1-5 | Set up logging: stdout/stderr to journald | done | Format: `bootconf: <time> <severity> <section::name> <message>` |
| P1-6 | Set up status file read/write | done | `/data/.bootconf/` directory, JSON format |
| P1-7 | Set up test infrastructure | done | Table-driven unit tests + integration tests |
| P1-8 | Create Makefile + build pipeline | done | Matches disco conventions |
| P1-9 | Add `--config` flag for arbitrary config path | done | Default: `/boot/firmware/bootconf.yaml` |
| P1-10 | Add `--verbose` flag for debug logging | done | Sets logging level to DEBUG |

---

## Phase 2: Modules

| # | Task | Status | Notes |
|---|------|--------|-------|
| P2-1 | Module: system (hostname, timezone) | done | Uses `hostnamectl`, `timedatectl` |
| P2-2 | Module: ssh (host key gen, service enable) | done | Supports dropbear + openssh via `ssh.daemon` field |
| P2-3 | Module: wifi (wpa_supplicant.conf, service enable) | done | `password_hash` only, no plaintext |
| P2-4 | Module: services (sentinel files, config copy) | done | `default_config.copy/source/destination` |
| P2-5 | Module: users (sysusers, homedir, authorized_keys, sudoers) | done | UID starts at 2000; shell always /bin/bash |
| P2-6 | Module: files (copy files/dirs into place) | done | Existing files copied as `.new` suffix |

---

## Phase 3: CLI & Integration

| # | Task | Status | Notes |
|---|------|--------|-------|
| P3-1 | `bootconf run` (full + `--section` + `--dry-run`) | done | dry-run tests everything, writes nothing |
| P3-2 | `bootconf status` (full + `--section` + `--failed` + `--full`) | done | Reads from `<basedir>/.bootconf/` |
| P3-3 | `bootconf validate` | done | Pure Go validation, offline-capable |
| P3-4 | `bootconf check` (runtime service status) | done | Checks systemctl, pgrep, id |
| P3-5 | `bootconf version` | done | |
| P3-6 | `bootconf help` | done | |
| P3-7 | Parallel execution (goroutines per section) | done | All sections run in parallel unless excluded |
| P3-8 | Exclude sections via `bootconf.exclude` | done | Replaces former `order` field |

---

## Phase 4: Testing & Documentation

| # | Task | Status | Notes |
|---|------|--------|-------|
| P4-1 | Unit tests for all modules | done | 73 tests across 11 packages |
| P4-2 | Integration tests | done | Dry-run pipeline + real wifi/services/users/files |
| P4-3 | Documentation (Diataxis framework) | pending | Deferred |
| P4-4 | Build instructions in README | done | README.md with build, usage, config, architecture |

---

## System Dependency Inventory

Tools required on the target system at runtime:

| Dependency | Used by | Required | Notes |
|------------|---------|----------|-------|
| `dropbearkey` | ssh module | If `ssh.daemon: dropbear` | Generates dropbear host keys |
| `ssh-keygen` | ssh module | If `ssh.daemon: openssh` | Generates openssh host keys |
| `hostnamectl` | system module | Always | Sets hostname |
| `timedatectl` | system module | Always | Sets timezone |
| `systemd-sysusers` | users module | Always | Creates users from sysusers.d config |
| `userdel` | users module | Always | Removes disabled users |
| `sudo` | users module | If users have `sudo: true` | Required for sudoers.d to take effect |
| `systemd` | all | Always | Init system, journal, unit management |
| ~~`wpa_passphrase`~~ | ~~wifi~~ | **Not needed** | Removed: only `password_hash` used |

---

## External Builder Tasks

These must be done in the OS builder (separate repo) to support bootconf:

| # | Task | Details | Status |
|---|------|---------|--------|
| B1 | Create `/etc/sudoers.d/include.conf` | Include `@includedir /data/config/sudoers.d` so bootconf can write sudoers configs to writable path | pending |
| B2 | Create `/data/config/sudoers.d/` | Directory with correct permissions (750, root:root) | pending |
| B3 | Configure systemd-sysusers to read `/data/config/sysusers.d/*` | Set `--root=/data/config` or equivalent so sysusers picks up our generated configs | pending |
| B4 | Create `/data/config/sysusers.d/` | Directory with correct permissions | pending |
| B5 | Install `userdel` | Required for removing disabled users (systemd-sysusers cannot remove users) | pending |
| B6 | Create `/data/config/services/` | Directory for sentinel files | pending |
| B7 | Create `/data/.bootconf/` | Directory for status files | pending |
| B8 | Create `/data/home/` | Base directory for user homes | pending |
| B9 | Create `/data/config/ssh/` | Directory for SSH host keys | pending |
| B10 | Create `/data/config/wifi/` | Directory for wpa_supplicant.conf | pending |
| B11 | Condition `ConditionPathExists=/boot/firmware/bootconf.yaml` | Add to bootconf.service unit so it silently skips if no config | pending |

---

## Systemd Unit File Changes

All service unit files that bootconf manages need:

```ini
[Unit]
After=bootconf.service
Requires=bootconf.service
ConditionPathExists=/data/config/services/<service-name>
```

Services that need this treatment (will grow):

| Service | Unit file location | Sentinel file | Status |
|---------|-------------------|---------------|--------|
| wifi | TBD | `<basedir>/services/wifi` | pending (builder repo) |
| ssh (dropbear/openssh) | TBD | `<basedir>/services/ssh` | pending (builder repo) |
| disco | TBD | `<basedir>/services/disco` | pending (builder repo) |

> Unit files are maintained in the builder repo, not here.

---

## Resolved Blockers

| Blocker | Resolution |
|---------|------------|
| Writable /etc/sudoers.d | Builder creates include.conf pointing to /data/config/sudoers.d/ |
| Writable /etc/sysusers.d | Builder configures systemd-sysusers to read /data/config/sysusers.d/ |
| SSH implementation | Support both dropbear and openssh via `ssh.daemon` field |
| wpa_passphrase dependency | Eliminated: only password_hash used |
| authorized_keys management | Full file ownership, no markers |
| Build/packaging | Makefile + go build, reference disco project |
