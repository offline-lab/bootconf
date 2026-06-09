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

func NewFilesModule(cfg config.FilesConfig) *FilesModule {
	return &FilesModule{enabled: cfg.Enabled, entries: cfg.Files}
}

func (f *FilesModule) Name() string {
	return "files"
}

func (f *FilesModule) Run(_ context.Context, dryRun bool) module.Result {
	if !f.enabled {
		return module.Result{
			Section: "files",
			Success: true,
			Message: "files disabled",
		}
	}

	if dryRun {
		return module.Result{
			Section: "files",
			Success: true,
			Message: "dry run: skipped",
		}
	}

	var errs []string

	for _, entry := range f.entries {
		destPath := entry.Destination

		if _, err := os.Stat(destPath); err == nil {
			destPath = destPath + ".new"
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0750); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", entry.Destination, err))
			continue
		}

		src, err := os.Open(entry.Source)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", entry.Source, err))
			continue
		}

		dst, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			_ = src.Close()
			errs = append(errs, fmt.Sprintf("%s: %v", entry.Destination, err))
			continue
		}

		_, copyErr := io.Copy(dst, src)
		_ = src.Close()
		_ = dst.Close()

		if copyErr != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", entry.Destination, copyErr))
			continue
		}

		mode, err := parseChmod(entry.Chmod)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: invalid chmod %q: %v", entry.Destination, entry.Chmod, err))
			continue
		}

		if err := os.Chmod(destPath, mode); err != nil {
			errs = append(errs, fmt.Sprintf("%s: chmod: %v", entry.Destination, err))
			continue
		}

		_ = os.Chown(destPath, 0, 0)
	}

	result := module.Result{
		Section: "files",
		Success: len(errs) == 0,
	}

	if len(errs) > 0 {
		result.Error = fmt.Sprintf("%d errors: %v", len(errs), errs)
		result.Message = fmt.Sprintf("completed with %d errors", len(errs))
	} else if len(f.entries) > 0 {
		result.Message = fmt.Sprintf("copied %d file(s)", len(f.entries))
	}

	return result
}

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
