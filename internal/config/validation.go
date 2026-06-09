package config

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	safeNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)
	hostnamePattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9.-]{0,251}[a-zA-Z0-9])?$`)
	timezonePattern = regexp.MustCompile(`^[a-zA-Z0-9_+\-/]+$`)
	hexHashPattern  = regexp.MustCompile(`^[0-9a-f]{64}$`)
	countryPattern  = regexp.MustCompile(`^[A-Z]{2}$`)
)

// validateSafePath prevents path traversal by ensuring paths are absolute and
// free of ".." components that could escape intended directories.
func validateSafePath(path, fieldName string) error {
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("%s must be an absolute path, got %q", fieldName, path)
	}

	if filepath.Clean(path) != path {
		return fmt.Errorf("%s must not contain . or .. components", fieldName)
	}

	return nil
}

// validateSafeName rejects names that could break shell quoting, file system
// operations, or systemd unit names by enforcing a strict alphanumeric pattern.
func validateSafeName(name, fieldName string) error {
	if !safeNamePattern.MatchString(name) {
		return fmt.Errorf("%s %q must match ^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$", fieldName, name)
	}

	return nil
}

// Validate checks every enabled section of the configuration. Sections that
// are disabled are skipped entirely, so a minimal config with only bootconf
// enabled will pass validation without requiring fields from other sections.
func (c *Config) Validate() error {
	validators := []func() error{
		c.validateBootconf,
		c.validateSystem,
		c.validateSSH,
		c.validateWifi,
		c.validateServices,
		c.validateUsers,
		c.validateFiles,
	}

	for _, fn := range validators {
		if err := fn(); err != nil {
			return err
		}
	}

	return nil
}

// validateBootconf ensures the boot configuration directory is present and
// points to a real location on disk when the section is active.
func (c *Config) validateBootconf() error {
	if !c.Bootconf.Enabled {
		return nil
	}

	if c.Bootconf.Directory == "" {
		return fmt.Errorf("bootconf.directory is required when bootconf is enabled")
	}

	return validateSafePath(c.Bootconf.Directory, "bootconf.directory")
}

// validateSystem guards against misconfigured hostnames that would break
// mDNS, DHCP, or TLS certificate matching at runtime.
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

// validateHostname enforces RFC 1123 so the device can reliably participate
// in DNS and mDNS without resolution failures.
func validateHostname(hostname string) error {
	if len(hostname) > 253 {
		return fmt.Errorf("system.hostname must be at most 253 characters")
	}

	if !hostnamePattern.MatchString(hostname) {
		return fmt.Errorf("system.hostname %q is not a valid RFC 1123 hostname", hostname)
	}

	return nil
}

// validateTimezone restricts characters to prevent injection through the
// timezone string when it is written into system configuration files.
func validateTimezone(timezone string) error {
	if !timezonePattern.MatchString(timezone) {
		return fmt.Errorf("system.timezone %q contains invalid characters", timezone)
	}

	return nil
}

// validateSSH prevents insecure or unsupported SSH daemon and key type
// selections that could leave the device with weak remote access.
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

	if !isValidEnum(c.SSH.Daemon, "dropbear", "openssh") {
		return fmt.Errorf("ssh.daemon must be 'dropbear' or 'openssh', got %q", c.SSH.Daemon)
	}

	if !isValidEnum(c.SSH.Keytype, "ed25519", "rsa", "ecdsa") {
		return fmt.Errorf("ssh.keytype must be 'ed25519', 'rsa', or 'ecdsa', got %q", c.SSH.Keytype)
	}

	return nil
}

// validateWifi ensures wireless credentials and regulatory settings are
// complete and well-formed so the device can associate on first boot.
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

	return validateCountryCode(c.Wifi.Country)
}

// validateSSID blocks control characters and quotes that could break
// wpa_supplicant or hostapd configuration file parsing.
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

// validatePasswordHash ensures the stored hash is a full SHA-256 hex digest,
// preventing truncated or malformed hashes from being written to config files.
func validatePasswordHash(hash string) error {
	if !hexHashPattern.MatchString(hash) {
		return fmt.Errorf("wifi.password_hash must be exactly 64 hexadecimal characters")
	}

	return nil
}

// validateCountryCode enforces ISO 3166-1 alpha-2 format so the regulatory
// domain is accepted by the wireless regulatory database.
func validateCountryCode(code string) error {
	if !countryPattern.MatchString(code) {
		return fmt.Errorf("wifi.country must be a 2-letter uppercase ISO 3166-1 code")
	}

	return nil
}

// validateServices checks that every enabled service has a valid name and
// that copy-config directives reference safe, absolute paths on disk.
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

		if err := validateServiceCopyConfig(svc, index); err != nil {
			return err
		}
	}

	return nil
}

// validateServiceCopyConfig ensures that when a service requests file copying,
// both source and destination paths are present and safe to use on the
// target filesystem, preventing traversal outside intended directories.
func validateServiceCopyConfig(svc ServiceEntry, index int) error {
	if !svc.DefaultConfig.Copy {
		return nil
	}

	if svc.DefaultConfig.Source == "" {
		return fmt.Errorf("services[%d]: service %q has copy=true but source is empty", index, svc.Name)
	}

	if err := validateSafePath(svc.DefaultConfig.Source, fmt.Sprintf("services[%d].source", index)); err != nil {
		return err
	}

	if svc.DefaultConfig.Destination == "" {
		return fmt.Errorf("services[%d]: service %q has copy=true but destination is empty", index, svc.Name)
	}

	return validateSafePath(svc.DefaultConfig.Destination, fmt.Sprintf("services[%d].destination", index))
}

// validateUsers ensures every enabled user has a safe name and home directory,
// preventing privilege escalation through malformed usernames or path traversal.
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

// IsValidUsername reports whether name is safe to pass to user-management
// commands. It is the bool form of the internal validateUsername check.
func IsValidUsername(name string) bool {
	return validateUsername(name, "") == nil
}

// validateUsername restricts names to a safe subset that cannot be
// misinterpreted by useradd, shadow-utils, or shell expansion, preventing
// both injection and interoperability issues across Linux distributions.
func validateUsername(name, fieldName string) error {
	if len(name) == 0 {
		return fmt.Errorf("%s must not be empty", fieldName)
	}

	if name[0] == '-' {
		return fmt.Errorf("%s must not start with a hyphen", fieldName)
	}

	for _, char := range name {
		isLower := char >= 'a' && char <= 'z'
		isDigit := char >= '0' && char <= '9'
		isUnderscore := char == '_'
		isHyphen := char == '-'

		if !(isLower || isDigit || isUnderscore || isHyphen) {
			return fmt.Errorf("%s %q must contain only lowercase letters, digits, underscores, and hyphens", fieldName, name)
		}
	}

	return nil
}

// validateFiles ensures every file mapping has absolute, traversal-free source
// and destination paths so the provisioning step cannot overwrite arbitrary
// files on the target system.
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

// isValidEnum checks that candidate matches one of the explicitly allowed
// values, preventing unrecognized or dangerous options from being accepted.
func isValidEnum(candidate string, allowed ...string) bool {
	for _, a := range allowed {
		if candidate == a {
			return true
		}
	}

	return false
}
