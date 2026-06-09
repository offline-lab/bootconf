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
	"github.com/offline-lab/bootconf/internal/module"
)

// ServicesModule creates/removes sentinel files and copies default configs
// for each configured service entry.
type ServicesModule struct {
	entries     []config.ServiceEntry
	enabled     bool
	servicesDir string
}

// NewServicesModule creates a ServicesModule from the given services config.
func NewServicesModule(cfg config.ServicesConfig) *ServicesModule {
	return &ServicesModule{
		entries:     cfg.Services,
		enabled:     cfg.Enabled,
		servicesDir: cfg.Directory,
	}
}

// Name returns the module identifier "services".
func (s *ServicesModule) Name() string { return "services" }

// Run processes all service entries: creating/removing sentinel files and
// provisioning default config files as needed.
func (s *ServicesModule) Run(_ context.Context, dryRun bool) module.Result {
	if !s.enabled {
		return module.Result{Section: s.Name(), Success: true, Message: "services disabled"}
	}

	if !dryRun {
		if err := os.MkdirAll(s.servicesDir, 0750); err != nil {
			return module.Result{Section: s.Name(), Success: false, Error: fmt.Sprintf("failed to create services dir: %v", err)}
		}
	}

	var errs []string

	for _, entry := range s.entries {
		if !dryRun && entry.Sentinel {
			sentinel := filepath.Join(s.servicesDir, entry.Name)

			if entry.Enabled {
				if err := os.WriteFile(sentinel, nil, 0640); err != nil {
					errs = append(errs, fmt.Sprintf("service %s: failed to create sentinel: %v", entry.Name, err))
					continue
				}
			} else {
				if err := os.Remove(sentinel); err != nil && !os.IsNotExist(err) {
					errs = append(errs, fmt.Sprintf("service %s: failed to remove sentinel: %v", entry.Name, err))
				}
			}
		}

		if !dryRun && entry.Enabled && entry.DefaultConfig.Copy {
			if err := provisionConfigFile(entry); err != nil {
				errs = append(errs, fmt.Sprintf("service %s: %v", entry.Name, err))
			}
		}
	}

	if len(errs) > 0 {
		return module.Result{
			Section: s.Name(),
			Success: false,
			Message: fmt.Sprintf("completed with %d error(s)", len(errs)),
			Error:   strings.Join(errs, "; "),
		}
	}

	return module.Result{Section: s.Name(), Success: true, Message: fmt.Sprintf("processed %d service(s)", len(s.entries))}
}

// provisionConfigFile copies the default config from source to destination.
// If the destination already exists, the new content is written to dest.new
// to avoid overwriting user-modified files. The file is chowned to root:root.
func provisionConfigFile(entry config.ServiceEntry) error {
	src, err := os.Open(entry.DefaultConfig.Source)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}

	defer func() { _ = src.Close() }()

	dest := entry.DefaultConfig.Destination

	if _, err := os.Stat(dest); err == nil {
		dest = dest + ".new"
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0750); err != nil {
		return fmt.Errorf("failed to create destination dir: %w", err)
	}

	dst, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}

	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		return fmt.Errorf("failed to copy: %w", err)
	}

	if err := dst.Chmod(0640); err != nil {
		_ = dst.Close()
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	_ = dst.Close()
	_ = os.Chown(dest, 0, 0)

	return nil
}
