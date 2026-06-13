// Package services manages service sentinel files and optional default config
// file provisioning. Each service entry can optionally request a config file
// be copied from a source path to a destination path — never overwriting
// existing files (writes to dest.new instead).
package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/offline-lab/bootconf/internal/config"
	"github.com/offline-lab/bootconf/internal/logging"
	"github.com/offline-lab/bootconf/internal/module"
	"github.com/offline-lab/bootconf/internal/run"
)

// ServicesModule creates/removes sentinel files and copies default configs
// for each configured service entry.
type ServicesModule struct {
	entries     []config.ServiceEntry
	enabled     bool
	servicesDir string
}

// New creates a ServicesModule from the given services config.
func New(cfg config.ServicesConfig) *ServicesModule {
	return &ServicesModule{
		entries:     cfg.Services,
		enabled:     cfg.Enabled,
		servicesDir: cfg.Directory,
	}
}

// Name returns the module identifier "services".
func (servicesModule *ServicesModule) Name() string { return "services" }

// Run processes all service entries: creating/removing sentinel files and
// provisioning default config files as needed.
func (servicesModule *ServicesModule) Run(ctx context.Context, dryRun bool, apply bool) module.Result {

	if !servicesModule.enabled {
		return module.Result{Section: servicesModule.Name(), Success: true, Message: "services disabled"}
	}

	if dryRun {
		logging.Info(servicesModule.Name(), "would create services directory %s (dry-run)", servicesModule.servicesDir)

	} else if err := os.MkdirAll(servicesModule.servicesDir, 0750); err != nil {
		errMsg := fmt.Sprintf("failed to create services dir %s: %v", servicesModule.servicesDir, err)
		logging.Error(servicesModule.Name(), "%s", errMsg)

		return module.Result{Section: servicesModule.Name(), Success: false, Error: errMsg}
	}

	var errs []string

	for _, entry := range servicesModule.entries {
		errs = append(errs, servicesModule.processServiceEntry(ctx, entry, dryRun, apply)...)
	}

	if len(errs) > 0 {
		errMsg := strings.Join(errs, "; ")
		logging.Error(servicesModule.Name(), "completed with errors: %s", errMsg)

		return module.Result{
			Section: servicesModule.Name(),
			Success: false,
			Message: fmt.Sprintf("completed with %d error(s)", len(errs)),
			Error:   errMsg,
		}
	}

	return module.Result{Section: servicesModule.Name(), Success: true, Message: fmt.Sprintf("processed %d service(s)", len(servicesModule.entries))}
}

func (servicesModule *ServicesModule) processServiceEntry(ctx context.Context, entry config.ServiceEntry, dryRun bool, apply bool) []string {
	var errs []string

	if entry.Sentinel {
		errs = append(errs, servicesModule.handleSentinel(ctx, entry, dryRun, apply)...)
	}

	if entry.Enabled && entry.DefaultConfig.Copy {
		if err := provisionConfigFile(servicesModule.Name(), entry, dryRun); err != nil {
			errMsg := fmt.Sprintf("service %s: %v", entry.Name, err)
			logging.Error(servicesModule.Name(), "%s", errMsg)
			errs = append(errs, errMsg)
		}
	}

	return errs
}

func (servicesModule *ServicesModule) handleSentinel(ctx context.Context, entry config.ServiceEntry, dryRun bool, apply bool) []string {
	sentinelPath := filepath.Join(servicesModule.servicesDir, entry.Name)

	if !entry.Enabled {
		if dryRun {
			logging.Info(servicesModule.Name(), "would remove sentinel %s (dry-run)", sentinelPath)
			return nil
		}

		logging.Info(servicesModule.Name(), "removing sentinel %s", sentinelPath)

		if err := os.Remove(sentinelPath); err != nil && !os.IsNotExist(err) {
			errMsg := fmt.Sprintf("service %s: failed to remove sentinel %s: %v", entry.Name, sentinelPath, err)
			logging.Error(servicesModule.Name(), "%s", errMsg)

			return []string{errMsg}
		}

		return nil
	}

	if dryRun {
		logging.Info(servicesModule.Name(), "would write sentinel %s (dry-run)", sentinelPath)

		if apply {
			logging.Info(servicesModule.Name(), "would run systemctl start %s (dry-run)", entry.SystemdUnit())
		}

		return nil
	}

	logging.Info(servicesModule.Name(), "writing sentinel %s", sentinelPath)

	if err := os.WriteFile(sentinelPath, nil, 0640); err != nil {
		errMsg := fmt.Sprintf("service %s: failed to create sentinel %s: %v", entry.Name, sentinelPath, err)
		logging.Error(servicesModule.Name(), "%s", errMsg)

		return []string{errMsg}
	}

	if apply {
		if err := run.Command(ctx, "systemctl", "start", entry.SystemdUnit()); err != nil {
			logging.Warn(servicesModule.Name(), "systemctl start %s: %v", entry.SystemdUnit(), err)
		}
	}

	return nil
}

// provisionConfigFile copies the default config from source to destination.
// If the destination already exists, the new content is written to dest.new
// to avoid overwriting user-modified files. The file is chowned to root:root.
func provisionConfigFile(sectionName string, entry config.ServiceEntry, dryRun bool) error {
	destinationPath := entry.DefaultConfig.Destination

	if !dryRun {
		if _, err := os.Stat(destinationPath); err == nil {
			destinationPath = destinationPath + ".new"
		}
	}

	if dryRun {
		logging.Info(sectionName, "would copy %s → %s (dry-run)", entry.DefaultConfig.Source, destinationPath)
		return nil
	}

	logging.Info(sectionName, "copying %s → %s", entry.DefaultConfig.Source, destinationPath)

	sourceFile, err := os.Open(entry.DefaultConfig.Source)

	if err != nil {
		return fmt.Errorf("failed to open source %s: %w", entry.DefaultConfig.Source, err)
	}

	defer func() { _ = sourceFile.Close() }()

	if err := os.MkdirAll(filepath.Dir(destinationPath), 0750); err != nil {
		return fmt.Errorf("failed to create destination dir for %s: %w", destinationPath, err)
	}

	destinationFile, err := os.Create(destinationPath)

	if err != nil {
		return fmt.Errorf("failed to create %s: %w", destinationPath, err)
	}

	if _, err := io.Copy(destinationFile, sourceFile); err != nil {
		_ = destinationFile.Close()

		return fmt.Errorf("failed to copy to %s: %w", destinationPath, err)
	}

	if err := destinationFile.Chmod(0640); err != nil {
		_ = destinationFile.Close()

		return fmt.Errorf("failed to set permissions on %s: %w", destinationPath, err)
	}

	_ = destinationFile.Close()

	if err := os.Chown(destinationPath, 0, 0); err != nil {
		logging.Warn(sectionName, "failed to chown %s to root: %v", destinationPath, err)
	}

	return nil
}
