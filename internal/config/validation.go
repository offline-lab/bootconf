package config

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var safeNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

func validateSafePath(path, fieldName string) error {
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("%s must be an absolute path, got %q", fieldName, path)
	}
	cleaned := filepath.Clean(path)
	if cleaned != path {
		return fmt.Errorf("%s must not contain . or .. components", fieldName)
	}
	if strings.Contains(path, "..") {
		return fmt.Errorf("%s must not contain path traversal sequences", fieldName)
	}
	return nil
}

func validateSafeName(name, fieldName string) error {
	if !safeNamePattern.MatchString(name) {
		return fmt.Errorf("%s %q must match ^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$", fieldName, name)
	}
	return nil
}

func (c *Config) Validate() error {
	if err := c.validateBootconf(); err != nil {
		return err
	}
	if err := c.validateSystem(); err != nil {
		return err
	}
	if err := c.validateSSH(); err != nil {
		return err
	}
	if err := c.validateWifi(); err != nil {
		return err
	}
	if err := c.validateServices(); err != nil {
		return err
	}
	if err := c.validateUsers(); err != nil {
		return err
	}
	if err := c.validateFiles(); err != nil {
		return err
	}
	return nil
}

func (c *Config) validateBootconf() error {
	if !c.Bootconf.Enabled {
		return nil
	}
	if c.Bootconf.Directory == "" {
		return fmt.Errorf("bootconf.directory is required when bootconf is enabled")
	}
	if err := validateSafePath(c.Bootconf.Directory, "bootconf.directory"); err != nil {
		return err
	}
	return nil
}

func (c *Config) validateSystem() error {
	if !c.System.Enabled {
		return nil
	}
	if c.System.Hostname != "" {
		if err := validateHostname(c.System.Hostname); err != nil {
			return err
		}
	}
	if c.System.Timezone != "" {
		if err := validateTimezone(c.System.Timezone); err != nil {
			return err
		}
	}
	return nil
}

func validateHostname(hostname string) error {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9][a-zA-Z0-9.-]{0,252}[a-zA-Z0-9]$`, hostname)
	if !matched && len(hostname) != 1 {
		return fmt.Errorf("system.hostname %q is not a valid RFC 1123 hostname", hostname)
	}
	if len(hostname) > 253 {
		return fmt.Errorf("system.hostname must be at most 253 characters")
	}
	return nil
}

func validateTimezone(timezone string) error {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_+\-/]+$`, timezone)
	if !matched {
		return fmt.Errorf("system.timezone %q contains invalid characters", timezone)
	}
	return nil
}

func (c *Config) validateSSH() error {
	if !c.SSH.Enabled {
		return nil
	}
	if c.SSH.Directory == "" {
		return fmt.Errorf("ssh.directory is required when ssh is enabled")
	}
	if err := validateSafePath(c.SSH.Directory, "ssh.directory"); err != nil {
		return err
	}

	validDaemons := []string{"dropbear", "openssh"}
	if !contains(validDaemons, c.SSH.Daemon) {
		return fmt.Errorf("ssh.daemon must be 'dropbear' or 'openssh', got %q", c.SSH.Daemon)
	}

	validKeytypes := []string{"ed25519", "rsa", "ecdsa"}
	if !contains(validKeytypes, c.SSH.Keytype) {
		return fmt.Errorf("ssh.keytype must be 'ed25519', 'rsa', or 'ecdsa', got %q", c.SSH.Keytype)
	}
	return nil
}

func (c *Config) validateWifi() error {
	if !c.Wifi.Enabled {
		return nil
	}
	if c.Wifi.Directory == "" {
		return fmt.Errorf("wifi.directory is required when wifi is enabled")
	}
	if err := validateSafePath(c.Wifi.Directory, "wifi.directory"); err != nil {
		return err
	}

	var missing []string
	if c.Wifi.SSID == "" {
		missing = append(missing, "ssid")
	}
	if c.Wifi.PasswordHash == "" {
		missing = append(missing, "password_hash")
	}
	if c.Wifi.Country == "" {
		missing = append(missing, "country")
	}
	if len(missing) > 0 {
		return fmt.Errorf("wifi is enabled but missing required fields: %s", strings.Join(missing, ", "))
	}
	if err := validateSSID(c.Wifi.SSID); err != nil {
		return err
	}
	if err := validatePasswordHash(c.Wifi.PasswordHash); err != nil {
		return err
	}
	if err := validateCountryCode(c.Wifi.Country); err != nil {
		return err
	}
	return nil
}

func validateSSID(ssid string) error {
	for _, char := range ssid {
		if char < 32 || char > 126 || char == '"' || char == '\\' {
			return fmt.Errorf("wifi.ssid contains invalid characters")
		}
	}
	if len(ssid) > 32 {
		return fmt.Errorf("wifi.ssid must be at most 32 characters")
	}
	return nil
}

func validatePasswordHash(hash string) error {
	matched, _ := regexp.MatchString(`^[0-9a-f]{64}$`, hash)
	if !matched {
		return fmt.Errorf("wifi.password_hash must be exactly 64 hexadecimal characters")
	}
	return nil
}

func validateCountryCode(code string) error {
	matched, _ := regexp.MatchString(`^[A-Z]{2}$`, code)
	if !matched {
		return fmt.Errorf("wifi.country must be a 2-letter uppercase ISO 3166-1 code")
	}
	return nil
}

func (c *Config) validateServices() error {
	if !c.Services.Enabled {
		return nil
	}
	if c.Services.Directory == "" {
		return fmt.Errorf("services.directory is required when services is enabled")
	}
	if err := validateSafePath(c.Services.Directory, "services.directory"); err != nil {
		return err
	}
	for index, svc := range c.Services.Services {
		if !svc.Enabled {
			continue
		}
		if svc.Name == "" {
			return fmt.Errorf("services[%d]: enabled service must have a name", index)
		}
		if err := validateSafeName(svc.Name, fmt.Sprintf("services[%d].name", index)); err != nil {
			return err
		}
		if svc.DefaultConfig.Copy {
			if svc.DefaultConfig.Source == "" {
				return fmt.Errorf("services[%d]: service %q has copy=true but source is empty", index, svc.Name)
			}
			if err := validateSafePath(svc.DefaultConfig.Source, fmt.Sprintf("services[%d].source", index)); err != nil {
				return err
			}
			if svc.DefaultConfig.Destination == "" {
				return fmt.Errorf("services[%d]: service %q has copy=true but destination is empty", index, svc.Name)
			}
			if err := validateSafePath(svc.DefaultConfig.Destination, fmt.Sprintf("services[%d].destination", index)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Config) validateUsers() error {
	if !c.Users.Enabled {
		return nil
	}

	if c.Users.Directory == "" {
		return fmt.Errorf("users.directory is required when users is enabled")
	}
	if err := validateSafePath(c.Users.Directory, "users.directory"); err != nil {
		return err
	}

	for index, user := range c.Users.Users {
		if !user.Enabled {
			continue
		}
		if user.Name == "" {
			return fmt.Errorf("users[%d]: enabled user must have a name", index)
		}
		if err := validateUsername(user.Name, fmt.Sprintf("users[%d].name", index)); err != nil {
			return err
		}
		if user.Home == "" {
			return fmt.Errorf("users[%d]: enabled user %q must have a home", index, user.Name)
		}
		if err := validateSafePath(user.Home, fmt.Sprintf("users[%d].home", index)); err != nil {
			return err
		}
	}
	return nil
}

func validateUsername(name, fieldName string) error {
	if len(name) == 0 {
		return fmt.Errorf("%s must not be empty", fieldName)
	}
	if name[0] == '-' {
		return fmt.Errorf("%s must not start with a hyphen", fieldName)
	}
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '_' || char == '-') {
			return fmt.Errorf("%s %q must contain only lowercase letters, digits, underscores, and hyphens", fieldName, name)
		}
	}
	return nil
}

func (c *Config) validateFiles() error {
	if !c.Files.Enabled {
		return nil
	}

	for index, file := range c.Files.Files {
		if file.Source == "" {
			return fmt.Errorf("files[%d]: source is required", index)
		}
		if err := validateSafePath(file.Source, fmt.Sprintf("files[%d].source", index)); err != nil {
			return err
		}
		if file.Destination == "" {
			return fmt.Errorf("files[%d]: destination is required", index)
		}
		if err := validateSafePath(file.Destination, fmt.Sprintf("files[%d].destination", index)); err != nil {
			return err
		}
	}
	return nil
}

func contains(haystack []string, needle string) bool {
	for _, candidate := range haystack {
		if candidate == needle {
			return true
		}
	}
	return false
}
