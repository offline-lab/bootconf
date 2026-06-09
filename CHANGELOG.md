# Changelog

All notable changes to bootconf will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.1.0] - 2026-06-09

Initial release.

### Added

- **CLI commands**: `run`, `validate`, `status`, `check`, `version`
- **sudo module removed**: Sudo access is now managed by adding users to the
  `sudo` group via systemd-sysusers `m` line in the users module. The `sudo`
  config section, `SudoConfig` struct, and `internal/module/sudo/` package have
  been removed. Users with `sudo: true` get `m <name> sudo` in their sysusers
  config. Users with `sudo: false` or who are disabled get `gpasswd -d <name> sudo`.
- **users module**: Now generates `m <name> sudo` sysusers line for sudo users,
  calls `gpasswd -d <name> sudo` when removing disabled users
- **system module**: Set hostname (`hostnamectl`) and timezone (`timedatectl`)
- **ssh module**: Host key generation for dropbear and openssh, SSH enable/disable sentinel
- **wifi module**: `wpa_supplicant.conf` generation from pre-hashed PSK, WiFi sentinel
- **services module**: Per-service sentinel files, optional default config file copy
- **users module**: systemd-sysusers config generation, home directory creation,
  `authorized_keys` management, sudo group membership, user removal on disable
- **files module**: Arbitrary file copy with chmod, `.new` suffix for existing files
- **Config validation**: Strict validation of all paths, usernames, service names,
  SSID format, PSK hash format, country codes, hostnames, timezones
- **Security hardening**: Path traversal prevention, setuid/setgid chmod rejection,
  username leading-hyphen rejection, sequential module execution
- **Status tracking**: JSON status file written after every run, readable via `status` command
- **Dry-run mode**: `--dry-run` flag on `run` command
- **Section targeting**: `--section` flag to run a single module
- **E2E tests**: Docker-based end-to-end test suite (26 assertions)
- **Unit tests**: 83+ tests with race detector across all packages
