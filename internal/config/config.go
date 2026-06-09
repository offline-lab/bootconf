// Package config handles loading, validating, and defaulting the bootconf
// YAML configuration file. Each section maps to a module that applies
// configuration at boot time.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration structure. Each field maps to a
// bootconf module that runs when enabled.
type Config struct {
	Bootconf BootconfConfig `yaml:"bootconf"`
	System   SystemConfig   `yaml:"system"`
	SSH      SSHConfig      `yaml:"ssh"`
	Wifi     WifiConfig     `yaml:"wifi"`
	Services ServicesConfig `yaml:"services"`
	Users    UsersConfig    `yaml:"users"`
	Files    FilesConfig    `yaml:"files"`
}

// BootconfConfig controls the overall bootconf tool: where it stores status
// and whether it runs at all.
type BootconfConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Directory string `yaml:"directory"`
}

// SystemConfig configures hostname and timezone on the target device.
type SystemConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Timezone string `yaml:"timezone"`
	Hostname string `yaml:"hostname"`
}

// SSHConfig controls SSH host key generation and daemon selection.
type SSHConfig struct {
	Enabled          bool   `yaml:"enabled"`
	Directory        string `yaml:"directory"`
	Keytype          string `yaml:"keytype"`
	GenerateHostKeys bool   `yaml:"generate_host_keys"`
	Daemon           string `yaml:"daemon"`
}

// WifiConfig holds wireless network credentials and wpa_supplicant settings.
type WifiConfig struct {
	Enabled      bool   `yaml:"enabled"`
	Directory    string `yaml:"directory"`
	SSID         string `yaml:"ssid"`
	PasswordHash string `yaml:"password_hash"`
	Country      string `yaml:"country"`
}

// ServicesConfig defines which system services should be enabled and
// optionally provisioned with default config files.
type ServicesConfig struct {
	Enabled   bool           `yaml:"enabled"`
	Directory string         `yaml:"directory"`
	Services  []ServiceEntry `yaml:"services"`
}

// ServiceEntry describes a single service: its name, whether it should run,
// and an optional default config to copy into place.
type ServiceEntry struct {
	Name          string        `yaml:"name"`
	Enabled       bool          `yaml:"enabled"`
	Sentinel      bool          `yaml:"sentinel"`
	DefaultConfig DefaultConfig `yaml:"default_config"`
}

// DefaultConfig specifies a source-to-destination file copy for a service.
type DefaultConfig struct {
	Copy        bool   `yaml:"copy"`
	Source      string `yaml:"source"`
	Destination string `yaml:"destination"`
}

// UsersConfig defines user accounts to create on the target device.
type UsersConfig struct {
	Enabled   bool        `yaml:"enabled"`
	Directory string      `yaml:"directory"`
	Users     []UserEntry `yaml:"users"`
}

// UserEntry describes a single user account: name, home directory, sudo
// membership, and SSH authorized keys.
type UserEntry struct {
	Name           string   `yaml:"name"`
	Enabled        bool     `yaml:"enabled"`
	Sudo           bool     `yaml:"sudo"`
	Home           string   `yaml:"home"`
	AuthorizedKeys []string `yaml:"authorized_keys"`
}

// FileEntry maps a source file to a destination path with specific permissions.
type FileEntry struct {
	Source      string `yaml:"source"`
	Destination string `yaml:"destination"`
	Chmod       string `yaml:"chmod"`
}

// FilesConfig lists arbitrary files to copy into the target filesystem.
type FilesConfig struct {
	Enabled bool        `yaml:"enabled"`
	Files   []FileEntry `yaml:"files"`
}

// Load reads and parses a bootconf YAML file, then applies defaults.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	cfg.SetDefaults()

	return cfg, nil
}

// SetDefaults fills in zero-valued fields with sensible defaults
// (e.g. ed25519 key type, dropbear daemon, 640 file permissions).
func (c *Config) SetDefaults() {
	if c.Bootconf.Directory == "" {
		c.Bootconf.Directory = "/data/bootconf"
	}
	if c.SSH.Keytype == "" {
		c.SSH.Keytype = "ed25519"
	}
	if c.SSH.Daemon == "" {
		c.SSH.Daemon = "dropbear"
	}
	for index := range c.Files.Files {
		if c.Files.Files[index].Chmod == "" {
			c.Files.Files[index].Chmod = "640"
		}
	}
}
