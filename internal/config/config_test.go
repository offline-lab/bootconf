package config

import (
	"os"
	"reflect"
	"testing"
)

// testConfigYAML is the complete YAML fixture for TestLoad. It covers every
// config section so that a new field added without a corresponding assertion
// will be obvious.
const testConfigYAML = `
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

// mustLoadYAML writes content to a temp file and calls Load, failing the test
// on any error. It is the standard fixture loader for config package tests.
func mustLoadYAML(t *testing.T, content string) *Config {
	t.Helper()
	f, err := os.CreateTemp("", "bootconf-test-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(f.Name())
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	return cfg
}

// TestLoad verifies that every section of a fully-populated YAML config is
// parsed and mapped correctly. Each section is isolated in its own sub-test
// so failures are easy to locate without wading through a wall of assertions.
func TestLoad(t *testing.T) {
	cfg := mustLoadYAML(t, testConfigYAML)

	t.Run("bootconf", func(t *testing.T) {
		// Order is populated by registry.ApplyDefaults, not config.SetDefaults,
		// so it remains nil after Load alone.
		want := BootconfConfig{
			Enabled:   true,
			Directory: "/test/bootconf",
		}
		if !reflect.DeepEqual(cfg.Bootconf, want) {
			t.Errorf("got %+v, want %+v", cfg.Bootconf, want)
		}
	})

	t.Run("system", func(t *testing.T) {
		want := SystemConfig{Enabled: true, Timezone: "UTC", Hostname: "testhost"}
		if cfg.System != want {
			t.Errorf("got %+v, want %+v", cfg.System, want)
		}
	})

	t.Run("ssh", func(t *testing.T) {
		want := SSHConfig{
			Enabled:          true,
			Directory:        "/test/ssh",
			Keytype:          "rsa",
			GenerateHostKeys: false,
			Daemon:           "openssh",
		}
		if cfg.SSH != want {
			t.Errorf("got %+v, want %+v", cfg.SSH, want)
		}
	})

	t.Run("wifi", func(t *testing.T) {
		want := WifiConfig{
			Enabled:      true,
			Directory:    "/test/wifi",
			SSID:         "testnet",
			PasswordHash: "hash123",
			Country:      "US",
		}
		if cfg.Wifi != want {
			t.Errorf("got %+v, want %+v", cfg.Wifi, want)
		}
	})

	t.Run("services", func(t *testing.T) {
		if !cfg.Services.Enabled || cfg.Services.Directory != "/test/services" {
			t.Errorf("header: enabled=%v dir=%q", cfg.Services.Enabled, cfg.Services.Directory)
		}
		if len(cfg.Services.Services) != 1 {
			t.Fatalf("count: got %d, want 1", len(cfg.Services.Services))
		}
		want := ServiceEntry{
			Name:     "disco",
			Enabled:  true,
			Sentinel: true,
			DefaultConfig: DefaultConfig{
				Copy:        true,
				Source:      "/etc/disco/disco.conf",
				Destination: "/data/config/disco/disco.conf",
			},
		}
		if cfg.Services.Services[0] != want {
			t.Errorf("Services[0]: got %+v, want %+v", cfg.Services.Services[0], want)
		}
	})

	t.Run("users", func(t *testing.T) {
		if !cfg.Users.Enabled || cfg.Users.Directory != "/test/users" {
			t.Errorf("header: enabled=%v dir=%q", cfg.Users.Enabled, cfg.Users.Directory)
		}
		if len(cfg.Users.Users) != 1 {
			t.Fatalf("count: got %d, want 1", len(cfg.Users.Users))
		}
		u := cfg.Users.Users[0]
		if u.Name != "admin" || !u.Enabled || !u.Sudo || u.Home != "/data/home/admin" {
			t.Errorf("core fields: name=%q enabled=%v sudo=%v home=%q", u.Name, u.Enabled, u.Sudo, u.Home)
		}
		if len(u.AuthorizedKeys) != 1 || u.AuthorizedKeys[0] != "ssh-ed25519 AAAA test@host" {
			t.Errorf("authorized_keys: got %v, want [ssh-ed25519 AAAA test@host]", u.AuthorizedKeys)
		}
	})

	t.Run("files", func(t *testing.T) {
		if len(cfg.Files.Files) != 1 {
			t.Fatalf("count: got %d, want 1", len(cfg.Files.Files))
		}
		want := FileEntry{
			Source:      "/boot/firmware/config/test.conf",
			Destination: "test/test.conf",
			Chmod:       "644",
		}
		if cfg.Files.Files[0] != want {
			t.Errorf("Files[0]: got %+v, want %+v", cfg.Files.Files[0], want)
		}
	})
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
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	if err := tmpFile.Close(); err != nil {
		t.Fatal(err)
	}

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

	cfg.Users = UsersConfig{Users: []UserEntry{{Name: "alice", Enabled: true}}}
	cfg.SetDefaults()
	if cfg.Users.Users[0].Home != "/home/alice" {
		t.Errorf("UserEntry.Home default = %q, want %q", cfg.Users.Users[0].Home, "/home/alice")
	}

	explicitHome := &Config{
		Users: UsersConfig{Users: []UserEntry{{Name: "bob", Home: "/custom/home/bob"}}},
	}
	explicitHome.SetDefaults()
	if explicitHome.Users.Users[0].Home != "/custom/home/bob" {
		t.Errorf("UserEntry.Home = %q, want %q (should not overwrite)", explicitHome.Users.Users[0].Home, "/custom/home/bob")
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
