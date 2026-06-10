package shell

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/offline-lab/bootconf/internal/config"
)

func TestShellDisabled(t *testing.T) {
	result := New(config.ShellConfig{Enabled: false}).Run(context.Background(), false)
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}
	if result.Message != "shell disabled" {
		t.Errorf("unexpected message: %q", result.Message)
	}
}

func TestShellDryRunNoWrites(t *testing.T) {
	workDir := t.TempDir()

	result := New(config.ShellConfig{
		Enabled:   true,
		Directory: workDir,
		Commands: []config.ShellCommand{
			{Name: "test-cmd", AllowFail: false, Command: "echo hello"},
		},
	}).Run(context.Background(), true)

	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}
	if result.Message != "shell configured (dry-run)" {
		t.Errorf("unexpected message: %q", result.Message)
	}
	entries, _ := os.ReadDir(workDir)
	if len(entries) != 0 {
		t.Errorf("dry-run must not write any files, found: %v", entries)
	}
}

func TestShellCommandSuccess(t *testing.T) {
	workDir := t.TempDir()

	result := New(config.ShellConfig{
		Enabled:   true,
		Directory: workDir,
		Commands: []config.ShellCommand{
			{Name: "echo-test", AllowFail: false, Command: "echo hello world"},
		},
	}).Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}
	if result.Message != "ran 1 command(s)" {
		t.Errorf("unexpected message: %q", result.Message)
	}

	logContent, err := os.ReadFile(filepath.Join(workDir, "echo-test.log"))
	if err != nil {
		t.Fatalf("log file not written: %v", err)
	}
	logText := string(logContent)
	if !strings.HasPrefix(logText, "Exit code: 0\n") {
		t.Errorf("log should start with 'Exit code: 0', got: %q", logText[:30])
	}
	if !strings.Contains(logText, "hello world") {
		t.Errorf("log missing stdout content: %s", logText)
	}
}

func TestShellAllowFailContinues(t *testing.T) {
	workDir := t.TempDir()

	result := New(config.ShellConfig{
		Enabled:   true,
		Directory: workDir,
		Commands: []config.ShellCommand{
			{Name: "fail-ok", AllowFail: true, Command: "exit 1"},
			{Name: "after-fail", AllowFail: false, Command: "echo ok"},
		},
	}).Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success when only allow_fail commands fail, got: %s", result.Error)
	}
	if result.Message != "ran 2 command(s)" {
		t.Errorf("unexpected message: %q", result.Message)
	}

	logContent, err := os.ReadFile(filepath.Join(workDir, "fail-ok.log"))
	if err != nil {
		t.Fatalf("log file not written: %v", err)
	}
	if !strings.HasPrefix(string(logContent), "Exit code: 1\n") {
		t.Errorf("expected exit code 1 in log, got: %s", string(logContent))
	}
}

func TestShellNoAllowFailStops(t *testing.T) {
	workDir := t.TempDir()

	result := New(config.ShellConfig{
		Enabled:   true,
		Directory: workDir,
		Commands: []config.ShellCommand{
			{Name: "hard-fail", AllowFail: false, Command: "exit 2"},
			{Name: "should-not-run", AllowFail: false, Command: "echo nope"},
		},
	}).Run(context.Background(), false)

	if result.Success {
		t.Fatal("expected failure when allow_fail is false and command exits non-zero")
	}
	if !strings.Contains(result.Error, "hard-fail") {
		t.Errorf("error should mention failing command, got: %s", result.Error)
	}

	if _, err := os.Stat(filepath.Join(workDir, "should-not-run.log")); err == nil {
		t.Error("second command must not run after allow_fail=false failure")
	}
}

func TestShellFirstBootRunsOnce(t *testing.T) {
	workDir := t.TempDir()

	moduleCfg := config.ShellConfig{
		Enabled:   true,
		Directory: workDir,
		Commands: []config.ShellCommand{
			{Name: "first-boot-cmd", AllowFail: false, FirstBoot: true, Command: "echo firstboot"},
		},
	}

	// First run: command executes.
	result := New(moduleCfg).Run(context.Background(), false)
	if !result.Success {
		t.Fatalf("first run failed: %s", result.Error)
	}
	if result.Message != "ran 1 command(s)" {
		t.Errorf("unexpected first-run message: %q", result.Message)
	}

	// Sentinel should exist after first run.
	sentinelPath := filepath.Join(workDir, "first-boot-cmd.firstboot")
	if _, err := os.Stat(sentinelPath); err != nil {
		t.Fatalf("firstboot sentinel not written: %v", err)
	}

	// Second run: command is skipped.
	result = New(moduleCfg).Run(context.Background(), false)
	if !result.Success {
		t.Fatalf("second run failed: %s", result.Error)
	}
	if result.Message != "ran 0 command(s)" {
		t.Errorf("expected 0 commands ran on second boot, got: %q", result.Message)
	}
}

func TestShellFirstBootSentinelWrittenOnFailure(t *testing.T) {
	workDir := t.TempDir()

	// allow_fail: true + firstboot: true — sentinel written even though command fails.
	result := New(config.ShellConfig{
		Enabled:   true,
		Directory: workDir,
		Commands: []config.ShellCommand{
			{Name: "fail-once", AllowFail: true, FirstBoot: true, Command: "exit 1"},
		},
	}).Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success (allow_fail=true), got: %s", result.Error)
	}

	sentinelPath := filepath.Join(workDir, "fail-once.firstboot")
	if _, err := os.Stat(sentinelPath); err != nil {
		t.Fatalf("sentinel should be written even on failure: %v", err)
	}
}

func TestShellPathInjected(t *testing.T) {
	workDir := t.TempDir()
	binDir := t.TempDir()

	// Place a script named "sentinel-bin" in binDir and confirm it's found via PATH.
	scriptPath := filepath.Join(binDir, "sentinel-bin")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\necho found-via-path"), 0750); err != nil {
		t.Fatal(err)
	}

	result := New(config.ShellConfig{
		Enabled:   true,
		Directory: workDir,
		Path:      binDir,
		Commands: []config.ShellCommand{
			{Name: "path-test", Command: "sentinel-bin"},
		},
	}).Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}

	logContent, err := os.ReadFile(filepath.Join(workDir, "path-test.log"))
	if err != nil {
		t.Fatalf("log not written: %v", err)
	}
	if !strings.Contains(string(logContent), "found-via-path") {
		t.Errorf("expected binary from injected PATH to run, log:\n%s", logContent)
	}
}

func TestShellLogFormat(t *testing.T) {
	workDir := t.TempDir()

	New(config.ShellConfig{
		Enabled:   true,
		Directory: workDir,
		Commands: []config.ShellCommand{
			{Name: "log-test", AllowFail: true, Command: "echo out; echo err >&2; exit 3"},
		},
	}).Run(context.Background(), false)

	logContent, err := os.ReadFile(filepath.Join(workDir, "log-test.log"))
	if err != nil {
		t.Fatalf("log not written: %v", err)
	}
	logText := string(logContent)

	if !strings.Contains(logText, "Exit code: 3") {
		t.Errorf("missing exit code in log: %s", logText)
	}
	if !strings.Contains(logText, "--- stdout ---") {
		t.Errorf("missing stdout section in log: %s", logText)
	}
	if !strings.Contains(logText, "--- stderr ---") {
		t.Errorf("missing stderr section in log: %s", logText)
	}
	if !strings.Contains(logText, "out\n") {
		t.Errorf("missing stdout content in log: %s", logText)
	}
	if !strings.Contains(logText, "err\n") {
		t.Errorf("missing stderr content in log: %s", logText)
	}
}
