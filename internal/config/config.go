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
	Bootconf  BootconfConfig  `yaml:"bootconf"`
	System    SystemConfig    `yaml:"system"`
	SSH       SSHConfig       `yaml:"ssh"`
	Wifi      WifiConfig      `yaml:"wifi"`
	Services  ServicesConfig  `yaml:"services"`
	Users     UsersConfig     `yaml:"users"`
	Files     FilesConfig     `yaml:"files"`
	Templates TemplatesConfig `yaml:"templates"`
	Shell     ShellConfig     `yaml:"shell"`
	UnitRun   UnitRunConfig   `yaml:"unitrun"`
}

// BootconfConfig controls the overall bootconf tool: where it stores status
// and whether it runs at all.
type BootconfConfig struct {
	Enabled   bool     `yaml:"enabled"`
	Directory string   `yaml:"directory"`
	Order     []string `yaml:"order"`
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
	Enabled     bool        `yaml:"enabled"`
	Directory   string      `yaml:"directory"`
	TmpfilesDir string      `yaml:"tmpfiles_dir"`
	Users       []UserEntry `yaml:"users"`
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

// FileEntry maps a source file or inline content to a destination path.
// Exactly one of Source or Content must be set.
type FileEntry struct {
	Source      string `yaml:"source"`
	Content     string `yaml:"content"`
	Destination string `yaml:"destination"`
	Chmod       string `yaml:"chmod"`
}

// FilesConfig lists arbitrary files to copy or write into the target filesystem.
type FilesConfig struct {
	Enabled bool        `yaml:"enabled"`
	Files   []FileEntry `yaml:"files"`
}

// TemplateEntry renders a Go text/template file with provided variables.
// Template syntax: {{ .variableName }}. Missing keys cause an error at render time.
type TemplateEntry struct {
	Source      string            `yaml:"source"`
	Destination string            `yaml:"destination"`
	Variables   map[string]string `yaml:"variables"`
	Chmod       string            `yaml:"chmod"`
}

// TemplatesConfig lists template files to render and install.
type TemplatesConfig struct {
	Enabled   bool            `yaml:"enabled"`
	Templates []TemplateEntry `yaml:"templates"`
}

// ShellCommand is a single shell command to execute at boot.
// Output (stdout, stderr, exit code) is written to <directory>/<name>.log.
type ShellCommand struct {
	Name      string `yaml:"name"`
	AllowFail bool   `yaml:"allow_fail"`
	FirstBoot bool   `yaml:"firstboot"`
	Command   string `yaml:"command"`
}

// ShellConfig lists shell commands to run during boot.
type ShellConfig struct {
	Enabled   bool           `yaml:"enabled"`
	Directory string         `yaml:"directory"`
	Path      string         `yaml:"path"`
	Commands  []ShellCommand `yaml:"commands"`
}

// UnitEntry describes a shell script to run via a generated systemd unit.
// When FirstBoot is true, ConditionFirstBoot=yes is added to the generated
// unit so systemd skips it after the first boot — no custom sentinel needed.
type UnitEntry struct {
	Name         string   `yaml:"name"`
	Enabled      bool     `yaml:"enabled"`
	FirstBoot    bool     `yaml:"firstboot"`
	Dependencies []string `yaml:"dependencies"`
	Command      string   `yaml:"command"`
}

// UnitRunConfig defines scripts to run via generated systemd units.
// Each enabled unit writes a script to Directory and a .service file to
// /etc/systemd/system/, then calls systemctl enable + daemon-reload.
type UnitRunConfig struct {
	Enabled   bool        `yaml:"enabled"`
	Directory string      `yaml:"directory"`
	Path      string      `yaml:"path"`
	Units     []UnitEntry `yaml:"units"`
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
// (e.g. ed25519 key type, dropbear daemon, 640 file permissions, /home/<name> for users).
// Bootconf.Order is NOT defaulted here; call registry.ApplyDefaults after Load.
func (cfg *Config) SetDefaults() {
	if cfg.Bootconf.Directory == "" {
		cfg.Bootconf.Directory = "/data/bootconf"
	}
	if cfg.SSH.Keytype == "" {
		cfg.SSH.Keytype = "ed25519"
	}
	if cfg.SSH.Daemon == "" {
		cfg.SSH.Daemon = "dropbear"
	}
	if cfg.Users.TmpfilesDir == "" {
		cfg.Users.TmpfilesDir = "/data/config/tmpfiles"
	}
	for index := range cfg.Users.Users {
		if cfg.Users.Users[index].Home == "" && cfg.Users.Users[index].Name != "" {
			cfg.Users.Users[index].Home = "/home/" + cfg.Users.Users[index].Name
		}
	}
	for index := range cfg.Files.Files {
		if cfg.Files.Files[index].Chmod == "" {
			cfg.Files.Files[index].Chmod = "640"
		}
	}
	for index := range cfg.Templates.Templates {
		if cfg.Templates.Templates[index].Chmod == "" {
			cfg.Templates.Templates[index].Chmod = "640"
		}
	}
}
