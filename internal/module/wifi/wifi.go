// Package wifi generates wpa_supplicant.conf from a pre-hashed password
// (SHA-256 PSK hash as produced by wpa_passphrase). The password is never
// stored in cleartext — only its hash appears in the config file.
package wifi

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/offline-lab/bootconf/internal/config"
	"github.com/offline-lab/bootconf/internal/module"
)

// WifiModule configures wireless networking by writing a wpa_supplicant.conf
// and creating a sentinel file to signal the init system.
type WifiModule struct {
	enabled      bool
	ssid         string
	passwordHash string
	country      string
	wifiDir      string
	servicesDir  string
}

// NewWifiModule creates a WifiModule from the given wifi config and services directory.
func NewWifiModule(cfg config.WifiConfig, servicesDir string) *WifiModule {
	return &WifiModule{
		enabled:      cfg.Enabled,
		ssid:         cfg.SSID,
		passwordHash: cfg.PasswordHash,
		country:      cfg.Country,
		wifiDir:      cfg.Directory,
		servicesDir:  servicesDir,
	}
}

// Name returns the module identifier "wifi".
func (m *WifiModule) Name() string { return "wifi" }

// Run enables or disables wifi based on configuration.
func (m *WifiModule) Run(_ context.Context, dryRun bool) module.Result {
	if !m.enabled {
		return m.disable(dryRun)
	}

	return m.enable(dryRun)
}

func (m *WifiModule) disable(dryRun bool) module.Result {
	sentinel := filepath.Join(m.servicesDir, "wifi")

	if dryRun {
		return module.Result{Section: m.Name(), Success: true, Message: "wifi disabled (dry-run)"}
	}

	if err := os.Remove(sentinel); err != nil && !os.IsNotExist(err) {
		return module.Result{Section: m.Name(), Success: false, Error: fmt.Errorf("failed to remove wifi sentinel: %w", err).Error()}
	}

	return module.Result{Section: m.Name(), Success: true, Message: "wifi disabled"}
}

// enable writes the wpa_supplicant.conf and creates the wifi sentinel file.
// If either directory does not exist, it is created.
func (m *WifiModule) enable(dryRun bool) module.Result {
	confPath := filepath.Join(m.wifiDir, "wpa_supplicant.conf")

	if dryRun {
		return module.Result{Section: m.Name(), Success: true, Message: "wifi enabled (dry-run)"}
	}

	if err := os.MkdirAll(m.wifiDir, 0700); err != nil {
		return module.Result{Section: m.Name(), Success: false, Error: fmt.Errorf("failed to create wifi directory: %w", err).Error()}
	}

	if err := os.WriteFile(confPath, []byte(m.renderWpaSupplicant()), 0600); err != nil {
		return module.Result{Section: m.Name(), Success: false, Error: fmt.Errorf("failed to write wpa_supplicant.conf: %w", err).Error()}
	}

	if err := os.MkdirAll(m.servicesDir, 0750); err != nil {
		return module.Result{Section: m.Name(), Success: false, Error: fmt.Errorf("failed to create services directory: %w", err).Error()}
	}

	sentinel := filepath.Join(m.servicesDir, "wifi")

	if err := os.WriteFile(sentinel, nil, 0640); err != nil {
		return module.Result{Section: m.Name(), Success: false, Error: fmt.Errorf("failed to create wifi sentinel: %w", err).Error()}
	}

	return module.Result{Section: m.Name(), Success: true, Message: "wifi enabled"}
}

// renderWpaSupplicant produces the wpa_supplicant.conf content. The psk field
// uses the pre-computed hash (64 hex chars) rather than a plaintext password.
func (m *WifiModule) renderWpaSupplicant() string {
	return fmt.Sprintf(`country=%s
ctrl_interface=DIR=/var/run/wpa_supplicant GROUP=netdev
update_config=1

network={
    ssid="%s"
    psk=%s
}
`, m.country, m.ssid, m.passwordHash)
}
