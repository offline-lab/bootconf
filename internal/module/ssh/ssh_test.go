package ssh

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/offline-lab/bootconf/internal/config"
)

func TestSSHEnabledCreatesSentinel(t *testing.T) {
	sshDir := t.TempDir()
	servicesDir := t.TempDir()

	cfg := config.SSHConfig{
		Daemon:           "dropbear",
		Keytype:          "ed25519",
		GenerateHostKeys: false,
		Enabled:          true,
		Directory:        sshDir,
	}

	module := New(cfg, servicesDir)
	result := module.Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	sentinel := filepath.Join(servicesDir, "ssh")
	if _, err := os.Stat(sentinel); os.IsNotExist(err) {
		t.Error("sentinel file should exist after enabling SSH")
	}
}

func TestSSHEnabledDryRun(t *testing.T) {
	sshDir := t.TempDir()
	servicesDir := t.TempDir()

	cfg := config.SSHConfig{
		Daemon:           "dropbear",
		Keytype:          "rsa",
		GenerateHostKeys: true,
		Enabled:          true,
		Directory:        sshDir,
	}

	module := New(cfg, servicesDir)
	result := module.Run(context.Background(), true)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if result.Section != "ssh" {
		t.Errorf("expected section ssh, got %s", result.Section)
	}

	sentinel := filepath.Join(servicesDir, "ssh")
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Error("sentinel file should not exist in dry-run")
	}

	hostKey := filepath.Join(sshDir, "hostkey")
	if _, err := os.Stat(hostKey); !os.IsNotExist(err) {
		t.Error("host key should not exist in dry-run")
	}
}

func TestSSHDisabledRemovesSentinel(t *testing.T) {
	servicesDir := t.TempDir()
	sentinel := filepath.Join(servicesDir, "ssh")

	if err := os.WriteFile(sentinel, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.SSHConfig{
		Daemon:    "dropbear",
		Keytype:   "rsa",
		Enabled:   false,
		Directory: t.TempDir(),
	}

	module := New(cfg, servicesDir)
	result := module.Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Error("sentinel file should have been removed")
	}
}

func TestSSHDisabledDryRun(t *testing.T) {
	servicesDir := t.TempDir()
	sentinel := filepath.Join(servicesDir, "ssh")

	if err := os.WriteFile(sentinel, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.SSHConfig{
		Daemon:    "dropbear",
		Keytype:   "rsa",
		Enabled:   false,
		Directory: t.TempDir(),
	}

	module := New(cfg, servicesDir)
	result := module.Run(context.Background(), true)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// Sentinel should still exist in dry-run — nothing is touched
	if _, err := os.Stat(sentinel); os.IsNotExist(err) {
		t.Error("sentinel file should not be removed in dry-run")
	}
}

func TestSSHInvalidDaemon(t *testing.T) {
	cfg := config.SSHConfig{
		Daemon:    "badvalue",
		Keytype:   "rsa",
		Enabled:   true,
		Directory: t.TempDir(),
	}

	module := New(cfg, t.TempDir())
	result := module.Run(context.Background(), false)

	if result.Success {
		t.Fatal("expected failure for invalid daemon")
	}
	if result.Error == "" {
		t.Error("expected non-empty error message")
	}
}

func TestSSHDisabledDoesNotValidateDaemon(t *testing.T) {
	servicesDir := t.TempDir()

	cfg := config.SSHConfig{
		Daemon:    "badvalue",
		Keytype:   "rsa",
		Enabled:   false,
		Directory: t.TempDir(),
	}

	module := New(cfg, servicesDir)
	result := module.Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success when disabled, got error: %s", result.Error)
	}
	if result.Message != "ssh disabled" {
		t.Errorf("expected 'ssh disabled', got %s", result.Message)
	}
}

func TestSSHExistingHostKeyNotRegenerated(t *testing.T) {
	sshDir := t.TempDir()
	servicesDir := t.TempDir()

	// Pre-create a dummy host key file
	hostKeyPath := filepath.Join(sshDir, "hostkey")
	if err := os.WriteFile(hostKeyPath, []byte("dummy-key"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.SSHConfig{
		Daemon:           "dropbear",
		Keytype:          "ed25519",
		GenerateHostKeys: true,
		Enabled:          true,
		Directory:        sshDir,
	}

	module := New(cfg, servicesDir)
	result := module.Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// Sentinel must still be created
	sentinel := filepath.Join(servicesDir, "ssh")
	if _, err := os.Stat(sentinel); os.IsNotExist(err) {
		t.Error("sentinel file should exist")
	}

	// Existing host key must not be overwritten
	content, err := os.ReadFile(hostKeyPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "dummy-key" {
		t.Error("existing host key should not have been regenerated")
	}
}

func TestSSHEnabledNoHostKeyGeneration(t *testing.T) {
	sshDir := t.TempDir()
	servicesDir := t.TempDir()

	cfg := config.SSHConfig{
		Daemon:           "openssh",
		Keytype:          "ed25519",
		GenerateHostKeys: false,
		Enabled:          true,
		Directory:        sshDir,
	}

	module := New(cfg, servicesDir)
	result := module.Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// Sentinel must be created
	sentinel := filepath.Join(servicesDir, "ssh")
	if _, err := os.Stat(sentinel); os.IsNotExist(err) {
		t.Error("sentinel file should exist")
	}

	// No host key file should be created
	hostKey := filepath.Join(sshDir, "hostkey")
	if _, err := os.Stat(hostKey); !os.IsNotExist(err) {
		t.Error("host key should not exist when GenerateHostKeys is false")
	}
}

func TestSSHSentinelDirCreated(t *testing.T) {
	sshDir := t.TempDir()
	// Non-existent nested path — writeSentinel must create it
	servicesDir := filepath.Join(t.TempDir(), "nested", "services")

	cfg := config.SSHConfig{
		Daemon:           "dropbear",
		Keytype:          "ed25519",
		GenerateHostKeys: false,
		Enabled:          true,
		Directory:        sshDir,
	}

	module := New(cfg, servicesDir)
	result := module.Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	sentinel := filepath.Join(servicesDir, "ssh")
	if _, err := os.Stat(sentinel); os.IsNotExist(err) {
		t.Error("sentinel file should exist with services dir auto-created")
	}

	// The parent directory itself must have been created
	info, err := os.Stat(servicesDir)
	if err != nil {
		t.Fatalf("services directory should exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("services path should be a directory")
	}
}

func TestSSHDisabledRemovesSentinelEvenIfMissing(t *testing.T) {
	servicesDir := t.TempDir()

	cfg := config.SSHConfig{
		Daemon:    "dropbear",
		Keytype:   "ed25519",
		Enabled:   false,
		Directory: t.TempDir(),
	}

	module := New(cfg, servicesDir)
	result := module.Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success when disabled with no sentinel, got error: %s", result.Error)
	}
	if result.Message != "ssh disabled" {
		t.Errorf("expected 'ssh disabled', got %s", result.Message)
	}
}
