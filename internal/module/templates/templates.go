// Package templates renders Go text/template files with provided variables
// and writes the result to the destination path. Existing destinations are
// never overwritten — output is placed at dest.new instead.
package templates

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"text/template"

	"github.com/offline-lab/bootconf/internal/config"
	"github.com/offline-lab/bootconf/internal/logging"
	"github.com/offline-lab/bootconf/internal/module"
)

// TemplatesModule renders template files with config-supplied variables.
type TemplatesModule struct {
	enabled bool
	entries []config.TemplateEntry
}

// New creates a TemplatesModule from the given templates config.
func New(cfg config.TemplatesConfig) *TemplatesModule {
	return &TemplatesModule{
		enabled: cfg.Enabled,
		entries: cfg.Templates,
	}
}

// Name returns the module identifier "templates".
func (templatesModule *TemplatesModule) Name() string { return "templates" }

// Run renders each template. In dry-run mode the template is still parsed and
// executed (catching syntax and missing-key errors) but no files are written.
func (templatesModule *TemplatesModule) Run(_ context.Context, dryRun bool) module.Result {
	if !templatesModule.enabled {
		return module.Result{Section: templatesModule.Name(), Success: true, Message: "templates disabled"}
	}

	var errs []string
	for _, entry := range templatesModule.entries {
		if err := templatesModule.renderTemplate(entry, dryRun); err != nil {
			logging.Error(templatesModule.Name(), "%s → %s: %v", entry.Source, entry.Destination, err)
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return module.Result{
			Section: templatesModule.Name(),
			Success: false,
			Error:   fmt.Sprintf("%d errors: %v", len(errs), errs),
			Message: fmt.Sprintf("completed with %d errors", len(errs)),
		}
	}
	if dryRun {
		return module.Result{Section: templatesModule.Name(), Success: true, Message: fmt.Sprintf("would render %d template(s) (dry-run)", len(templatesModule.entries))}
	}
	return module.Result{Section: templatesModule.Name(), Success: true, Message: fmt.Sprintf("rendered %d template(s)", len(templatesModule.entries))}
}

func (templatesModule *TemplatesModule) renderTemplate(entry config.TemplateEntry, dryRun bool) error {
	sourceContent, err := os.ReadFile(entry.Source)
	if err != nil {
		return fmt.Errorf("read %s: %w", entry.Source, err)
	}

	// Parse and execute before any I/O — catches syntax and missing-key errors
	// in dry-run too.
	tmpl, err := template.New(filepath.Base(entry.Source)).Option("missingkey=error").Parse(string(sourceContent))
	if err != nil {
		return fmt.Errorf("parse %s: %w", entry.Source, err)
	}

	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, entry.Variables); err != nil {
		return fmt.Errorf("render %s: %w", entry.Source, err)
	}

	if dryRun {
		logging.Info(templatesModule.Name(), "would render %s → %s (dry-run)", entry.Source, entry.Destination)
		return nil
	}

	destinationPath := entry.Destination
	if _, err := os.Stat(destinationPath); err == nil {
		destinationPath = destinationPath + ".new"
	}

	logging.Info(templatesModule.Name(), "rendering %s → %s", entry.Source, destinationPath)

	if err := os.MkdirAll(filepath.Dir(destinationPath), 0750); err != nil {
		return fmt.Errorf("create parent dir for %s: %w", destinationPath, err)
	}

	if err := os.WriteFile(destinationPath, rendered.Bytes(), 0600); err != nil {
		return fmt.Errorf("write %s: %w", destinationPath, err)
	}

	mode, err := parseChmod(entry.Chmod)
	if err != nil {
		return fmt.Errorf("invalid chmod %q for %s: %w", entry.Chmod, destinationPath, err)
	}
	if err := os.Chmod(destinationPath, mode); err != nil {
		return fmt.Errorf("chmod %s: %w", destinationPath, err)
	}
	if err := os.Chown(destinationPath, 0, 0); err != nil {
		logging.Warn(templatesModule.Name(), "failed to chown %s to root: %v", destinationPath, err)
	}

	return nil
}

// parseChmod converts an octal chmod string (e.g. "640") to a FileMode.
// Values above 0777 are rejected — bootconf must never create setuid files.
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
