// Package system configures hostname and timezone during early boot.
// Changes are applied via hostnamectl and timedatectl.
package system

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

// SystemModule sets hostname and timezone when the system section is enabled.
// Both fields are optional — the module is a no-op if neither is set.
type SystemModule struct {
	enabled   bool
	hostname  string
	timezone  string
	statusDir string
}

// New creates a SystemModule from the given system config and status directory.
func New(cfg config.SystemConfig, statusDir string) *SystemModule {
	return &SystemModule{
		enabled:   cfg.Enabled,
		hostname:  cfg.Hostname,
		timezone:  cfg.Timezone,
		statusDir: statusDir,
	}
}

// Name returns the module identifier "system".
func (systemModule *SystemModule) Name() string { return "system" }

// Run applies hostname and timezone configuration if the section is enabled.
func (systemModule *SystemModule) Run(ctx context.Context, dryRun bool) module.Result {
	if !systemModule.enabled {
		return module.Result{Section: systemModule.Name(), Success: true, Message: "system disabled"}
	}

	if systemModule.hostname == "" && systemModule.timezone == "" {
		return module.Result{Section: systemModule.Name(), Success: true, Message: "nothing to configure"}
	}

	if !isWritable(systemModule.statusDir) {
		err := fmt.Sprintf("status directory %s is not writable", systemModule.statusDir)
		logging.Error(systemModule.Name(), "%s", err)
		return module.Result{Section: systemModule.Name(), Success: false, Error: err}
	}

	if systemModule.hostname != "" {
		if dryRun {
			logging.Info(systemModule.Name(), "would run: hostnamectl set-hostname %q (dry-run)", systemModule.hostname)
		} else {
			logging.Info(systemModule.Name(), "setting hostname to %q", systemModule.hostname)
			if err := run.Command(ctx, "hostnamectl", "set-hostname", systemModule.hostname); err != nil {
				logging.Error(systemModule.Name(), "failed to set hostname: %v", err)
				return module.Result{Section: systemModule.Name(), Success: false, Error: fmt.Sprintf("failed to set hostname: %v", err)}
			}
		}
	}

	if systemModule.timezone != "" {
		if dryRun {
			logging.Info(systemModule.Name(), "would run: timedatectl set-timezone %q (dry-run)", systemModule.timezone)
		} else {
			logging.Info(systemModule.Name(), "setting timezone to %q", systemModule.timezone)
			if err := run.Command(ctx, "timedatectl", "set-timezone", systemModule.timezone); err != nil {
				logging.Error(systemModule.Name(), "failed to set timezone: %v", err)
				return module.Result{Section: systemModule.Name(), Success: false, Error: fmt.Sprintf("failed to set timezone: %v", err)}
			}
		}
	}

	if dryRun {
		return module.Result{Section: systemModule.Name(), Success: true, Message: "system configured (dry-run)"}
	}
	return module.Result{Section: systemModule.Name(), Success: true, Message: "system configured"}
}

// isWritable checks if a directory accepts writes by creating and removing a
// temporary probe file. Used as a pre-flight check before applying system changes
// since hostnamectl/timedatectl both require a writable /run.
func isWritable(dir string) bool {
	probeFile := filepath.Join(dir, ".bootconf-writable")
	if err := os.WriteFile(probeFile, []byte{}, 0600); err != nil {
		return false
	}
	_ = os.Remove(probeFile)
	return true
}
