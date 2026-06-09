# Config Restructuring Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refactor all config structs and modules so every path is explicitly configurable per-section instead of derived from a single basedir. Add the new `sudo` section. Remove the basedir-derived path logic entirely.

**Architecture:** Each section owns its own directory path. Sentinel files all go to `services.directory`. The `bootconf.directory` is only for status persistence. A new standalone `sudo` module replaces the user-embedded sudo logic.

---

## Diff Summary

### Config struct changes

| Before | After | Notes |
|--------|-------|-------|
| `BootconfConfig.Basedir` | `BootconfConfig.Directory` | Status only, renamed |
| — | `BootconfConfig.Enabled` | Master switch |
| — | `SSHConfig.Directory` | Host key location |
| — | `WifiConfig.Directory` | wpa_supplicant.conf location |
| `Config.Services []ServiceEntry` | `Config.Services ServicesConfig` | Struct with `Enabled`, `Directory`, `Services []` |
| — | `ServiceEntry.Sentinel` | Per-service sentinel control |
| `Config.Users []UserEntry` | `Config.Users UsersConfig` | Struct with `Directory`, `Users []` |
| — | New `Config.Sudo SudoConfig` | Own section with `Enabled`, `Directory`, `ExtraSettings` |
| `UserEntry.Sudo bool` | Removed | Sudo is now in the sudo section, not per-user |
| `FileEntry.Dest` | `FileEntry.Destination` | Renamed, absolute path |

### Path mapping (new)

| What | Old path | New path |
|------|----------|----------|
| Status file | `<basedir>/.bootconf/status.json` | `<bootconf.directory>/.bootconf/status.json` |
| SSH host keys | `<basedir>/ssh/hostkey` | `<ssh.directory>/hostkey` |
| SSH sentinel | `<basedir>/services/ssh` | `<services.directory>/ssh` |
| Wifi config | `<basedir>/wifi/wpa_supplicant.conf` | `<wifi.directory>/wpa_supplicant.conf` |
| Wifi sentinel | `<basedir>/services/wifi` | `<services.directory>/wifi` |
| Service sentinel | `<basedir>/services/<name>` | `<services.directory>/<name>` |
| Service config copy | relative to basedir | absolute `default_config.destination` |
| Sysusers config | `<basedir>/sysusers.d/<name>.conf` | `<users.directory>/<name>.conf` |
| Sudoers config | `<basedir>/sudoers.d/<name>` | `<sudo.directory>/<name>.conf` |
| User sentinel | `<basedir>/services/user-<name>` | **Removed** — no user sentinels |
| File copy | relative to basedir | absolute `destination` |

---

### Task 1: Update config structs

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Modify: `internal/config/validation.go`
- Modify: `internal/config/validation_test.go`

**Changes to `config.go`:**

```go
type Config struct {
    Bootconf BootconfConfig  `yaml:"bootconf"`
    System   SystemConfig    `yaml:"system"`
    SSH      SSHConfig       `yaml:"ssh"`
    Wifi     WifiConfig      `yaml:"wifi"`
    Services ServicesConfig  `yaml:"services"`
    Sudo     SudoConfig      `yaml:"sudo"`
    Users    UsersConfig     `yaml:"users"`
    Files    []FileEntry     `yaml:"files"`
}

type BootconfConfig struct {
    Enabled  bool     `yaml:"enabled"`
    Directory string  `yaml:"directory"`
    Exclude  []string `yaml:"exclude"`
}

type SSHConfig struct {
    Enabled          bool   `yaml:"enabled"`
    Directory        string `yaml:"directory"`
    Keytype          string `yaml:"keytype"`
    GenerateHostKeys bool   `yaml:"generate_host_keys"`
    Daemon           string `yaml:"daemon"`
}

type WifiConfig struct {
    Enabled      bool   `yaml:"enabled"`
    Directory    string `yaml:"directory"`
    SSID         string `yaml:"ssid"`
    PasswordHash string `yaml:"password_hash"`
    Country      string `yaml:"country"`
}

type ServicesConfig struct {
    Enabled   bool           `yaml:"enabled"`
    Directory string         `yaml:"directory"`
    Services  []ServiceEntry `yaml:"services"`
}

type ServiceEntry struct {
    Name          string        `yaml:"name"`
    Enabled       bool          `yaml:"enabled"`
    Sentinel      bool          `yaml:"sentinel"`
    DefaultConfig DefaultConfig `yaml:"default_config"`
}

type SudoConfig struct {
    Enabled       bool     `yaml:"enabled"`
    Directory     string   `yaml:"directory"`
    ExtraSettings []string `yaml:"extra_settings"`
}

type UsersConfig struct {
    Directory string      `yaml:"directory"`
    Users     []UserEntry `yaml:"users"`
}

type UserEntry struct {
    Name           string   `yaml:"name"`
    Enabled        bool     `yaml:"enabled"`
    Sudo           bool     `yaml:"sudo"`        // still needed to know which users get sudoers
    Home           string   `yaml:"home"`
    AuthorizedKeys []string `yaml:"authorized_keys"`
}

type FileEntry struct {
    Source      string `yaml:"source"`
    Destination string `yaml:"destination"`
    Chmod       string `yaml:"chmod"`
}
```

`SetDefaults()` changes:
- `Bootconf.Directory` defaults to `/data/bootconf`
- `SSH.Keytype` → `ed25519`
- `SSH.Daemon` → `dropbear`
- `FileEntry.Chmod` → `640`

**Changes to `validation.go`:**

Add validators:
- `validateBootconf()`: `directory` required when enabled
- `validateSSH()`: `directory` required when enabled (in addition to daemon/keytype)
- `validateWifi()`: `directory` required when enabled
- `validateServices()`: `directory` required when enabled
- `validateSudo()`: `directory` required when enabled
- `validateUsers()`: `directory` required when any user enabled

Remove:
- The `basedir` required check (no longer a universal concept)

**Step 1:** Update config structs, SetDefaults, Load
**Step 2:** Update tests to match new YAML structure
**Step 3:** Update validation + validation tests
**Step 4:** Run `make test`, fix all failures

---

### Task 2: Update SSH module

**Files:**
- Modify: `internal/module/ssh/ssh.go`
- Modify: `internal/module/ssh/ssh_test.go`

**Changes:**

Constructor: `New(daemon, keytype string, generateHostKeys, enabled bool, sshDir, servicesDir string)`

- Host keys: `sshDir/hostkey` instead of `basedir/ssh/hostkey`
- Sentinel: `servicesDir/ssh` instead of `basedir/services/ssh`
- All test paths updated to use explicit temp dirs

---

### Task 3: Update Wifi module

**Files:**
- Modify: `internal/module/wifi/wifi.go`
- Modify: `internal/module/wifi/wifi_test.go`

**Changes:**

Constructor: `New(enabled bool, ssid, passwordHash, country, wifiDir, servicesDir string)`

- Config: `wifiDir/wpa_supplicant.conf` instead of `basedir/wifi/wpa_supplicant.conf`
- Sentinel: `servicesDir/wifi` instead of `basedir/services/wifi`

---

### Task 4: Update Services module

**Files:**
- Modify: `internal/module/services/services.go`
- Modify: `internal/module/services/services_test.go`

**Changes:**

Constructor: `New(entries []config.ServiceEntry, servicesDir string)`

- Sentinel: `servicesDir/<name>` instead of `basedir/services/<name>`
- Config copy: `destination` is now absolute — no more `filepath.Join(basedir, dest)`, use `dest` directly
- Honor `ServiceEntry.Sentinel` — only create sentinel if true

---

### Task 5: Create Sudo module

**Files:**
- Create: `internal/module/sudo/sudo.go`
- Create: `internal/module/sudo/sudo_test.go`

**Spec:**

Constructor: `New(enabled bool, directory string, users []config.UserEntry, extraSettings []string)`

Behavior:
- `Name()` returns `"sudo"`
- If not enabled: skip
- Create `<directory>/` dir (perms 0750)
- For each user with `sudo: true` and `enabled: true`:
  - Write `<directory>/<name>.conf` with content:
    ```
    <name> ALL=(ALL) ALL
    ```
  - Perms 0440
- If `extra_settings` is not empty:
  - Write `<directory>/extra.conf` with each setting on a line
  - Perms 0440
- All writes skipped in dry-run

Tests:
- `TestSudoCreatesConfig` — user with sudo=true, file created
- `TestSudoNoSudoUsers` — no sudo users, no files
- `TestSudoExtraSettings` — extra.conf created
- `TestSudoDisabled` — disabled, nothing created
- `TestSudoDryRun` — dry-run, nothing created

---

### Task 6: Update Users module

**Files:**
- Modify: `internal/module/users/users.go`
- Modify: `internal/module/users/users_test.go`

**Changes:**

Constructor: `New(entries []config.UserEntry, uidStart int, usersDir string)`

- Sysusers: `usersDir/<name>.conf` instead of `basedir/sysusers.d/<name>.conf`
- Remove all sudoers logic (moved to sudo module)
- Remove sentinel file creation (no user sentinels)
- Keep: homedir, .ssh, authorized_keys, chown

---

### Task 7: Update Files module

**Files:**
- Modify: `internal/module/files/files.go`
- Modify: `internal/module/files/files_test.go`

**Changes:**

- `entry.Dest` → `entry.Destination` (field rename)
- Destination is absolute — no more `filepath.Join(basedir, dest)`, use directly

---

### Task 8: Update CLI run command

**Files:**
- Modify: `cmd/bootconf/commands/run.go`
- Modify: `cmd/bootconf/commands/validate.go` (if needed)
- Modify: `cmd/bootconf/commands/status.go` (status dir path)
- Modify: `cmd/bootconf/commands/check.go` (if needed)

**Changes to `run.go`:**

- Check `cfg.Bootconf.Enabled` — if false, exit 0 with log message
- Construct modules with explicit directories:
  ```go
  ssh.New(cfg.SSH.Daemon, cfg.SSH.Keytype, cfg.SSH.GenerateHostKeys, cfg.SSH.Enabled, cfg.SSH.Directory, cfg.Services.Directory)
  wifi.New(cfg.Wifi.Enabled, cfg.Wifi.SSID, cfg.Wifi.PasswordHash, cfg.Wifi.Country, cfg.Wifi.Directory, cfg.Services.Directory)
  services.New(cfg.Services.Services, cfg.Services.Directory)
  sudo.New(cfg.Sudo.Enabled, cfg.Sudo.Directory, cfg.Users.Users, cfg.Sudo.ExtraSettings)
  users.New(cfg.Users.Users, 2000, cfg.Users.Directory)
  files.New(cfg.Files)
  ```
- Status dir: `filepath.Join(cfg.Bootconf.Directory, ".bootconf")`

---

### Task 9: Update integration tests

**Files:**
- Modify: `test/integration_test.go`

Update YAML test fixtures to use new config structure with explicit directories per section. Add sudo module assertions.

---

### Task 10: Verify and clean up

- Run `make clean && make && make test` — all tests pass
- Run `go fmt ./...`
- Run `./build/bin/bootconf validate --config bootconf.yaml`
- Update `planning/` docs
