# services

Manages service sentinel files and optionally provisions default config files.

## What it does

For each configured service entry:

- If `sentinel: true` and `enabled: true`: creates an empty file at `<directory>/<name>`.
- If `sentinel: true` and `enabled: false`: removes `<directory>/<name>` if it exists.
- If `default_config.copy: true` and `enabled: true`: copies the default config from source to destination on first presence.

Bootconf does not start or stop daemons. The sentinel pattern lets your systemd units decide whether to run a service:

```ini
[Unit]
ConditionPathExists=/etc/bootconf/services/myservice
```

## Config

```yaml
services:
  enabled: true
  directory: /etc/bootconf/services
  services:
    - name: myservice
      enabled: true
      sentinel: true
      default_config:
        copy: true
        source: /etc/myservice/myservice.conf
        destination: /data/config/myservice/myservice.conf
```

| Field | Type | Description |
|-------|------|-------------|
| `enabled` | bool | Enable the module |
| `directory` | string | Where sentinel files are written |
| `services[].name` | string | Service name; used as the sentinel filename |
| `services[].enabled` | bool | Create (`true`) or remove (`false`) the sentinel |
| `services[].sentinel` | bool | Whether to manage a sentinel file for this service |
| `services[].default_config.copy` | bool | Copy the default config if set to `true` |
| `services[].default_config.source` | string | Source path for the default config |
| `services[].default_config.destination` | string | Destination path for the default config |

## Default config copy

When `default_config.copy: true`, bootconf copies the source file to the destination path:

- If the destination already exists, the content is written alongside it as `<destination>.new` — existing config is **never overwritten**.
- The destination directory is created if it does not exist (mode `0750`).
- The copied file is owned by `root:root` with mode `0640`.

This is designed for first-boot provisioning of a writable config from a read-only default. After the first copy, the file is yours — bootconf will not touch it again.

## Dry-run

```
INFO [services] would write sentinel /etc/bootconf/services/myservice (dry-run)
INFO [services] would copy /etc/myservice/myservice.conf → /data/config/myservice/myservice.conf (dry-run)
```
