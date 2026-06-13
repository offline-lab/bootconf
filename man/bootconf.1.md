% BOOTCONF(1) bootconf | Boot-time Configuration
% Flip Hess
% June 2026

# NAME

bootconf - declarative boot configuration for Linux

# SYNOPSIS

**bootconf** *command* [options]

# DESCRIPTION

Bootconf reads a YAML configuration file and applies system settings
during early boot, before other services start. It handles hostname,
timezone, SSH host keys, WiFi credentials, user accounts, service
sentinel files, and arbitrary file provisioning.

It runs on every boot, so configuration changes take effect on the
next reboot without reinstalling or reimaging the system. It writes
status to a JSON file so downstream tooling can verify success.

# COMMANDS

**run**
: Apply all configuration sections from the config file.

**check**
: Verify that the running system matches the configuration. Checks
  service status, user existence, and process state.

**validate**
: Parse and validate the config file without making changes. Useful
  in CI pipelines.

**status**
: Show results of the most recent bootconf run.

**version**
: Print version, commit, and build time.

# OPTIONS

**-c**, **--config** *path*
: Path to the configuration file.
  Default: `/boot/firmware/bootconf.yaml`

**-v**, **--verbose**
: Enable verbose output.

**--dry-run**
: (run command) Show what would be done without making changes.

**--section** *name*
: (run/check command) Only operate on a specific section.

**--failed**
: (status command) Only show failed sections.

**--full**
: (status command) Show all details including duration and messages.

# CONFIGURATION

The configuration file uses YAML. See `/etc/bootconf/bootconf.yaml.example`
for a complete annotated reference.

Sections: **bootconf**, **system**, **ssh**, **wifi**, **services**,
**users**, **files**.

Each section has an `enabled` flag. Disabled sections are skipped
entirely during validation and execution.

# EXIT STATUS

**0**
: Success.

**1**
: Failure (invalid config, runtime error, or health check failure).

# FILES

`/boot/firmware/bootconf.yaml`
: Default configuration file path.

`/data/config/bootconf/.bootconf/status.json`
: JSON status file from the last run.

# EXAMPLES

Validate config without changes:

    bootconf validate

Apply configuration:

    bootconf run

Apply with verbose output:

    bootconf run -v

Check system health:

    bootconf check

Show last run status:

    bootconf status

Show only failed sections with details:

    bootconf status --failed --full

Dry run to preview changes:

    bootconf run --dry-run

Run only the SSH section:

    bootconf run --section ssh

# SEE ALSO

hostnamectl(1), timedatectl(1), wpa_passphrase(8), useradd(8)
