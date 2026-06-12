// Package files copies configured files from source to destination paths.
// Existing files are never overwritten — new content goes to dest.new instead.
package files

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/offline-lab/bootconf/internal/config"
	"github.com/offline-lab/bootconf/internal/logging"
	"github.com/offline-lab/bootconf/internal/module"
)

// FilesModule copies configured files into the basedir, never overwriting
// existing entries. If a destination already exists, the content is written
// to dest + ".new" instead.
type FilesModule struct {
	enabled bool
	entries []config.FileEntry
}

// New creates a FilesModule from the given files config.
func New(cfg config.FilesConfig) *FilesModule {
	return &FilesModule{enabled: cfg.Enabled, entries: cfg.Files}
}

// Name returns the module identifier "files".
func (filesModule *FilesModule) Name() string { return "files" }

// Run copies all configured files, never overwriting existing content.
func (filesModule *FilesModule) Run(_ context.Context, dryRun bool) module.Result {
	if !filesModule.enabled {
		return module.Result{Section: filesModule.Name(), Success: true, Message: "files disabled"}
	}

	var errs []string

	for _, entry := range filesModule.entries {
		if err := filesModule.copyEntry(entry, dryRun); err != nil {
			logging.Error(filesModule.Name(), "copy %s → %s: %v", entry.Source, entry.Destination, err)
			errs = append(errs, err.Error())
		}
	}

	result := module.Result{Section: filesModule.Name(), Success: len(errs) == 0}

	if len(errs) > 0 {
		result.Error = fmt.Sprintf("%d errors: %v", len(errs), errs)
		result.Message = fmt.Sprintf("completed with %d errors", len(errs))
	} else if len(filesModule.entries) > 0 {
		if dryRun {
			result.Message = fmt.Sprintf("would write %d file(s) (dry-run)", len(filesModule.entries))
		} else {
			result.Message = fmt.Sprintf("wrote %d file(s)", len(filesModule.entries))
		}
	}

	return result
}

func (filesModule *FilesModule) copyEntry(entry config.FileEntry, dryRun bool) error {
	destinationPath := entry.Destination

	// Determine final path before any I/O so dry-run shows the actual target.
	if !dryRun {
		if _, err := os.Stat(destinationPath); err == nil {
			destinationPath = destinationPath + ".new"
		}
	}

	if dryRun {
		if entry.Source != "" {
			logging.Info(filesModule.Name(), "would copy %s → %s (chmod %s) (dry-run)", entry.Source, destinationPath, entry.Chmod)
		} else {
			logging.Info(filesModule.Name(), "would write content → %s (chmod %s) (dry-run)", destinationPath, entry.Chmod)
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(destinationPath), 0750); err != nil {
		return fmt.Errorf("%s: create parent dir: %w", entry.Destination, err)
	}

	if entry.Source != "" {
		logging.Info(filesModule.Name(), "copying %s → %s", entry.Source, destinationPath)

		sourceFile, err := os.Open(entry.Source)
		if err != nil {
			return fmt.Errorf("%s: open source: %w", entry.Source, err)
		}
		defer func() { _ = sourceFile.Close() }()

		// Create with 0600; Chmod below sets the final permissions from config.
		destinationFile, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		if err != nil {
			return fmt.Errorf("%s: create destination: %w", entry.Destination, err)
		}

		_, copyErr := io.Copy(destinationFile, sourceFile)
		_ = destinationFile.Close()
		if copyErr != nil {
			return fmt.Errorf("%s: copy: %w", entry.Destination, copyErr)
		}

	} else {
		logging.Info(filesModule.Name(), "writing content → %s", destinationPath)

		if err := os.WriteFile(destinationPath, []byte(entry.Content), 0600); err != nil {
			return fmt.Errorf("%s: write content: %w", entry.Destination, err)
		}
	}

	mode, err := parseChmod(entry.Chmod)
	if err != nil {
		return fmt.Errorf("%s: invalid chmod %q: %w", entry.Destination, entry.Chmod, err)
	}

	if err := os.Chmod(destinationPath, mode); err != nil {
		return fmt.Errorf("%s: chmod: %w", entry.Destination, err)
	}

	if err := os.Chown(destinationPath, 0, 0); err != nil {
		logging.Warn(filesModule.Name(), "failed to chown %s to root: %v", destinationPath, err)
	}

	return nil
}

// parseChmod converts an octal chmod string (e.g. "640") to a FileMode.
// Values above 0777 are rejected — this tool must never create setuid files.
func parseChmod(octalStr string) (os.FileMode, error) {
	value, err := strconv.ParseUint(octalStr, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse %q as octal: %w", octalStr, err)
	}
	if value > 0777 {
		return 0, fmt.Errorf("chmod %q must not include setuid/setgid/sticky bits", octalStr)
	}
	return os.FileMode(value), nil
}
