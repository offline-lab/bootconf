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
func (cfg *Config) Validate() error {
	validators := []func() error{
		cfg.validateBootconf,
		cfg.validateSystem,
		cfg.validateSSH,
		cfg.validateWifi,
		cfg.validateServices,
		cfg.validateUsers,
		cfg.validateFiles,
		cfg.validateTemplates,
		cfg.validateShell,
		cfg.validateUnitRun,
	}

	for _, validator := range validators {
		if err := validator(); err != nil {
			return err
		}
	}

	return nil
}

// validateBootconf ensures the boot configuration directory is present and
// points to a real location on disk when the section is active.
// Order validation (unknown/duplicate names) is handled by registry.Validate.
func (cfg *Config) validateBootconf() error {
	if !cfg.Bootconf.Enabled {
		return nil
	}

	if cfg.Bootconf.Directory == "" {
		return fmt.Errorf("bootconf.directory is required when bootconf is enabled")
	}

	return validateSafePath(cfg.Bootconf.Directory, "bootconf.directory")
}

// validateSystem guards against misconfigured hostnames that would break
// mDNS, DHCP, or TLS certificate matching at runtime.
func (cfg *Config) validateSystem() error {
	if !cfg.System.Enabled {
		return nil
	}

	if cfg.System.Hostname != "" {
		if err := validateHostname(cfg.System.Hostname); err != nil {
			return err
		}
	}

	if cfg.System.Timezone != "" {
		if err := validateTimezone(cfg.System.Timezone); err != nil {
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
func (cfg *Config) validateSSH() error {
	if !cfg.SSH.Enabled {
		return nil
	}

	if cfg.SSH.Directory == "" {
		return fmt.Errorf("ssh.directory is required when ssh is enabled")
	}

	if err := validateSafePath(cfg.SSH.Directory, "ssh.directory"); err != nil {
		return err
	}

	if !isValidEnum(cfg.SSH.Daemon, "dropbear", "openssh") {
		return fmt.Errorf("ssh.daemon must be 'dropbear' or 'openssh', got %q", cfg.SSH.Daemon)
	}

	if !isValidEnum(cfg.SSH.Keytype, "ed25519", "rsa", "ecdsa") {
		return fmt.Errorf("ssh.keytype must be 'ed25519', 'rsa', or 'ecdsa', got %q", cfg.SSH.Keytype)
	}

	return nil
}

// validateWifi ensures wireless credentials and regulatory settings are
// complete and well-formed so the device can associate on first boot.
func (cfg *Config) validateWifi() error {
	if !cfg.Wifi.Enabled {
		return nil
	}

	if cfg.Wifi.Directory == "" {
		return fmt.Errorf("wifi.directory is required when wifi is enabled")
	}

	if err := validateSafePath(cfg.Wifi.Directory, "wifi.directory"); err != nil {
		return err
	}

	var missing []string
	if cfg.Wifi.SSID == "" {
		missing = append(missing, "ssid")
	}
	if cfg.Wifi.PasswordHash == "" {
		missing = append(missing, "password_hash")
	}
	if cfg.Wifi.Country == "" {
		missing = append(missing, "country")
	}

	if len(missing) > 0 {
		return fmt.Errorf("wifi is enabled but missing required fields: %s", strings.Join(missing, ", "))
	}

	if err := validateSSID(cfg.Wifi.SSID); err != nil {
		return err
	}

	if err := validatePasswordHash(cfg.Wifi.PasswordHash); err != nil {
		return err
	}

	return validateCountryCode(cfg.Wifi.Country)
}

// validateSSID blocks control characters and quotes that could break
// wpa_supplicant or hostapd configuration file parsing.
func validateSSID(ssid string) error {
	for _, character := range ssid {
		if character < 32 || character > 126 || character == '"' || character == '\\' {
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
func (cfg *Config) validateServices() error {
	if !cfg.Services.Enabled {
		return nil
	}

	if cfg.Services.Directory == "" {
		return fmt.Errorf("services.directory is required when services is enabled")
	}

	if err := validateSafePath(cfg.Services.Directory, "services.directory"); err != nil {
		return err
	}

	for index, service := range cfg.Services.Services {
		if !service.Enabled {
			continue
		}

		if service.Name == "" {
			return fmt.Errorf("services[%d]: enabled service must have a name", index)
		}

		if err := validateSafeName(service.Name, fmt.Sprintf("services[%d].name", index)); err != nil {
			return err
		}

		if err := validateServiceCopyConfig(service, index); err != nil {
			return err
		}
	}

	return nil
}

// validateServiceCopyConfig ensures that when a service requests file copying,
// both source and destination paths are present and safe to use on the
// target filesystem, preventing traversal outside intended directories.
func validateServiceCopyConfig(service ServiceEntry, index int) error {
	if !service.DefaultConfig.Copy {
		return nil
	}

	if service.DefaultConfig.Source == "" {
		return fmt.Errorf("services[%d]: service %q has copy=true but source is empty", index, service.Name)
	}

	if err := validateSafePath(service.DefaultConfig.Source, fmt.Sprintf("services[%d].source", index)); err != nil {
		return err
	}

	if service.DefaultConfig.Destination == "" {
		return fmt.Errorf("services[%d]: service %q has copy=true but destination is empty", index, service.Name)
	}

	return validateSafePath(service.DefaultConfig.Destination, fmt.Sprintf("services[%d].destination", index))
}

// validateUsers ensures every enabled user has a safe name and home directory,
// preventing privilege escalation through malformed usernames or path traversal.
func (cfg *Config) validateUsers() error {
	if !cfg.Users.Enabled {
		return nil
	}

	if cfg.Users.Directory == "" {
		return fmt.Errorf("users.directory is required when users is enabled")
	}

	if err := validateSafePath(cfg.Users.Directory, "users.directory"); err != nil {
		return err
	}

	if cfg.Users.TmpfilesDir == "" {
		return fmt.Errorf("users.tmpfiles_dir is required when users is enabled")
	}

	if err := validateSafePath(cfg.Users.TmpfilesDir, "users.tmpfiles_dir"); err != nil {
		return err
	}

	for index, user := range cfg.Users.Users {
		if !user.Enabled {
			continue
		}

		if user.Name == "" {
			return fmt.Errorf("users[%d]: enabled user must have a name", index)
		}

		if err := validateUsername(user.Name); err != nil {
			return fmt.Errorf("users[%d].name: %w", index, err)
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
	return validateUsername(name) == nil
}

// validateUsername restricts names to a safe subset that cannot be
// misinterpreted by useradd, shadow-utils, or shell expansion, preventing
// both injection and interoperability issues across Linux distributions.
func validateUsername(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("must not be empty")
	}

	if name[0] == '-' {
		return fmt.Errorf("must not start with a hyphen")
	}

	for _, character := range name {
		isLower := character >= 'a' && character <= 'z'
		isDigit := character >= '0' && character <= '9'
		isUnderscore := character == '_'
		isHyphen := character == '-'

		if !(isLower || isDigit || isUnderscore || isHyphen) {
			return fmt.Errorf("%q must contain only lowercase letters, digits, underscores, and hyphens", name)
		}
	}

	return nil
}

// validateFiles ensures every file entry has a valid source or content (not
// both, not neither), and that all paths are absolute and traversal-free.
func (cfg *Config) validateFiles() error {
	if !cfg.Files.Enabled {
		return nil
	}

	for index, file := range cfg.Files.Files {
		if file.Source == "" && file.Content == "" {
			return fmt.Errorf("files[%d]: source or content is required", index)
		}
		if file.Source != "" && file.Content != "" {
			return fmt.Errorf("files[%d]: source and content are mutually exclusive", index)
		}
		if file.Source != "" {
			if err := validateSafePath(file.Source, fmt.Sprintf("files[%d].source", index)); err != nil {
				return err
			}
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

// validateTemplates ensures every template entry has valid source and
// destination paths so the render step cannot write to arbitrary locations.
func (cfg *Config) validateTemplates() error {
	if !cfg.Templates.Enabled {
		return nil
	}

	for index, entry := range cfg.Templates.Templates {
		if entry.Source == "" {
			return fmt.Errorf("templates[%d]: source is required", index)
		}
		if err := validateSafePath(entry.Source, fmt.Sprintf("templates[%d].source", index)); err != nil {
			return err
		}
		if entry.Destination == "" {
			return fmt.Errorf("templates[%d]: destination is required", index)
		}
		if err := validateSafePath(entry.Destination, fmt.Sprintf("templates[%d].destination", index)); err != nil {
			return err
		}
	}

	return nil
}

// validateShell ensures every command has a unique safe name and non-empty
// command body. Names are used as filenames for log and sentinel files.
func (cfg *Config) validateShell() error {
	if !cfg.Shell.Enabled {
		return nil
	}
	if cfg.Shell.Directory == "" {
		return fmt.Errorf("shell.directory is required when shell is enabled")
	}
	if err := validateSafePath(cfg.Shell.Directory, "shell.directory"); err != nil {
		return err
	}
	if containsNewline(cfg.Shell.Path) {
		return fmt.Errorf("shell.path must not contain newline characters")
	}
	for index, command := range cfg.Shell.Commands {
		if command.Name == "" {
			return fmt.Errorf("shell.commands[%d]: name is required", index)
		}
		if err := validateSafeName(command.Name, fmt.Sprintf("shell.commands[%d].name", index)); err != nil {
			return err
		}
		if command.Command == "" {
			return fmt.Errorf("shell.commands[%d]: command is required", index)
		}
	}
	return nil
}

// validateUnitRun ensures every enabled unit has a safe name and non-empty
// command body. Names are used as script filenames and systemd unit names.
func (cfg *Config) validateUnitRun() error {
	if !cfg.UnitRun.Enabled {
		return nil
	}
	if cfg.UnitRun.Directory == "" {
		return fmt.Errorf("unitrun.directory is required when unitrun is enabled")
	}
	if err := validateSafePath(cfg.UnitRun.Directory, "unitrun.directory"); err != nil {
		return err
	}
	if containsNewline(cfg.UnitRun.Path) {
		return fmt.Errorf("unitrun.path must not contain newline characters")
	}
	for index, unit := range cfg.UnitRun.Units {
		if unit.Name == "" {
			return fmt.Errorf("unitrun.units[%d]: name is required", index)
		}
		if err := validateSafeName(unit.Name, fmt.Sprintf("unitrun.units[%d].name", index)); err != nil {
			return err
		}
		if unit.Enabled && unit.Command == "" {
			return fmt.Errorf("unitrun.units[%d] %q: command is required when enabled", index, unit.Name)
		}
		for depIndex, dep := range unit.Dependencies {
			if containsNewline(dep) {
				return fmt.Errorf("unitrun.units[%d].dependencies[%d] must not contain newline characters", index, depIndex)
			}
		}
	}
	return nil
}

// containsNewline reports whether s contains a CR or LF character that could
// inject extra lines when the value is written into a unit file or environment.
func containsNewline(s string) bool {
	return strings.ContainsAny(s, "\r\n")
}

// isValidEnum checks that candidate matches one of the explicitly allowed
// values, preventing unrecognized or dangerous options from being accepted.
func isValidEnum(candidate string, allowed ...string) bool {
	for _, allowedValue := range allowed {
		if candidate == allowedValue {
			return true
		}
	}

	return false
}
