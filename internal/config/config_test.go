package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	content := `
bootconf:
  enabled: true
  directory: /test/bootconf
system:
  enabled: true
  timezone: UTC
  hostname: testhost
ssh:
  enabled: true
  directory: /test/ssh
  keytype: rsa
  generate_host_keys: false
  daemon: openssh
wifi:
  enabled: true
  directory: /test/wifi
  ssid: testnet
  password_hash: hash123
  country: US
services:
  enabled: true
  directory: /test/services
  services:
    - name: disco
      enabled: true
      sentinel: true
      default_config:
        copy: true
        source: /etc/disco/disco.conf
        destination: /data/config/disco/disco.conf
users:
  enabled: true
  directory: /test/users
  users:
    - name: admin
      enabled: true
      sudo: true
      home: /data/home/admin
      authorized_keys:
        - "ssh-ed25519 AAAA test@host"
files:
  enabled: true
  files:
    - source: /boot/firmware/config/test.conf
      destination: test/test.conf
      chmod: "644"
`
	tmpFile, err := os.CreateTemp("", "bootconf-test-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if !cfg.Bootconf.Enabled {
		t.Error("Bootconf.Enabled = false, want true")
	}
	if cfg.Bootconf.Directory != "/test/bootconf" {
		t.Errorf("Bootconf.Directory = %q, want %q", cfg.Bootconf.Directory, "/test/bootconf")
	}
	if cfg.System.Timezone != "UTC" {
		t.Errorf("Timezone = %q, want %q", cfg.System.Timezone, "UTC")
	}
	if cfg.System.Hostname != "testhost" {
		t.Errorf("Hostname = %q, want %q", cfg.System.Hostname, "testhost")
	}
	if !cfg.SSH.Enabled {
		t.Error("SSH.Enabled = false, want true")
	}
	if cfg.SSH.Directory != "/test/ssh" {
		t.Errorf("SSH.Directory = %q, want %q", cfg.SSH.Directory, "/test/ssh")
	}
	if cfg.SSH.Keytype != "rsa" {
		t.Errorf("SSH.Keytype = %q, want %q", cfg.SSH.Keytype, "rsa")
	}
	if cfg.SSH.GenerateHostKeys {
		t.Error("SSH.GenerateHostKeys = true, want false")
	}
	if cfg.SSH.Daemon != "openssh" {
		t.Errorf("SSH.Daemon = %q, want %q", cfg.SSH.Daemon, "openssh")
	}
	if !cfg.Wifi.Enabled {
		t.Error("Wifi.Enabled = false, want true")
	}
	if cfg.Wifi.Directory != "/test/wifi" {
		t.Errorf("Wifi.Directory = %q, want %q", cfg.Wifi.Directory, "/test/wifi")
	}
	if cfg.Wifi.SSID != "testnet" {
		t.Errorf("Wifi.SSID = %q, want %q", cfg.Wifi.SSID, "testnet")
	}
	if cfg.Wifi.PasswordHash != "hash123" {
		t.Errorf("Wifi.PasswordHash = %q, want %q", cfg.Wifi.PasswordHash, "hash123")
	}
	if cfg.Wifi.Country != "US" {
		t.Errorf("Wifi.Country = %q, want %q", cfg.Wifi.Country, "US")
	}
	if !cfg.Services.Enabled {
		t.Error("Services.Enabled = false, want true")
	}
	if cfg.Services.Directory != "/test/services" {
		t.Errorf("Services.Directory = %q, want %q", cfg.Services.Directory, "/test/services")
	}
	if len(cfg.Services.Services) != 1 {
		t.Fatalf("Services length = %d, want 1", len(cfg.Services.Services))
	}
	svc := cfg.Services.Services[0]
	if svc.Name != "disco" {
		t.Errorf("Service Name = %q, want %q", svc.Name, "disco")
	}
	if !svc.Enabled {
		t.Error("Service Enabled = false, want true")
	}
	if !svc.Sentinel {
		t.Error("Service Sentinel = false, want true")
	}
	if !svc.DefaultConfig.Copy {
		t.Error("Service DefaultConfig.Copy = false, want true")
	}
	if svc.DefaultConfig.Source != "/etc/disco/disco.conf" {
		t.Errorf("Service DefaultConfig.Source = %q, want %q", svc.DefaultConfig.Source, "/etc/disco/disco.conf")
	}
	if svc.DefaultConfig.Destination != "/data/config/disco/disco.conf" {
		t.Errorf("Service DefaultConfig.Destination = %q, want %q", svc.DefaultConfig.Destination, "/data/config/disco/disco.conf")
	}
	if cfg.Users.Directory != "/test/users" {
		t.Errorf("Users.Directory = %q, want %q", cfg.Users.Directory, "/test/users")
	}
	if len(cfg.Users.Users) != 1 {
		t.Fatalf("Users length = %d, want 1", len(cfg.Users.Users))
	}
	usr := cfg.Users.Users[0]
	if usr.Name != "admin" {
		t.Errorf("User Name = %q, want %q", usr.Name, "admin")
	}
	if !usr.Enabled {
		t.Error("User Enabled = false, want true")
	}
	if !usr.Sudo {
		t.Error("User Sudo = false, want true")
	}
	if usr.Home != "/data/home/admin" {
		t.Errorf("User Home = %q, want %q", usr.Home, "/data/home/admin")
	}
	if len(usr.AuthorizedKeys) != 1 || usr.AuthorizedKeys[0] != "ssh-ed25519 AAAA test@host" {
		t.Errorf("User AuthorizedKeys = %v, want [ssh-ed25519 AAAA test@host]", usr.AuthorizedKeys)
	}
	if len(cfg.Files.Files) != 1 {
		t.Fatalf("Files length = %d, want 1", len(cfg.Files.Files))
	}
	fileEntry := cfg.Files.Files[0]
	if fileEntry.Source != "/boot/firmware/config/test.conf" {
		t.Errorf("File Source = %q, want %q", fileEntry.Source, "/boot/firmware/config/test.conf")
	}
	if fileEntry.Destination != "test/test.conf" {
		t.Errorf("File Destination = %q, want %q", fileEntry.Destination, "test/test.conf")
	}
	if fileEntry.Chmod != "644" {
		t.Errorf("File Chmod = %q, want %q", fileEntry.Chmod, "644")
	}
}

func TestLoadMissing(t *testing.T) {
	_, err := Load("/nonexistent/path/bootconf.yaml")
	if err == nil {
		t.Fatal("Load() should fail for nonexistent file")
	}
}

func TestLoadEmpty(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "bootconf-test-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Load() of empty file failed: %v", err)
	}

	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}
}

func TestSetDefaults(t *testing.T) {
	cfg := &Config{}
	cfg.SetDefaults()

	if cfg.Bootconf.Directory != "/data/bootconf" {
		t.Errorf("Bootconf.Directory default = %q, want %q", cfg.Bootconf.Directory, "/data/bootconf")
	}
	if cfg.SSH.Keytype != "ed25519" {
		t.Errorf("SSH.Keytype default = %q, want %q", cfg.SSH.Keytype, "ed25519")
	}
	if cfg.SSH.Daemon != "dropbear" {
		t.Errorf("SSH.Daemon default = %q, want %q", cfg.SSH.Daemon, "dropbear")
	}

	cfg.Files = FilesConfig{Files: []FileEntry{{Source: "/a", Destination: "/b"}}}
	cfg.SetDefaults()
	if cfg.Files.Files[0].Chmod != "640" {
		t.Errorf("FileEntry.Chmod default = %q, want %q", cfg.Files.Files[0].Chmod, "640")
	}

	explicitCfg := &Config{
		Bootconf: BootconfConfig{Directory: "/custom"},
		SSH:      SSHConfig{Keytype: "rsa", Daemon: "openssh"},
		Files:    FilesConfig{Files: []FileEntry{{Source: "/a", Destination: "/b", Chmod: "755"}}},
	}
	explicitCfg.SetDefaults()
	if explicitCfg.Bootconf.Directory != "/custom" {
		t.Errorf("Bootconf.Directory = %q, want %q (should not overwrite)", explicitCfg.Bootconf.Directory, "/custom")
	}
	if explicitCfg.SSH.Keytype != "rsa" {
		t.Errorf("SSH.Keytype = %q, want %q (should not overwrite)", explicitCfg.SSH.Keytype, "rsa")
	}
	if explicitCfg.SSH.Daemon != "openssh" {
		t.Errorf("SSH.Daemon = %q, want %q (should not overwrite)", explicitCfg.SSH.Daemon, "openssh")
	}
	if explicitCfg.Files.Files[0].Chmod != "755" {
		t.Errorf("FileEntry.Chmod = %q, want %q (should not overwrite)", explicitCfg.Files.Files[0].Chmod, "755")
	}
}
