// Package system configures hostname and timezone on a readonly Linux appliance during early boot. Changes are applied via hostnamectl and timedatectl.
package system

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/offline-lab/bootconf/internal/config"
	"github.com/offline-lab/bootconf/internal/module"
)

// SystemModule sets hostname and timezone when the system section is enabled. Both fields are optional — the module is a no-op if neither is set.
type SystemModule struct {
	enabled   bool
	hostname  string
	timezone  string
	statusDir string
}

// NewSystemModule creates a SystemModule from the given system config and status directory.
func NewSystemModule(cfg config.SystemConfig, statusDir string) *SystemModule {
	return &SystemModule{
		enabled:   cfg.Enabled,
		hostname:  cfg.Hostname,
		timezone:  cfg.Timezone,
		statusDir: statusDir,
	}
}

// Name returns the module identifier "system".
func (s *SystemModule) Name() string { return "system" }

// Run applies hostname and timezone configuration if the section is enabled.
func (s *SystemModule) Run(ctx context.Context, dryRun bool) module.Result {
	if !s.enabled {
		return module.Result{Section: s.Name(), Success: true, Message: "system disabled"}
	}

	if s.hostname == "" && s.timezone == "" {
		return module.Result{Section: s.Name(), Success: true, Message: "nothing to configure"}
	}

	if !isWritable(s.statusDir) {
		return module.Result{Section: s.Name(), Success: false, Error: "basedir not writable"}
	}

	if dryRun {
		return module.Result{Section: s.Name(), Success: true, Message: "dry-run: would configure system"}
	}

	if s.hostname != "" {
		if err := runCommand(ctx, "hostnamectl", "set-hostname", s.hostname); err != nil {
			return module.Result{Section: s.Name(), Success: false, Error: fmt.Sprintf("failed to set hostname: %v", err)}
		}
	}

	if s.timezone != "" {
		if err := runCommand(ctx, "timedatectl", "set-timezone", s.timezone); err != nil {
			return module.Result{Section: s.Name(), Success: false, Error: fmt.Sprintf("failed to set timezone: %v", err)}
		}
	}

	return module.Result{Section: s.Name(), Success: true, Message: "system configured"}
}

// isWritable verifies a directory is writable by creating and removing a temporary file. Used as a pre-flight check before attempting system changes.
func isWritable(dir string) bool {
	testFile := filepath.Join(dir, ".bootconf-writable")
	if err := os.WriteFile(testFile, []byte{}, 0600); err != nil {
		return false
	}
	_ = os.Remove(testFile)
	return true
}

func runCommand(ctx context.Context, name string, args ...string) error {
	if err := exec.CommandContext(ctx, name, args...).Run(); err != nil {
		return fmt.Errorf("failed to run %s: %w", name, err)
	}
	return nil
}
