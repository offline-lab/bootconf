package wifi

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/offline-lab/bootconf/internal/config"
)

func TestWifiEnabledCreatesConfig(t *testing.T) {
	wifiDir := t.TempDir()
	servicesDir := t.TempDir()

	cfg := config.WifiConfig{
		Enabled:      true,
		SSID:         "MyNetwork",
		PasswordHash: "a1b2c3d4e5f6",
		Country:      "US",
		Directory:    wifiDir,
	}
	wifi := New(cfg, servicesDir)
	result := wifi.Run(context.Background(), false, false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	confPath := filepath.Join(wifiDir, "wpa_supplicant.conf")
	data, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("wpa_supplicant.conf not found: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, `ssid="MyNetwork"`) {
		t.Error("config missing ssid")
	}
	if !strings.Contains(content, "psk=a1b2c3d4e5f6") {
		t.Error("config missing psk")
	}
	if !strings.Contains(content, "country=US") {
		t.Error("config missing country")
	}

	sentinel := filepath.Join(servicesDir, "wifi")
	if _, err := os.Stat(sentinel); os.IsNotExist(err) {
		t.Error("sentinel file not created")
	}

	info, err := os.Stat(confPath)
	if err != nil {
		t.Fatalf("failed to stat config file: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("expected config file perms 0600, got %04o", perm)
	}
}

func TestWifiEnabledDryRun(t *testing.T) {
	wifiDir := t.TempDir()
	servicesDir := t.TempDir()

	cfg := config.WifiConfig{
		Enabled:      true,
		SSID:         "MyNetwork",
		PasswordHash: "a1b2c3d4e5f6",
		Country:      "US",
		Directory:    wifiDir,
	}
	wifi := New(cfg, servicesDir)
	result := wifi.Run(context.Background(), true, false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	confPath := filepath.Join(wifiDir, "wpa_supplicant.conf")
	if _, err := os.Stat(confPath); !os.IsNotExist(err) {
		t.Error("wpa_supplicant.conf should not exist in dry-run")
	}

	sentinel := filepath.Join(servicesDir, "wifi")
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Error("sentinel should not exist in dry-run")
	}
}

func TestWifiDisabledRemovesSentinel(t *testing.T) {
	servicesDir := t.TempDir()
	sentinel := filepath.Join(servicesDir, "wifi")
	if err := os.WriteFile(sentinel, nil, 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.WifiConfig{
		Enabled:   false,
		Directory: t.TempDir(),
	}
	wifi := New(cfg, servicesDir)
	result := wifi.Run(context.Background(), false, false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Error("sentinel should have been removed")
	}
}

func TestWifiDisabledDryRun(t *testing.T) {
	cfg := config.WifiConfig{
		Enabled:   false,
		Directory: t.TempDir(),
	}
	wifi := New(cfg, t.TempDir())
	result := wifi.Run(context.Background(), true, false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	if !strings.Contains(result.Message, "dry-run") {
		t.Errorf("expected dry-run in message, got: %s", result.Message)
	}
}

func TestWifiDisabledNoSentinelToRemove(t *testing.T) {
	servicesDir := t.TempDir()

	cfg := config.WifiConfig{
		Enabled:   false,
		Directory: t.TempDir(),
	}
	wifi := New(cfg, servicesDir)
	result := wifi.Run(context.Background(), false, false)

	if !result.Success {
		t.Fatalf("expected success when no sentinel exists, got error: %s", result.Error)
	}

	if result.Message != "wifi disabled" {
		t.Errorf("expected 'wifi disabled' message, got: %s", result.Message)
	}
}

func TestWifiConfigContent(t *testing.T) {
	wifiDir := t.TempDir()
	servicesDir := t.TempDir()

	cfg := config.WifiConfig{
		Enabled:      true,
		SSID:         "TestSSID",
		PasswordHash: "deadbeef1234",
		Country:      "DE",
		Directory:    wifiDir,
	}
	wifi := New(cfg, servicesDir)
	result := wifi.Run(context.Background(), false, false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	data, err := os.ReadFile(filepath.Join(wifiDir, "wpa_supplicant.conf"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)

	for _, want := range []string{
		"country=DE",
		"ctrl_interface=DIR=/var/run/wpa_supplicant GROUP=netdev",
		"update_config=1",
		`ssid="TestSSID"`,
		"psk=deadbeef1234",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("config missing %q", want)
		}
	}
}

func TestWifiEnabledCreatesDirs(t *testing.T) {
	baseDir := t.TempDir()
	wifiDir := filepath.Join(baseDir, "wifi", "nested")
	servicesDir := filepath.Join(baseDir, "services", "nested")

	cfg := config.WifiConfig{
		Enabled:      true,
		SSID:         "TestNet",
		PasswordHash: "hash123",
		Country:      "GB",
		Directory:    wifiDir,
	}
	wifi := New(cfg, servicesDir)
	result := wifi.Run(context.Background(), false, false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	wifiInfo, err := os.Stat(wifiDir)
	if err != nil {
		t.Fatalf("wifi directory not created: %v", err)
	}
	if !wifiInfo.IsDir() {
		t.Fatal("wifi path is not a directory")
	}

	servicesInfo, err := os.Stat(servicesDir)
	if err != nil {
		t.Fatalf("services directory not created: %v", err)
	}
	if !servicesInfo.IsDir() {
		t.Fatal("services path is not a directory")
	}
}

func TestWifiDisabledRemovesConfigSentinelOnly(t *testing.T) {
	wifiDir := t.TempDir()
	servicesDir := t.TempDir()

	confPath := filepath.Join(wifiDir, "wpa_supplicant.conf")
	originalContent := []byte("original wpa config content\n")
	if err := os.WriteFile(confPath, originalContent, 0600); err != nil {
		t.Fatal(err)
	}

	sentinel := filepath.Join(servicesDir, "wifi")
	if err := os.WriteFile(sentinel, nil, 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.WifiConfig{
		Enabled:   false,
		Directory: wifiDir,
	}
	wifi := New(cfg, servicesDir)
	result := wifi.Run(context.Background(), false, false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Error("sentinel should have been removed")
	}

	data, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("wpa_supplicant.conf should still exist: %v", err)
	}
	if string(data) != string(originalContent) {
		t.Errorf("wpa_supplicant.conf was modified: got %q, want %q", string(data), string(originalContent))
	}
}
