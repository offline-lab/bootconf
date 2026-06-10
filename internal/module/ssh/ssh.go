// Package ssh manages SSH host key generation and the sentinel file that
// signals whether SSH should be enabled. Bootconf does not start or stop
// daemons directly — instead it creates/removes sentinel files in the
// services directory, and the init system reads those to decide what to run.
package ssh

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/offline-lab/bootconf/internal/config"
	"github.com/offline-lab/bootconf/internal/logging"
	"github.com/offline-lab/bootconf/internal/module"
	"github.com/offline-lab/bootconf/internal/run"
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

// New creates an SSHModule from the given SSH config and services directory.
func New(cfg config.SSHConfig, servicesDir string) *SSHModule {
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
func (sshModule *SSHModule) Name() string { return "ssh" }

// Run enables or disables SSH based on configuration, generating host keys if needed.
func (sshModule *SSHModule) Run(ctx context.Context, dryRun bool) module.Result {
	sentinelPath := filepath.Join(sshModule.servicesDir, "ssh")

	if !sshModule.enabled {
		logging.Info(sshModule.Name(), "disabling ssh, removing sentinel %s", sentinelPath)
		if dryRun {
			logging.Info(sshModule.Name(), "would remove %s (dry-run)", sentinelPath)
		} else {
			if err := os.Remove(sentinelPath); err != nil && !os.IsNotExist(err) {
				logging.Warn(sshModule.Name(), "failed to remove sentinel %s: %v", sentinelPath, err)
			}
		}
		return module.Result{Section: sshModule.Name(), Success: true, Message: "ssh disabled"}
	}

	if sshModule.daemon != "dropbear" && sshModule.daemon != "openssh" {
		err := fmt.Sprintf("invalid daemon %q: must be dropbear or openssh", sshModule.daemon)
		logging.Error(sshModule.Name(), "%s", err)
		return module.Result{Section: sshModule.Name(), Success: false, Error: err}
	}

	hostKeyPath := filepath.Join(sshModule.sshDir, "hostkey")

	if sshModule.generateHostKeys {
		if _, err := os.Stat(hostKeyPath); os.IsNotExist(err) {
			if dryRun {
				logging.Info(sshModule.Name(), "would generate host key at %s using %s (dry-run)", hostKeyPath, sshModule.daemon)
			} else {
				logging.Info(sshModule.Name(), "generating host key at %s using %s", hostKeyPath, sshModule.daemon)
				if err := sshModule.generateHostKey(ctx, hostKeyPath); err != nil {
					logging.Error(sshModule.Name(), "failed to generate host key: %v", err)
					return module.Result{Section: sshModule.Name(), Success: false, Error: err.Error()}
				}
			}
		} else {
			logging.Debug(sshModule.Name(), "host key already exists at %s, skipping generation", hostKeyPath)
		}
	}

	if dryRun {
		logging.Info(sshModule.Name(), "would write sentinel %s (dry-run)", sentinelPath)
	} else {
		logging.Info(sshModule.Name(), "writing sentinel %s", sentinelPath)
		if err := writeSentinel(sentinelPath); err != nil {
			logging.Error(sshModule.Name(), "failed to write sentinel: %v", err)
			return module.Result{Section: sshModule.Name(), Success: false, Error: err.Error()}
		}
	}

	if dryRun {
		return module.Result{Section: sshModule.Name(), Success: true, Message: "ssh enabled (dry-run)"}
	}
	return module.Result{Section: sshModule.Name(), Success: true, Message: "ssh enabled"}
}

func (sshModule *SSHModule) generateHostKey(ctx context.Context, keyPath string) error {
	if err := os.MkdirAll(sshModule.sshDir, 0700); err != nil {
		return fmt.Errorf("create ssh directory %s: %w", sshModule.sshDir, err)
	}

	if sshModule.daemon == "dropbear" {
		return run.Command(ctx, "dropbearkey", "-t", sshModule.keytype, "-f", keyPath)
	}
	return run.Command(ctx, "ssh-keygen", "-t", sshModule.keytype, "-f", keyPath, "-N", "")
}

// writeSentinel creates the sentinel file signalling the init system to enable
// the SSH daemon. The parent directory is created if it does not yet exist.
func writeSentinel(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return fmt.Errorf("create services directory: %w", err)
	}
	if err := os.WriteFile(path, []byte{}, 0640); err != nil {
		return fmt.Errorf("write sentinel %s: %w", path, err)
	}
	return nil
}
