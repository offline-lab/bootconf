# Bootconf — Decision Log

> Record every design decision with rationale.
> Keep ready for handover.

Format: `| Date | Decision | Rationale | Affects |`

---

## Decisions Made

| Date | Decision | Rationale | Affects |
|------|----------|-----------|---------|
| 2026-06-08 | Go as implementation language | Specified in project.md; single binary, no runtime deps, cross-compile support | All |
| 2026-06-08 | Module-per-section architecture | Each YAML section gets its own Go package under `internal/module/`; easy to extend | Architecture |
| 2026-06-08 | Sentinel files for service enable | `<basedir>/services/<name>` existence gates systemd units via `ConditionPathExists`; works on readonly fs | Services, systemd units |
| 2026-06-08 | Status persisted to `/data/.bootconf/` | Directory for flexibility; JSON format; allows `bootconf status` without re-running | CLI, status module |
| 2026-06-08 | Non-blocking section execution | A failure in one section does not prevent others from running; matches project.md spec | Error handling, module runner |
| 2026-06-08 | Config on FAT32 `/boot/firmware` | Allows editing from Windows/macOS; trade-off: no Unix perms, no encryption | Security, config format |
| 2026-06-09 | Distro-agnostic design | Not pinned to Buildroot; configurable `basedir` makes it portable to any Linux distro | Architecture, config |
| 2026-06-09 | arm64 target, no x86_64 for now | Dev environment is arm64 mac; target hardware is arm64. Code stays arch-agnostic. | Build |
| 2026-06-09 | `--config` flag for arbitrary config path | Config may not always be at `/boot/firmware/bootconf.yaml`; allows testing and alternative setups | CLI |
| 2026-06-09 | Removed `order` field, added `exclude` | Simpler mental model: run everything in parallel, explicitly exclude what you don't want | Config, execution model |
| 2026-06-09 | All sections default to `enabled: false` | Explicit opt-in is safer than implicit opt-out; missing sections are simply no-ops | Config, error handling |
| 2026-06-09 | Removed plaintext `password`, only `password_hash` | Secure by design; no plaintext credentials in config; eliminates `wpa_passphrase` dependency | Wifi module, dependencies |
| 2026-06-09 | Support both dropbear and openssh | User may choose SSH implementation via `ssh.daemon` field; both are common in embedded | SSH module |
| 2026-06-09 | Users always get `/bin/bash` | Simple default, no configuration needed | Users module |
| 2026-06-09 | Full file ownership of authorized_keys | Skip markers — simpler code; manual changes will be wiped. Users must manage keys through config. | Users module |
| 2026-06-09 | UID starts at 2000, static allocation | Overlay is ephemeral — users recreated each boot; static UIDs ensure consistency. Apps start at 6000. | Users module |
| 2026-06-09 | Users: `enabled: false` triggers removal | Use `userdel` to remove; systemd-sysusers cannot remove users | Users module |
| 2026-06-09 | Sudoers written to `<basedir>/sudoers.d/` | Readonly `/etc/sudoers.d/` — builder creates an include.conf that bridges to `/data/config/sudoers.d/` | Users module, builder |
| 2026-06-09 | Sysusers configs in `<basedir>/sysusers.d/` | Builder configures systemd-sysusers to read from `/data/config/sysusers.d/` | Users module, builder |
| 2026-06-09 | Plain stdout/stderr logging | systemd captures service output to journald automatically; no native journal library needed | Logging |
| 2026-06-09 | Log format: `bootconf: <time> <severity> <section::name> <message>` | Includes section name for parallel execution visibility | Logging |
| 2026-06-09 | Module: `github.com/offline-lab/bootconf` | Follows Go convention; matches org repo structure | Build |
| 2026-06-09 | Go latest stable LTS | Keep go.mod version easy to change | Build |
| 2026-06-09 | Makefile + go build (reference disco) | Consistent with other projects in the org | Build |
| 2026-06-09 | Go struct validation (no external schema file) | Schema validation is internal only, not reused by other tools; clear error messages are the priority | Validation |
| 2026-06-09 | `validate` works fully offline | Boot-time tool may have no network; no downloads for validation | CLI |
| 2026-06-09 | Missing config = silent exit 0 | Unit file uses `ConditionPathExists`; no config means nothing to do | CLI, error handling |
| 2026-06-09 | Dry-run = real checks, no writes | Tests permissions, paths, validity — but creates/modifies nothing | CLI |
| 2026-06-09 | Existing files written as `.new` suffix | Never overwrite user-modified configs; they can inspect and adopt `.new` files | Files module, services module |
| 2026-06-09 | Unit tests + docker integration | QEMU overkill; docker acceptable for real fs behavior testing | Testing |
| 2026-06-09 | Service unit files maintained in builder repo | Bootconf only creates sentinel files; unit file modifications are a separate concern | Scope |

---

## Closed Decisions (from Q&A)

| ID | Question | Decision | Date |
|----|----------|----------|------|
| D1 | Writable /etc paths | Write to `<basedir>/sudoers.d/` and `<basedir>/sysusers.d/`; builder bridges with include/config | 2026-06-09 |
| D2 | SSH implementation | Support both dropbear and openssh via `ssh.daemon` field | 2026-06-09 |
| D3 | authorized_keys management | Full file ownership, no markers | 2026-06-09 |
| D4 | Status file format | JSON in `/data/.bootconf/` directory | 2026-06-09 |
| D5 | Config validation | Go struct validation with clear error messages, no external schema | 2026-06-09 |
| D6 | Journal logging method | Plain stdout/stderr (systemd captures to journal) | 2026-06-09 |
| D7 | Build system | Makefile + go build, reference disco project | 2026-06-09 |
| D8 | CLI framework | cobra (standard Go CLI framework) | 2026-06-09 |
| D9 | Testing strategy | Table-driven unit tests + docker integration | 2026-06-09 |
| D10 | Parallel implementation | Goroutines per section | 2026-06-09 |

---

## Open Decisions

None — all Q&A questions resolved.

---

## Implementation Plan

Saved to `docs/plans/2026-06-09-implementation.md`. Covers 13 tasks:

1. Project scaffold (go.mod, Makefile, cobra CLI)
2. Config struct + YAML parsing + validation
3. Logging package
4. Module interface + parallel runner + status tracking
5. System module (hostname, timezone)
6. SSH module (dropbear + openssh)
7. Wifi module (wpa_supplicant.conf from hash)
8. Services module (sentinel files, config copy)
9. Users module (sysusers, homedirs, keys, sudoers)
10. Files module (copy with .new suffix)
11. CLI subcommands (run, status, validate, check)
12. Integration test
13. README + cleanup

---

## Deferred

| Topic | Reason | Revisit when |
|-------|--------|-------------|
| x86_64 support | Not needed now; code is arch-agnostic | Hardware expansion |
| OpenSSH-specific options | Only dropbear tested initially | openssh users request features |
| Config file encryption | FAT32 limitation accepted for now | Security requirements change |
| Config file migration/versioning | Single version for now | Breaking config changes |
