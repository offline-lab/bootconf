// Package shell executes configured shell commands at boot time. Each command
// runs via bash -c, with stdout, stderr, and exit code written to a log file
// in the configured directory. Commands with firstboot: true run only once —
// a sentinel file prevents re-execution on subsequent boots.
package shell

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/offline-lab/bootconf/internal/config"
	"github.com/offline-lab/bootconf/internal/logging"
	"github.com/offline-lab/bootconf/internal/module"
)

// ShellModule runs configured shell commands and logs their output.
type ShellModule struct {
	enabled   bool
	directory string
	path      string
	commands  []config.ShellCommand
}

// New creates a ShellModule from the given shell config.
func New(cfg config.ShellConfig) *ShellModule {
	return &ShellModule{
		enabled:   cfg.Enabled,
		directory: cfg.Directory,
		path:      cfg.Path,
		commands:  cfg.Commands,
	}
}

// Name returns the module identifier "shell".
func (shellModule *ShellModule) Name() string { return "shell" }

// Run executes each command in order. A command with allow_fail: false that
// exits non-zero stops execution and marks the module as failed.
func (shellModule *ShellModule) Run(ctx context.Context, dryRun bool, _ bool) module.Result {
	if !shellModule.enabled {
		return module.Result{Section: shellModule.Name(), Success: true, Message: "shell disabled"}
	}

	if !dryRun {
		if err := os.MkdirAll(shellModule.directory, 0750); err != nil {
			errMsg := fmt.Sprintf("failed to create shell directory %s: %v", shellModule.directory, err)
			logging.Error(shellModule.Name(), "%s", errMsg)

			return module.Result{Section: shellModule.Name(), Success: false, Error: errMsg}
		}
	}

	ran := 0
	for _, command := range shellModule.commands {
		skipped, err := shellModule.runCommand(ctx, command, dryRun)

		if skipped {
			continue
		}

		if !dryRun {
			ran++
		}

		if err != nil {
			errMsg := fmt.Sprintf("command %q: %v", command.Name, err)

			return module.Result{Section: shellModule.Name(), Success: false, Error: errMsg}
		}
	}

	if dryRun {
		return module.Result{Section: shellModule.Name(), Success: true, Message: "shell configured (dry-run)"}
	}

	return module.Result{Section: shellModule.Name(), Success: true, Message: fmt.Sprintf("ran %d command(s)", ran)}
}

// runCommand executes a single command. Returns (skipped=true, nil) when a
// firstboot command has already run. Returns (false, error) only when
// allow_fail is false and the command exits non-zero.
func (shellModule *ShellModule) runCommand(ctx context.Context, command config.ShellCommand, dryRun bool) (bool, error) {
	firstBootSentinel := filepath.Join(shellModule.directory, command.Name+".firstboot")

	if command.FirstBoot {
		if _, err := os.Stat(firstBootSentinel); err == nil {
			logging.Debug(shellModule.Name(), "skipping %q: already ran on first boot", command.Name)

			return true, nil
		}
	}

	if dryRun {
		logging.Info(shellModule.Name(), "would run command %q (dry-run)", command.Name)

		return false, nil
	}

	logging.Info(shellModule.Name(), "running command %q", command.Name)

	cmd := exec.CommandContext(ctx, "bash", "-c", command.Command)

	if shellModule.path != "" {
		current := os.Getenv("PATH")

		if current == "" {
			current = "/usr/sbin:/usr/bin:/sbin:/bin"
		}

		cmd.Env = append(os.Environ(), "PATH="+strings.Join([]string{shellModule.path, current}, ":"))
	}

	var stdoutBuf, stderrBuf bytes.Buffer

	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	runErr := cmd.Run()

	exitCode := 0

	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()

		} else {
			exitCode = -1
		}
	}

	logPath := filepath.Join(shellModule.directory, command.Name+".log")
	logContent := fmt.Sprintf("Exit code: %d\n--- stdout ---\n%s--- stderr ---\n%s",
		exitCode, stdoutBuf.String(), stderrBuf.String())

	if err := os.WriteFile(logPath, []byte(logContent), 0640); err != nil {
		logging.Warn(shellModule.Name(), "failed to write log %s: %v", logPath, err)
	}

	// Write firstboot sentinel after the command runs, regardless of exit code,
	// so a failed firstboot command does not loop forever on every boot.

	if command.FirstBoot {
		if err := os.WriteFile(firstBootSentinel, []byte{}, 0640); err != nil {
			logging.Warn(shellModule.Name(), "failed to write firstboot sentinel %s: %v", firstBootSentinel, err)
		}
	}

	if runErr != nil {
		logging.Error(shellModule.Name(), "command %q exited with code %d", command.Name, exitCode)

		if !command.AllowFail {
			return false, fmt.Errorf("exited with code %d", exitCode)
		}

		logging.Info(shellModule.Name(), "command %q failed but allow_fail is true, continuing", command.Name)

	} else {
		logging.Info(shellModule.Name(), "command %q completed (exit 0)", command.Name)
	}

	return false, nil
}
