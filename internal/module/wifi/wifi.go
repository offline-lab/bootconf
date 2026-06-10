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
	"github.com/offline-lab/bootconf/internal/logging"
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

// New creates a WifiModule from the given wifi config and services directory.
func New(cfg config.WifiConfig, servicesDir string) *WifiModule {
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
func (wifiModule *WifiModule) Name() string { return "wifi" }

// Run enables or disables wifi based on configuration.
func (wifiModule *WifiModule) Run(_ context.Context, dryRun bool) module.Result {
	if !wifiModule.enabled {
		return wifiModule.disable(dryRun)
	}
	return wifiModule.enable(dryRun)
}

func (wifiModule *WifiModule) disable(dryRun bool) module.Result {
	sentinelPath := filepath.Join(wifiModule.servicesDir, "wifi")

	logging.Info(wifiModule.Name(), "disabling wifi, removing sentinel %s", sentinelPath)

	if dryRun {
		logging.Info(wifiModule.Name(), "would remove %s (dry-run)", sentinelPath)
	} else if err := os.Remove(sentinelPath); err != nil && !os.IsNotExist(err) {
		errMsg := fmt.Sprintf("failed to remove wifi sentinel %s: %v", sentinelPath, err)
		logging.Error(wifiModule.Name(), "%s", errMsg)
		return module.Result{Section: wifiModule.Name(), Success: false, Error: errMsg}
	}

	if dryRun {
		return module.Result{Section: wifiModule.Name(), Success: true, Message: "wifi disabled (dry-run)"}
	}
	return module.Result{Section: wifiModule.Name(), Success: true, Message: "wifi disabled"}
}

// enable writes the wpa_supplicant.conf and creates the wifi sentinel file.
// Existing wpa_supplicant.conf is overwritten since this is always authoritative —
// unlike service default configs, wifi credentials are managed exclusively here.
func (wifiModule *WifiModule) enable(dryRun bool) module.Result {
	configFilePath := filepath.Join(wifiModule.wifiDir, "wpa_supplicant.conf")
	sentinelPath := filepath.Join(wifiModule.servicesDir, "wifi")

	logging.Info(wifiModule.Name(), "enabling wifi for ssid %q", wifiModule.ssid)

	if dryRun {
		logging.Info(wifiModule.Name(), "would create directory %s (dry-run)", wifiModule.wifiDir)
	} else if err := os.MkdirAll(wifiModule.wifiDir, 0700); err != nil {
		errMsg := fmt.Sprintf("failed to create wifi directory %s: %v", wifiModule.wifiDir, err)
		logging.Error(wifiModule.Name(), "%s", errMsg)
		return module.Result{Section: wifiModule.Name(), Success: false, Error: errMsg}
	}

	if dryRun {
		logging.Info(wifiModule.Name(), "would write wpa_supplicant.conf to %s (dry-run)", configFilePath)
	} else {
		logging.Info(wifiModule.Name(), "writing wpa_supplicant.conf to %s", configFilePath)
		if err := os.WriteFile(configFilePath, []byte(wifiModule.renderWpaSupplicant()), 0600); err != nil {
			errMsg := fmt.Sprintf("failed to write wpa_supplicant.conf: %v", err)
			logging.Error(wifiModule.Name(), "%s", errMsg)
			return module.Result{Section: wifiModule.Name(), Success: false, Error: errMsg}
		}
	}

	if dryRun {
		logging.Info(wifiModule.Name(), "would create services directory %s and write sentinel %s (dry-run)", wifiModule.servicesDir, sentinelPath)
	} else {
		if err := os.MkdirAll(wifiModule.servicesDir, 0750); err != nil {
			errMsg := fmt.Sprintf("failed to create services directory %s: %v", wifiModule.servicesDir, err)
			logging.Error(wifiModule.Name(), "%s", errMsg)
			return module.Result{Section: wifiModule.Name(), Success: false, Error: errMsg}
		}
		logging.Info(wifiModule.Name(), "writing sentinel %s", sentinelPath)
		if err := os.WriteFile(sentinelPath, nil, 0640); err != nil {
			errMsg := fmt.Sprintf("failed to create wifi sentinel %s: %v", sentinelPath, err)
			logging.Error(wifiModule.Name(), "%s", errMsg)
			return module.Result{Section: wifiModule.Name(), Success: false, Error: errMsg}
		}
	}

	if dryRun {
		return module.Result{Section: wifiModule.Name(), Success: true, Message: "wifi enabled (dry-run)"}
	}
	return module.Result{Section: wifiModule.Name(), Success: true, Message: "wifi enabled"}
}

// renderWpaSupplicant produces the wpa_supplicant.conf content. The psk field
// uses the pre-computed hash (64 hex chars) rather than a plaintext password.
func (wifiModule *WifiModule) renderWpaSupplicant() string {
	return fmt.Sprintf(`country=%s
ctrl_interface=DIR=/var/run/wpa_supplicant GROUP=netdev
update_config=1

network={
    ssid="%s"
    psk=%s
}
`, wifiModule.country, wifiModule.ssid, wifiModule.passwordHash)
}
