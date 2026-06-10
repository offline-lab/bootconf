// Package commands defines the Cobra CLI commands for bootconf.
package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/offline-lab/bootconf/internal/config"
	"github.com/offline-lab/bootconf/internal/output"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check runtime state against configuration",
	Long: `Verify that the currently running system matches the bootconf
configuration. Checks service status, user existence, and process state.`,
	Run: runCheck,
}

// healthCheck describes a single runtime verification step.
type healthCheck struct {
	label string
	check func() error
}

func runCheck(_ *cobra.Command, _ []string) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "error: no config file found")
		os.Exit(1)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "validation error: %v\n", err)
		os.Exit(1)
	}

	checks := buildHealthChecks(cfg)
	table := output.NewTable("Section", "Status", "Detail")

	passCount := 0
	failCount := 0

	for _, entry := range checks {
		if err := entry.check(); err != nil {
			table.AddRow(entry.label, "FAIL", err.Error())
			failCount++
		} else {
			table.AddRow(entry.label, "PASS", "")
			passCount++
		}
	}

	table.Render()
	fmt.Printf("\n%d check(s): %d passed, %d failed\n", passCount+failCount, passCount, failCount)

	if failCount > 0 {
		os.Exit(1)
	}
}

// buildHealthChecks returns the ordered list of runtime checks derived from
// the configuration. Only enabled sections produce checks.
func buildHealthChecks(cfg *config.Config) []healthCheck {
	var checks []healthCheck

	if cfg.SSH.Enabled {
		sshDaemon := "dropbear"
		if cfg.SSH.Daemon == "openssh" {
			sshDaemon = "ssh"
		}
		checks = append(checks, healthCheck{
			label: "ssh",
			check: func() error { return checkActiveService(sshDaemon) },
		})
	}

	if cfg.Wifi.Enabled {
		checks = append(checks, healthCheck{
			label: "wifi",
			check: func() error { return checkRunningProcess("wpa_supplicant") },
		})
	}

	for _, service := range cfg.Services.Services {
		if service.Enabled {
			serviceName := service.Name
			checks = append(checks, healthCheck{
				label: "service/" + serviceName,
				check: func() error { return checkActiveService(serviceName) },
			})
		}
	}

	for _, user := range cfg.Users.Users {
		if user.Enabled {
			userName := user.Name
			checks = append(checks, healthCheck{
				label: "user/" + userName,
				check: func() error { return checkUserExists(userName) },
			})
		}
	}

	return checks
}

// checkActiveService verifies a systemd service is in active state.
func checkActiveService(service string) error {
	if err := exec.Command("systemctl", "is-active", service).Run(); err != nil {
		return fmt.Errorf("service %s is not active", service)
	}
	return nil
}

// checkRunningProcess verifies a process is currently running by exact name match.
func checkRunningProcess(name string) error {
	if err := exec.Command("pgrep", "-x", name).Run(); err != nil {
		return fmt.Errorf("process %s is not running", name)
	}
	return nil
}

// checkUserExists verifies a system user account exists.
func checkUserExists(name string) error {
	if err := exec.Command("id", name).Run(); err != nil {
		return fmt.Errorf("user %s does not exist", name)
	}
	return nil
}
