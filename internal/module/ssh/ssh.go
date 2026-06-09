// Package ssh manages SSH host key generation and the sentinel file that
// signals whether SSH should be enabled. Bootconf does not start or stop
// daemons directly — instead it creates/removes sentinel files in the
// services directory, and the init system reads those to decide what to run.
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
//
// When enabled, a sentinel file is created at <servicesDir>/ssh. The init
// system checks for this file to decide whether to start the SSH daemon.
// When disabled, the sentinel is removed, signaling the init system to
// not start SSH.
//
// Host keys are generated once and reused on subsequent boots. The key
// material is written to <sshDir>/hostkey with mode 0600.
type SSHModule struct {
	daemon           string
	keytype          string
	generateHostKeys bool
	enabled          bool
	sshDir           string
	servicesDir      string
}

// NewSSHModule creates an SSHModule from the given SSH config and services directory.
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

// Name returns the module identifier "ssh".
func (m *SSHModule) Name() string { return "ssh" }

// Run enables or disables SSH based on configuration, generating host keys if needed.
func (m *SSHModule) Run(ctx context.Context, dryRun bool) module.Result {
	sentinelPath := filepath.Join(m.servicesDir, "ssh")

	if !m.enabled {
		if !dryRun {
			removeSentinel(sentinelPath)
		}

		return module.Result{Section: m.Name(), Success: true, Message: "ssh disabled"}
	}

	if m.daemon != "dropbear" && m.daemon != "openssh" {
		return module.Result{Section: m.Name(), Success: false, Error: fmt.Sprintf("invalid daemon %q: must be \"dropbear\" or \"openssh\"", m.daemon)}
	}

	hostKeyPath := filepath.Join(m.sshDir, "hostkey")

	if m.generateHostKeys {
		if _, err := os.Stat(hostKeyPath); os.IsNotExist(err) {
			if dryRun {
				return module.Result{Section: m.Name(), Success: true, Message: "ssh enabled (dry-run: skipped host key generation)"}
			}

			if err := m.generateHostKey(ctx, hostKeyPath); err != nil {
				return module.Result{Section: m.Name(), Success: false, Error: err.Error()}
			}
		}
	}

	if !dryRun {
		if err := writeSentinel(sentinelPath); err != nil {
			return module.Result{Section: m.Name(), Success: false, Error: err.Error()}
		}
	}

	return module.Result{Section: m.Name(), Success: true, Message: "ssh enabled"}
}

// generateHostKey creates the SSH host key directory and generates a new key using the daemon-appropriate tool (dropbearkey or ssh-keygen).
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

// writeSentinel creates the sentinel file and its parent directory. The sentinel signals the init system to enable this service.
func writeSentinel(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return fmt.Errorf("failed to create services directory: %w", err)
	}

	if err := os.WriteFile(path, []byte{}, 0640); err != nil {
		return fmt.Errorf("failed to create sentinel file: %w", err)
	}

	return nil
}

// removeSentinel removes the sentinel file, signaling the init system to disable this service. Missing files are silently ignored.
func removeSentinel(path string) {
	_ = os.Remove(path)
}
