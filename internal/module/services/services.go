package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/offline-lab/bootconf/internal/config"
	"github.com/offline-lab/bootconf/internal/module"
)

// ServicesModule manages service sentinel files and optional default config copying.
type ServicesModule struct {
	entries     []config.ServiceEntry
	enabled     bool
	servicesDir string
}

func NewServicesModule(cfg config.ServicesConfig) *ServicesModule {
	return &ServicesModule{
		entries:     cfg.Services,
		enabled:     cfg.Enabled,
		servicesDir: cfg.Directory,
	}
}

func (s *ServicesModule) Name() string {
	return "services"
}

func (s *ServicesModule) Run(_ context.Context, dryRun bool) module.Result {
	var errors []error

	for _, entry := range s.entries {
		if entry.Enabled {
			if entry.Sentinel && !dryRun {
				if err := os.MkdirAll(s.servicesDir, 0750); err != nil {
					errors = append(errors, fmt.Errorf("service %s: failed to create services dir: %w", entry.Name, err))
					continue
				}

				sentinel := filepath.Join(s.servicesDir, entry.Name)
				if err := os.WriteFile(sentinel, nil, 0640); err != nil {
					errors = append(errors, fmt.Errorf("service %s: failed to create sentinel: %w", entry.Name, err))
					continue
				}
			}

			if entry.DefaultConfig.Copy && !dryRun {
				if err := copyDefaultConfig(entry); err != nil {
					errors = append(errors, fmt.Errorf("service %s: %w", entry.Name, err))
				}
			}

		} else {
			if entry.Sentinel && !dryRun {
				sentinel := filepath.Join(s.servicesDir, entry.Name)
				if err := os.Remove(sentinel); err != nil && !os.IsNotExist(err) {
					errors = append(errors, fmt.Errorf("service %s: failed to remove sentinel: %w", entry.Name, err))
				}
			}
		}
	}

	if len(errors) > 0 {
		return module.Result{
			Section: s.Name(),
			Success: false,
			Message: fmt.Sprintf("completed with %d error(s)", len(errors)),
			Error:   formatErrors(errors),
		}
	}

	return module.Result{
		Section: s.Name(),
		Success: true,
		Message: fmt.Sprintf("processed %d service(s)", len(s.entries)),
	}
}

func copyDefaultConfig(entry config.ServiceEntry) error {
	src, err := os.Open(entry.DefaultConfig.Source)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer src.Close()

	dest := entry.DefaultConfig.Destination
	if _, err := os.Stat(dest); err == nil {
		dest = dest + ".new"
	}

	destDir := filepath.Dir(dest)
	if err := os.MkdirAll(destDir, 0750); err != nil {
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

func formatErrors(errors []error) string {
	combined := ""
	for _, serviceErr := range errors {
		if combined != "" {
			combined += "; "
		}
		combined += serviceErr.Error()
	}
	return combined
}
