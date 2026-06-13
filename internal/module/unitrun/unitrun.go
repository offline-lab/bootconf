// Package unitrun writes shell scripts and systemd unit files from config,
// then enables them so the init system runs them at the right point in boot.
package unitrun

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/offline-lab/bootconf/internal/config"
	"github.com/offline-lab/bootconf/internal/logging"
	"github.com/offline-lab/bootconf/internal/module"
	"github.com/offline-lab/bootconf/internal/run"
)

// UnitRunModule writes a shell script and systemd unit file for each
// configured unit, then calls systemctl enable + daemon-reload so the
// init system picks up the changes without a reboot.
type UnitRunModule struct {
	enabled   bool
	directory string
	unitsDir  string
	path      string
	units     []config.UnitEntry
}

// New creates a UnitRunModule from the given unitrun config.
func New(cfg config.UnitRunConfig) *UnitRunModule {
	return &UnitRunModule{
		enabled:   cfg.Enabled,
		directory: cfg.Directory,
		unitsDir:  "/etc/systemd/system",
		path:      cfg.Path,
		units:     cfg.Units,
	}
}

// Name returns the module identifier "unitrun".
func (unitRunModule *UnitRunModule) Name() string { return "unitrun" }

// Run provisions or removes each configured unit, then reloads the systemd
// daemon once to apply all changes in a single pass.
func (unitRunModule *UnitRunModule) Run(ctx context.Context, dryRun bool, _ bool) module.Result {

	if !unitRunModule.enabled {
		return module.Result{Section: unitRunModule.Name(), Success: true, Message: "unitrun disabled"}
	}

	var errs []string

	for _, unit := range unitRunModule.units {
		if !unit.Enabled {
			if err := unitRunModule.removeUnit(ctx, unit, dryRun); err != nil {
				logging.Error(unitRunModule.Name(), "remove unit %q: %v", unit.Name, err)
				errs = append(errs, err.Error())
			}
			continue
		}

		if err := unitRunModule.provisionUnit(ctx, unit, dryRun); err != nil {
			logging.Error(unitRunModule.Name(), "provision unit %q: %v", unit.Name, err)
			errs = append(errs, err.Error())
		}
	}

	if !dryRun && len(unitRunModule.units) > 0 {
		if err := run.Command(ctx, "systemctl", "daemon-reload"); err != nil {
			logging.Warn(unitRunModule.Name(), "daemon-reload failed: %v", err)
		}
	}

	if len(errs) > 0 {
		errMsg := strings.Join(errs, "; ")

		return module.Result{
			Section: unitRunModule.Name(),
			Success: false,
			Error:   errMsg,
			Message: fmt.Sprintf("completed with %d error(s)", len(errs)),
		}
	}

	if dryRun {
		return module.Result{Section: unitRunModule.Name(), Success: true, Message: fmt.Sprintf("would provision %d unit(s) (dry-run)", len(unitRunModule.units))}
	}

	return module.Result{Section: unitRunModule.Name(), Success: true, Message: fmt.Sprintf("provisioned %d unit(s)", len(unitRunModule.units))}
}

func (unitRunModule *UnitRunModule) provisionUnit(ctx context.Context, unit config.UnitEntry, dryRun bool) error {
	scriptPath := filepath.Join(unitRunModule.directory, unit.Name+".sh")
	serviceName := "bootconf-" + unit.Name + ".service"
	serviceFilePath := filepath.Join(unitRunModule.unitsDir, serviceName)

	dependencies := unit.Dependencies

	if unit.FirstBoot {
		dependencies = append(dependencies, "ConditionFirstBoot=yes")
	}

	if dryRun {
		logging.Info(unitRunModule.Name(), "would create directory %s (dry-run)", unitRunModule.directory)
		logging.Info(unitRunModule.Name(), "would write script %s (dry-run)", scriptPath)
		logging.Info(unitRunModule.Name(), "would write unit file %s (dry-run)", serviceFilePath)

		if unit.FirstBoot {
			logging.Info(unitRunModule.Name(), "would add ConditionFirstBoot=yes to %s (dry-run)", serviceName)
		}

		logging.Info(unitRunModule.Name(), "would systemctl enable %s (dry-run)", serviceName)

		return nil
	}

	logging.Info(unitRunModule.Name(), "provisioning unit %q", unit.Name)

	if err := os.MkdirAll(unitRunModule.directory, 0750); err != nil {
		return fmt.Errorf("create script directory %s: %w", unitRunModule.directory, err)
	}

	if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\n"+unit.Command), 0750); err != nil {
		return fmt.Errorf("write script %s: %w", scriptPath, err)
	}

	logging.Info(unitRunModule.Name(), "wrote script %s", scriptPath)

	if err := os.WriteFile(serviceFilePath, []byte(renderServiceFile(unit.Name, dependencies, scriptPath, unitRunModule.path)), 0644); err != nil {
		return fmt.Errorf("write unit file %s: %w", serviceFilePath, err)
	}

	logging.Info(unitRunModule.Name(), "wrote unit file %s", serviceFilePath)

	if err := run.Command(ctx, "systemctl", "enable", serviceName); err != nil {
		return fmt.Errorf("systemctl enable %s: %w", serviceName, err)
	}

	logging.Info(unitRunModule.Name(), "enabled %s", serviceName)

	return nil
}

func (unitRunModule *UnitRunModule) removeUnit(ctx context.Context, unit config.UnitEntry, dryRun bool) error {
	scriptPath := filepath.Join(unitRunModule.directory, unit.Name+".sh")
	serviceName := "bootconf-" + unit.Name + ".service"
	serviceFilePath := filepath.Join(unitRunModule.unitsDir, serviceName)

	if dryRun {
		logging.Info(unitRunModule.Name(), "would disable %s (dry-run)", serviceName)
		logging.Info(unitRunModule.Name(), "would remove %s (dry-run)", serviceFilePath)
		logging.Info(unitRunModule.Name(), "would remove %s (dry-run)", scriptPath)

		return nil
	}

	logging.Info(unitRunModule.Name(), "removing unit %q", unit.Name)

	if err := run.Command(ctx, "systemctl", "disable", serviceName); err != nil {
		logging.Warn(unitRunModule.Name(), "failed to disable %s: %v", serviceName, err)
	}

	if err := os.Remove(serviceFilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove unit file %s: %w", serviceFilePath, err)
	}

	if err := os.Remove(scriptPath); err != nil && !os.IsNotExist(err) {
		logging.Warn(unitRunModule.Name(), "failed to remove script %s: %v", scriptPath, err)
	}

	return nil
}

// renderServiceFile produces the content of a bootconf-<name>.service unit file.
// Dependencies (After=, Before=, Conflicts=, etc.) are written verbatim into
// the [Unit] section so callers control ordering without a separate DSL.
func renderServiceFile(unitName string, dependencies []string, scriptPath string, extraPath string) string {
	var sb strings.Builder

	sb.WriteString("[Unit]\n")
	fmt.Fprintf(&sb, "Description=Bootconf Unit Task %s\n", unitName)
	sb.WriteString("DefaultDependencies=no\n")

	for _, dependency := range dependencies {
		sb.WriteString(dependency + "\n")
	}

	sb.WriteString("\n[Service]\n")
	sb.WriteString("Type=oneshot\n")
	sb.WriteString("RemainAfterExit=no\n")

	if extraPath != "" {
		fmt.Fprintf(&sb, "Environment=PATH=%s:/usr/sbin:/usr/bin:/sbin:/bin\n", extraPath)
	}

	fmt.Fprintf(&sb, "ExecStart=%s\n", scriptPath)
	sb.WriteString("\n[Install]\n")
	sb.WriteString("WantedBy=multi-user.target\n")

	return sb.String()
}
