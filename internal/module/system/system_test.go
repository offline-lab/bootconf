package system

import (
	"context"
	"testing"

	"github.com/offline-lab/bootconf/internal/config"
)

func TestSystemDisabled(t *testing.T) {
	statusDir := t.TempDir()

	cfg := config.SystemConfig{
		Enabled:  false,
		Hostname: "myhost",
		Timezone: "UTC",
	}

	module := New(cfg, statusDir)
	result := module.Run(context.Background(), false, false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if result.Message != "system disabled" {
		t.Errorf("expected 'system disabled', got %s", result.Message)
	}
}

func TestSystemDryRun(t *testing.T) {
	statusDir := t.TempDir()

	cfg := config.SystemConfig{
		Enabled:  true,
		Hostname: "myhost",
		Timezone: "UTC",
	}

	module := New(cfg, statusDir)
	result := module.Run(context.Background(), true, false)

	if result.Section != "system" {
		t.Errorf("expected section system, got %s", result.Section)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
}

func TestSystemNothingToConfigure(t *testing.T) {
	statusDir := t.TempDir()

	cfg := config.SystemConfig{
		Enabled: true,
	}

	module := New(cfg, statusDir)
	result := module.Run(context.Background(), false, false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if result.Message != "nothing to configure" {
		t.Errorf("expected 'nothing to configure', got %s", result.Message)
	}
}

func TestSystemHostnameOnly(t *testing.T) {
	statusDir := t.TempDir()

	cfg := config.SystemConfig{
		Enabled:  true,
		Hostname: "myhost",
	}

	module := New(cfg, statusDir)
	result := module.Run(context.Background(), true, false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
}

func TestSystemTimezoneOnly(t *testing.T) {
	statusDir := t.TempDir()

	cfg := config.SystemConfig{
		Enabled:  true,
		Timezone: "Europe/Berlin",
	}

	module := New(cfg, statusDir)
	result := module.Run(context.Background(), true, false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
}

func TestSystemDryRunNoExecute(t *testing.T) {
	statusDir := t.TempDir()

	cfg := config.SystemConfig{
		Enabled:  true,
		Hostname: "testhost",
		Timezone: "UTC",
	}

	module := New(cfg, statusDir)
	result := module.Run(context.Background(), true, false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if result.Message != "system configured (dry-run)" {
		t.Errorf("expected 'system configured (dry-run)', got %s", result.Message)
	}
}

func TestSystemStatusDirNotWritable(t *testing.T) {
	cfg := config.SystemConfig{
		Enabled:  true,
		Hostname: "myhost",
		Timezone: "UTC",
	}

	module := New(cfg, "/nonexistent/path")
	result := module.Run(context.Background(), false, false)

	if result.Success {
		t.Fatal("expected failure for non-writable status dir")
	}
	if result.Error != "status directory /nonexistent/path is not writable" {
		t.Errorf("expected writable-check error, got %s", result.Error)
	}
}

func TestSystemDisabledIgnoresUnwritableDir(t *testing.T) {
	cfg := config.SystemConfig{
		Enabled:  false,
		Hostname: "myhost",
		Timezone: "UTC",
	}

	module := New(cfg, "/nonexistent/path")
	result := module.Run(context.Background(), false, false)

	if !result.Success {
		t.Fatalf("expected success when disabled even with unwritable dir, got error: %s", result.Error)
	}
	if result.Message != "system disabled" {
		t.Errorf("expected 'system disabled', got %s", result.Message)
	}
}
