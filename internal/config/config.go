package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Bootconf BootconfConfig `yaml:"bootconf"`
	System   SystemConfig   `yaml:"system"`
	SSH      SSHConfig      `yaml:"ssh"`
	Wifi     WifiConfig     `yaml:"wifi"`
	Services ServicesConfig `yaml:"services"`
	Users    UsersConfig    `yaml:"users"`
	Files    FilesConfig    `yaml:"files"`
}

type BootconfConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Directory string `yaml:"directory"`
}

type SystemConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Timezone string `yaml:"timezone"`
	Hostname string `yaml:"hostname"`
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

type DefaultConfig struct {
	Copy        bool   `yaml:"copy"`
	Source      string `yaml:"source"`
	Destination string `yaml:"destination"`
}

type UsersConfig struct {
	Enabled   bool        `yaml:"enabled"`
	Directory string      `yaml:"directory"`
	Users     []UserEntry `yaml:"users"`
}

type UserEntry struct {
	Name           string   `yaml:"name"`
	Enabled        bool     `yaml:"enabled"`
	Sudo           bool     `yaml:"sudo"`
	Home           string   `yaml:"home"`
	AuthorizedKeys []string `yaml:"authorized_keys"`
}

type FileEntry struct {
	Source      string `yaml:"source"`
	Destination string `yaml:"destination"`
	Chmod       string `yaml:"chmod"`
}

type FilesConfig struct {
	Enabled bool        `yaml:"enabled"`
	Files   []FileEntry `yaml:"files"`
}

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
