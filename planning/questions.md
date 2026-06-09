# Bootconf Design Q&A

Answer inline below each question. Leave anything you're unsure about blank
and we'll circle back.

---

## 1. System Context

**1a.** What distro is this running on? Buildroot? Something custom? This
determines what tooling is available and how packages are managed.

> We are running on buildroot, but I want this also to work on other distros.
  The code is not that distro dependent: As long as we set a basedir that is
  writeable, this can run everywhere: See ../disco for another go project that
  we offer for multiple distros

**1b.** What's the target architecture? ARM (Raspberry Pi? iMX?), x86? This
matters for cross-compilation.

> arm64, we're on a mac running arm64 and have a buildbox present if needed.

**1c.** Is this always running on a single hardware platform, or does it need
to support multiple boards?

> We want to allow as many boards in the future: Don't pin on the hardware
> platform. At the same time: we're not building for x86_64 right now.

---

## 2. Filesystem Layout

**2a.** Is `/data` always writable at the point bootconf runs, or do we need to
handle cases where it might not be mounted yet? You said bootconf runs "after
all partitions are mounted" — is that guaranteed by the systemd unit ordering?

> yes in this case it is. but you should always check if you can write and bail
> out if not.

> do not add wait loops for volumes, we handle that in systemd configuration

**2b.** Is `/boot/firmware` always available at bootconf run time? (Since
that's where `bootconf.yaml` lives.)

> yes! But add a --config flag to make sure we can pass a config from anywhere.

**2c.** `/data/.bootconf` — is this a file (JSON/status) or a directory with
multiple files? What format do you want for the status data?

> the status can be json or yaml or whatever works. .bootconf is a directory we
> can use for any purpose to write files we need later.

**2d.** `/etc/sudoers.d/<name>.conf` — this is on the **readonly** filesystem.
How does this work? Is there a bind mount or overlay for `/etc/sudoers.d/`? Or
should we write sudoers configs to `/data/config/sudoers.d/` and have the unit
files/symlinks point there?

> Good one! Please add a task for me in builder to create a
> /etc/sudoers.d/include.conf that includes some files in /data/config/sudoers.d
> Additionally: create /data/config/sudoers.d with the correct permissions

**2e.** Similarly, `sysusers.d` — is `/etc/sysusers.d/` writable, or should we
write to `/data/config/sysusers.d/` and have a bind mount/symlink?

> Also good one: I think we should configure systemd-sysusers
> /data/config/sysusers.d/* or something: We can do that on the builder side:
> Please keep task for it

---

## 3. Bootconf.yaml & Config Semantics

**3a.** The `order` field — when `parallel: true`, does order only matter for
the status output, or should there still be sequential ordering within parallel
groups?

> I removed the order key: Lets not use it and always run in parallel

**3b.** What happens if `order` is set but doesn't include all sections? The
doc says "if set and fields are left out: do not configure them." So if someone
lists only `[system, ssh]`, wifi/users/services/files are simply skipped?

> Lets always run all sections except the ones in bootconf.exclude: []
> I've updated the config file accordingly.

**3c.** What happens if `order` is **not** set at all? The doc says "use a
default order" — is that the order shown in the example (system, ssh, files,
services, wifi, users)?

> We just removed order and replaced it with exclude: run all in parallel but do
> not run the excluded sections

**3e.** For `wifi.password` vs `wifi.password_hash` — is `password` always
plaintext that gets run through `wpa_passphrase`? What if both are set — does
`password_hash` take precedence?

> I just removed the password: We only use password_hash: secure by design and
> no plain passwords in our config directory.
> And saves us from installing another dependency

---

## 4. Users & SSH

**4a.** SSH is dropbear (since `dropbearkey` is mentioned). Confirm? Or could
it be OpenSSH as well?

> I added the daemon section, which is now set dropbear. Lets support both.
> if the daemon is set to openssh, use ssh tools, if set to dropbear, use
> dropbear tools

**4b.** For users — what shell should they get? Is there a default, or should
it be configurable in the YAML?

> always /bin/bash, no exceptions :)

**4c.** The authorized_keys file uses `# ---BEGIN CONFIG--- / # ---END CONFIG---`
markers. Is this a convention you already use, or something new? The idea is:
bootconf only manages lines between the markers, and leaves any manually added
keys alone?

> You know what, skip the markers: we claim the whole file: if you want to
> manage authorized_keys: you have to do it through the config file.
> manual changes will be wiped :)

**4d.** What UID/GID range should users get? Should it be configurable, or do
we let systemd-sysusers handle allocation?

> lets start with 2000 and up: our apps start at 6000.

**4e.** If a user is removed from the YAML, should bootconf disable/remove
them, or just leave them as-is?

> I'm not sure how systemd-sysusers does this, but it cannot remove users.
> I think removal is good, but I just added the `enabled: true` flag to the
> user.
> If disabled: remove, if enabled: add.
> systemd-sysusers recommends to remove using userdel: I will install those in
> the builder.

> If the userfile is kept in the overlay: The users are gone with every reboot
> and recreated at boot time. if this is the case, we should make sure we have
> static uid/gid
---

## 5. Services

**5a.** You mention that unit files already have (or will have)
`ConditionPathExists=/data/config/services/<name>`. Are these unit files
maintained in a separate repo (the OS build), or do we also need to
generate/modify them here?

> So far none, I will add those later in another task in another repo.

**5b.** For `copy_default_config: true` — the source is always
`/etc/<name>/<name>.conf`? Or should the source path be configurable?

> Lets set a source: I just changed the config.
> default_config:
>   copy: true
>    source: /etc/disco/disco.conf
>    destination: disco/disco.conf # relative to basedir

---

## 6. Logging

**6a.** For journald logging — do you want native journal integration (e.g.,
`sd_journal_send` with structured fields), or is plain stdout/stderr
sufficient? systemd already captures stdout/stderr from services into the
journal.

> plain stdout/stderr

**6b.** The doc says "Make sure that each section is shown in the logging by
adding it to the log format" — do you want something like
`[wifi] configuring SSID 'something'` as the format?

> something like: "bootconf: <time> <severity> <section::wifi> <message>"  exactly!

---

## 7. Build & Project

**7a.** Go module name? Something like `github.com/offline-lab/bootconf`?

> yes!

**7b.** Go version — do you have a preference, or should I target the latest
stable?

> Latest stable LTS, but make it universal enough that switching is easy

**7c.** Do you want a Makefile? Just `go build`? How is this built and packaged
into the OS image?

> both! See disco for a better understanding. Also add the agent documentation
> and instructions in the readme on how to build

---

## 8. Validation & Schema

**8a.** For the config schema — do you want a standalone JSON Schema file that
can be used by editors/validators, or is Go struct validation (with tags)
sufficient? Or both?

> ideally we make a real schema and do validation so that we can also create
> nice error messages. The tool runs at boot, so good error messages are
> important.

> I cannot decide, I like json schema validation with a file, but feel free to
> do that with go objects only: the schema validation will not be reused by
> other tools or in a library so no need to overdo things.

**8b.** Should `bootconf validate` be able to run offline (no system
dependencies needed), or is it okay if it needs the same environment?

> yes! it should never download or have a network connection to validate: build
> in plz

---

## 9. Error Handling & Edge Cases

**9a.** If the config file (`/boot/firmware/bootconf.yaml`) doesn't exist —
what should happen? Silent exit 0? Error? Generate a default config?

> silent exit 0: we use a condition in the unit file to check for /boot/firmware/bootconf.yaml

**9b.** What if the config file exists but is empty or has only the
`bootconf:` header with no sections?

> assume `enabled: false` for all sections unless enabled in config
> if verbose logging is enabled, --verbose, we should first log config not
> found, then log for each section that
> it's not being executed because it's not enabled.

**9c.** The `--dry-run` flag — should it actually check file permissions, disk
space, etc., or just print what it *would* do?

> do check, no writes: we want to run that on the cli to test if it works :)
> a real dry-run: Run, test everything, but don't execute or create files.

---

## 10. Testing

**10a.** For testing — are you thinking mostly table-driven unit tests with
fake filesystems (afero, etc.), or do you also want to spin up QEMU with the
actual image?

> I think qemu is a bit overkill, but do what you need to do: Testing this kind
> of stuff is hard so if we have to run some things in a docker to test that is
> acceptable.
