package templates

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/offline-lab/bootconf/internal/config"
)

func writeTemplate(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestTemplatesDisabled(t *testing.T) {
	result := New(config.TemplatesConfig{Enabled: false}).Run(context.Background(), false)
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}
	if result.Message != "templates disabled" {
		t.Errorf("unexpected message: %q", result.Message)
	}
}

func TestTemplatesDryRunNoWrites(t *testing.T) {
	sourceDir := t.TempDir()
	destDir := t.TempDir()
	sourcePath := writeTemplate(t, sourceDir, "test.tpl", "Hello {{ .name }}")
	destPath := filepath.Join(destDir, "test.conf")

	result := New(config.TemplatesConfig{
		Enabled: true,
		Templates: []config.TemplateEntry{
			{Source: sourcePath, Destination: destPath, Variables: map[string]string{"name": "world"}, Chmod: "640"},
		},
	}).Run(context.Background(), true)

	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}
	if _, err := os.Stat(destPath); err == nil {
		t.Error("dry-run must not write destination file")
	}
}

func TestTemplatesRender(t *testing.T) {
	sourceDir := t.TempDir()
	destDir := t.TempDir()
	sourcePath := writeTemplate(t, sourceDir, "config.tpl", "host={{ .host }}\nport={{ .port }}")
	destPath := filepath.Join(destDir, "config.conf")

	result := New(config.TemplatesConfig{
		Enabled: true,
		Templates: []config.TemplateEntry{
			{
				Source:      sourcePath,
				Destination: destPath,
				Variables:   map[string]string{"host": "localhost", "port": "8080"},
				Chmod:       "640",
			},
		},
	}).Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}

	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("destination not written: %v", err)
	}
	got := string(content)
	if !strings.Contains(got, "host=localhost") || !strings.Contains(got, "port=8080") {
		t.Errorf("rendered content incorrect: %q", got)
	}
}

func TestTemplatesExistingDestGetsNewSuffix(t *testing.T) {
	sourceDir := t.TempDir()
	destDir := t.TempDir()
	sourcePath := writeTemplate(t, sourceDir, "app.tpl", "value={{ .val }}")
	destPath := filepath.Join(destDir, "app.conf")

	if err := os.WriteFile(destPath, []byte("existing"), 0640); err != nil {
		t.Fatal(err)
	}

	New(config.TemplatesConfig{
		Enabled: true,
		Templates: []config.TemplateEntry{
			{Source: sourcePath, Destination: destPath, Variables: map[string]string{"val": "new"}, Chmod: "640"},
		},
	}).Run(context.Background(), false)

	if _, err := os.Stat(destPath + ".new"); err != nil {
		t.Errorf("expected dest.new to exist: %v", err)
	}
	original, _ := os.ReadFile(destPath)
	if string(original) != "existing" {
		t.Error("original file must not be overwritten")
	}
}

func TestTemplatesMissingKeyError(t *testing.T) {
	sourceDir := t.TempDir()
	destDir := t.TempDir()
	sourcePath := writeTemplate(t, sourceDir, "missing.tpl", "value={{ .missing }}")
	destPath := filepath.Join(destDir, "out.conf")

	result := New(config.TemplatesConfig{
		Enabled: true,
		Templates: []config.TemplateEntry{
			{Source: sourcePath, Destination: destPath, Variables: map[string]string{"other": "x"}, Chmod: "640"},
		},
	}).Run(context.Background(), false)

	if result.Success {
		t.Fatal("expected failure for missing template key")
	}
}

func TestTemplatesParseError(t *testing.T) {
	sourceDir := t.TempDir()
	destDir := t.TempDir()
	sourcePath := writeTemplate(t, sourceDir, "broken.tpl", "{{ .foo")
	destPath := filepath.Join(destDir, "out.conf")

	result := New(config.TemplatesConfig{
		Enabled: true,
		Templates: []config.TemplateEntry{
			{Source: sourcePath, Destination: destPath, Variables: map[string]string{}, Chmod: "640"},
		},
	}).Run(context.Background(), false)

	if result.Success {
		t.Fatal("expected failure for invalid template syntax")
	}
}

func TestTemplatesDryRunCatchesMissingKey(t *testing.T) {
	sourceDir := t.TempDir()
	destDir := t.TempDir()
	sourcePath := writeTemplate(t, sourceDir, "tpl.tpl", "{{ .notpresent }}")
	destPath := filepath.Join(destDir, "out.conf")

	// dry-run still renders (to catch errors) — missing key should fail even in dry-run.
	result := New(config.TemplatesConfig{
		Enabled: true,
		Templates: []config.TemplateEntry{
			{Source: sourcePath, Destination: destPath, Variables: map[string]string{}, Chmod: "640"},
		},
	}).Run(context.Background(), true)

	if result.Success {
		t.Fatal("dry-run should still catch missing key errors")
	}
}
