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

type SystemModule struct {
	enabled   bool
	hostname  string
	timezone  string
	statusDir string
}

func NewSystemModule(cfg config.SystemConfig, statusDir string) *SystemModule {
	return &SystemModule{
		enabled:   cfg.Enabled,
		hostname:  cfg.Hostname,
		timezone:  cfg.Timezone,
		statusDir: statusDir,
	}
}

func (s *SystemModule) Name() string {
	return "system"
}

func (s *SystemModule) Run(_ context.Context, dryRun bool) module.Result {
	if !s.enabled {
		return module.Result{
			Section: s.Name(),
			Success: true,
			Message: "system disabled",
		}
	}

	if s.hostname == "" && s.timezone == "" {
		return module.Result{
			Section: s.Name(),
			Success: true,
			Message: "nothing to configure",
		}
	}

	if !isWritable(s.statusDir) {
		return module.Result{
			Section: s.Name(),
			Success: false,
			Error:   "basedir not writable",
		}
	}

	if dryRun {
		return module.Result{
			Section: s.Name(),
			Success: true,
			Message: "dry-run: would configure system",
		}
	}

	if s.hostname != "" {
		if err := runCommand("hostnamectl", "set-hostname", s.hostname); err != nil {
			return module.Result{
				Section: s.Name(),
				Success: false,
				Error:   fmt.Sprintf("failed to set hostname: %v", err),
			}
		}
	}

	if s.timezone != "" {
		if err := runCommand("timedatectl", "set-timezone", s.timezone); err != nil {
			return module.Result{
				Section: s.Name(),
				Success: false,
				Error:   fmt.Sprintf("failed to set timezone: %v", err),
			}
		}
	}

	return module.Result{
		Section: s.Name(),
		Success: true,
		Message: "system configured",
	}
}

func isWritable(dir string) bool {
	testFile := filepath.Join(dir, ".bootconf-writable")
	if err := os.WriteFile(testFile, []byte{}, 0600); err != nil {
		return false
	}
	_ = os.Remove(testFile)
	return true
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %s: %w", name, err)
	}
	return nil
}
