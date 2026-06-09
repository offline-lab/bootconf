package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/offline-lab/bootconf/internal/config"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check runtime state against configuration",
	Long: `Verify that the currently running system matches the bootconf
configuration. Checks service status, user existence, and process state.`,
	Run: func(cmd *cobra.Command, args []string) {
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

		allHealthy := true

		// SSH
		if cfg.SSH.Enabled {
			daemonName := "dropbear"
			if cfg.SSH.Daemon == "openssh" {
				daemonName = "ssh"
			}
			if err := checkActive(daemonName); err != nil {
				fmt.Printf("  %-12s FAILED: %v\n", "ssh", err)
				allHealthy = false
			} else {
				fmt.Printf("  %-12s OK\n", "ssh")
			}
		}

		// Wifi
		if cfg.Wifi.Enabled {
			if err := checkProcess("wpa_supplicant"); err != nil {
				fmt.Printf("  %-12s FAILED: %v\n", "wifi", err)
				allHealthy = false
			} else {
				fmt.Printf("  %-12s OK\n", "wifi")
			}
		}

		// Services
		for _, svc := range cfg.Services.Services {
			if !svc.Enabled {
				continue
			}
			if err := checkActive(svc.Name); err != nil {
				fmt.Printf("  %-12s FAILED: %v\n", "service/"+svc.Name, err)
				allHealthy = false
			} else {
				fmt.Printf("  %-12s OK\n", "service/"+svc.Name)
			}
		}

		// Users
		for _, user := range cfg.Users.Users {
			if !user.Enabled {
				continue
			}
			if err := checkUserExists(user.Name); err != nil {
				fmt.Printf("  %-12s FAILED: %v\n", "user/"+user.Name, err)
				allHealthy = false
			} else {
				fmt.Printf("  %-12s OK\n", "user/"+user.Name)
			}
		}

		if !allHealthy {
			os.Exit(1)
		}
	},
}

func checkActive(service string) error {
	cmd := exec.Command("systemctl", "is-active", service)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("service %s is not active", service)
	}
	return nil
}

func checkProcess(name string) error {
	cmd := exec.Command("pgrep", "-x", name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("process %s is not running", name)
	}
	return nil
}

func checkUserExists(name string) error {
	cmd := exec.Command("id", name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("user %s does not exist", name)
	}
	return nil
}
