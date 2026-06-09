# bootconf

Bootconf is a small go utility that configures our readonly operating system at
boot time.

The bootconf and it's config, are triggered right after all the partitions are
mounted but before any other service is started. All other services depend on
this unit.

Bootconf is a single binary configuration utility written in go
that does boot configuration: It is not meant to run only at first boot,
but runs every boot and can change configurations on the fly prior to booting
the rest of the operating system.

It is not a service configuration mechanism either: We use it in conjunction
with systemd on a readonly filesystem.

After bootconf has run, all services can be started because their configuration
is in place.


## Context / Usage

In our readonly operating system, only /boot is fat32 and as such readable on
microsoft and mac OSX.

We want to be able to place a bootconf.yaml configuration file in /boot/firmware
and run bootconf /boot/firmware/bootconf.yaml from systemd as one of the first
units.

Because our filesystem is readonly, we have a data directory for writeable data:
config files that users want to change. Everything else is readonly.

bootconf then:
- Creates config files in the writeable /data/config directory for services if
  needed.
- Create a /data/config/services/<service> file
  All unit files for these services then depend on bootconf.service and the
  condition that /data/config/services/<service> exists.
  This way we can leave all our unit files enabled, but they only start when a
  file is present in the writeable filesystem.
- Generates SSH hostkeys
- Generates a wifi config
- Sets the hostname
- Sets the timezone
- Create sysuser.d configuration files that take care of the user configuration
- Creates the homedir for each user
- Sets permissions
- Copies authorized_keys in place for each user
- Creates sudo configuration

Each nested section in the bootconf.yaml that is created as an example shuold
have it's own module. We want the tool to be modular and easy to extend with
additional services.

Each section should also be run individually by adding `--section <>`

bootconf should have several cli subcommands:

- status    -> Should show the last run's exit code and status.
- run       -> Runs the actual confguration run.
- validate  -> Validates the config file using a config schema.
- check     -> Check services and show a fully overview of running/not running

Do not keep a log file: instead log to journald

Before writing any file: Assume the filesystem can be readonly, so test then
write or catch error.

Each action gets their own status, but we do not stop running: If a section
fails, try executing the other items if a list
if the section fails and it's not a loop over a list of items, then just move to
  the next section.



## Cli

### Bootconf help

bootconf help -> shows help

Show the full usage, link to a webpage


### bootconf version

Show the version of this build


### bootconf check

bootconf check: show the status of all the defined configuration: not the status,
the result:

  - check if ssh is running
  - check if wifi is up and running
  - check if all defined services are up
  - check if all users can be looked up

### bootconf validate

bootconf validate: Validates the config file using a config schema

We should keep a schema for validation purposes for our config file.

### Bootconf run

Does the actual execution of the commands as defined in the config file

bootconf run: runs the actual confguration run
bootconf run --dry-run: should run a dryrun without really creating files.
bootconf run --section <section> -> runs the section only

### Bootconf status

bootconf status should show the overall status from a status file that is written in
/data/.bootconf.

For each section success or fail should be kept and if failed, the section that
failed.

bootconf status -> shows the recorded status of the last run
bootconf status --section <section> -> shows the status of the last run of that section
bootconf status --failed -> only shows what failed
bootconf status --full   -> shows the full trace of each section and the success or not


## Services

For services, we create /data/config/services/<name>

if the file exists, the unit file has the condition to only run if the file is
  present.


## Wifi

for wifi we need to create a config file with the username+password and the
  country

We store this in /data/config/wifi/wpa_supplicant.conf and leave it as such.
Systemd runs the init script that starts wifi if this file exists and
/data/config/services/wifi exists.

## Users

We let systemd take care of creating users.
We only create the config file that systemd-sysusers need to do this.

We do create the homedir and add authorized_keys


## Logging

Logging is important: We should make sure you can easily see which module runs
what at what time, even things are running at the same time.

Make sure that each section is shown in the logging by adding it to the log
format (the function or section)


## Testing

We need a way to test this: Ideally we don't really configure wpa_supplicant or
dropbear, but we can install them and test syntaxes

We should have enough coverage and e2e tests to safely make updates

## Documentation

We shuold have a clear and concise documentation that folloows diataxis
framework to offer a wide range of useful instructions and information
