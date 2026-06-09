package ssh

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/offline-lab/bootconf/internal/config"
	"github.com/offline-lab/bootconf/internal/module"
)

// SSHModule manages SSH host key generation and service enable/disable
// for both dropbear and openssh daemons.
type SSHModule struct {
	daemon           string
	keytype          string
	generateHostKeys bool
	enabled          bool
	sshDir           string
	servicesDir      string
}

func NewSSHModule(cfg config.SSHConfig, servicesDir string) *SSHModule {
	return &SSHModule{
		daemon:           cfg.Daemon,
		keytype:          cfg.Keytype,
		generateHostKeys: cfg.GenerateHostKeys,
		enabled:          cfg.Enabled,
		sshDir:           cfg.Directory,
		servicesDir:      servicesDir,
	}
}

func (m *SSHModule) Name() string {
	return "ssh"
}

func (m *SSHModule) Run(ctx context.Context, dryRun bool) module.Result {
	sentinelPath := filepath.Join(m.servicesDir, "ssh")

	if !m.enabled {
		if !dryRun {
			removeSentinel(sentinelPath)
		}

		return module.Result{
			Section: m.Name(),
			Success: true,
			Message: "ssh disabled",
		}
	}

	if m.daemon != "dropbear" && m.daemon != "openssh" {
		return module.Result{
			Section: m.Name(),
			Success: false,
			Error:   fmt.Sprintf("invalid daemon %q: must be \"dropbear\" or \"openssh\"", m.daemon),
		}
	}

	hostKeyPath := filepath.Join(m.sshDir, "hostkey")

	if m.generateHostKeys {
		if _, err := os.Stat(hostKeyPath); os.IsNotExist(err) {
			if dryRun {
				return module.Result{
					Section: m.Name(),
					Success: true,
					Message: "ssh enabled (dry-run: skipped host key generation)",
				}
			}

			if err := m.generateHostKey(ctx, hostKeyPath); err != nil {
				return module.Result{
					Section: m.Name(),
					Success: false,
					Error:   err.Error(),
				}
			}
		}
	}

	if !dryRun {
		if err := createSentinel(sentinelPath); err != nil {
			return module.Result{
				Section: m.Name(),
				Success: false,
				Error:   err.Error(),
			}
		}
	}

	return module.Result{
		Section: m.Name(),
		Success: true,
		Message: "ssh enabled",
	}
}

func (m *SSHModule) generateHostKey(ctx context.Context, keyPath string) error {
	if err := os.MkdirAll(m.sshDir, 0700); err != nil {
		return fmt.Errorf("failed to create ssh directory: %w", err)
	}

	var cmd *exec.Cmd

	if m.daemon == "dropbear" {
		cmd = exec.CommandContext(ctx, "dropbearkey", "-t", m.keytype, "-f", keyPath)
	} else {
		cmd = exec.CommandContext(ctx, "ssh-keygen", "-t", m.keytype, "-f", keyPath, "-N", "")
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate host key: %w", err)
	}

	return nil
}

func createSentinel(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return fmt.Errorf("failed to create services directory: %w", err)
	}

	if err := os.WriteFile(path, []byte{}, 0644); err != nil {
		return fmt.Errorf("failed to create sentinel file: %w", err)
	}

	return nil
}

func removeSentinel(path string) {
	_ = os.Remove(path)
}
