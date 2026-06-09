// Package files copies configured files from source to destination paths.
// Existing files are never overwritten — new content goes to dest.new instead.
// This is the "bring your own config" mechanism for the readonly appliance.
package files

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/offline-lab/bootconf/internal/config"
	"github.com/offline-lab/bootconf/internal/module"
)

// FilesModule copies configured files into the basedir, never overwriting
// existing entries. If a destination already exists, the content is written
// to dest + ".new" instead.
type FilesModule struct {
	enabled bool
	entries []config.FileEntry
}

// NewFilesModule creates a FilesModule from the given files config.
func NewFilesModule(cfg config.FilesConfig) *FilesModule {
	return &FilesModule{enabled: cfg.Enabled, entries: cfg.Files}
}

// Name returns the module identifier "files".
func (f *FilesModule) Name() string { return "files" }

// Run copies all configured files, never overwriting existing content.
func (f *FilesModule) Run(_ context.Context, dryRun bool) module.Result {
	if !f.enabled {
		return module.Result{Section: f.Name(), Success: true, Message: "files disabled"}
	}

	if dryRun {
		return module.Result{Section: f.Name(), Success: true, Message: "dry run: skipped"}
	}

	var errs []string

	for _, entry := range f.entries {
		if err := f.copyEntry(entry); err != nil {
			errs = append(errs, err.Error())
		}
	}

	result := module.Result{Section: f.Name(), Success: len(errs) == 0}

	if len(errs) > 0 {
		result.Error = fmt.Sprintf("%d errors: %v", len(errs), errs)
		result.Message = fmt.Sprintf("completed with %d errors", len(errs))
	} else if len(f.entries) > 0 {
		result.Message = fmt.Sprintf("copied %d file(s)", len(f.entries))
	}

	return result
}

// copyEntry copies a single file entry. If the destination already exists, the new content is written to dest.new. The file is chowned to root:root.
func (f *FilesModule) copyEntry(entry config.FileEntry) error {
	destPath := entry.Destination

	if _, err := os.Stat(destPath); err == nil {
		destPath = destPath + ".new"
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0750); err != nil {
		return fmt.Errorf("%s: %v", entry.Destination, err)
	}

	src, err := os.Open(entry.Source)
	if err != nil {
		return fmt.Errorf("%s: %v", entry.Source, err)
	}
	defer func() { _ = src.Close() }()

	dst, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("%s: %v", entry.Destination, err)
	}

	_, copyErr := io.Copy(dst, src)
	_ = dst.Close()

	if copyErr != nil {
		return fmt.Errorf("%s: %v", entry.Destination, copyErr)
	}

	mode, err := parseChmod(entry.Chmod)
	if err != nil {
		return fmt.Errorf("%s: invalid chmod %q: %v", entry.Destination, entry.Chmod, err)
	}

	if err := os.Chmod(destPath, mode); err != nil {
		return fmt.Errorf("%s: chmod: %v", entry.Destination, err)
	}

	_ = os.Chown(destPath, 0, 0)

	return nil
}

// parseChmod converts an octal chmod string (e.g. "640") to a FileMode. Rejects setuid/setgid/sticky bits — this tool should not create suid files.
func parseChmod(s string) (os.FileMode, error) {
	value, err := strconv.ParseUint(s, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse chmod: %w", err)
	}
	if value > 0777 {
		return 0, fmt.Errorf("chmod must not include setuid/setgid/sticky bits")
	}
	return os.FileMode(value), nil
}
