// Package commands defines the Cobra CLI commands for bootconf.
package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	configPath string
	verbose    bool
)

var rootCmd = &cobra.Command{
	Use:   "bootconf",
	Short: "Declarative boot configuration for Linux",
	Long: `Bootconf reads a YAML configuration file and applies system settings
during early boot, before other services start. It manages hostname, timezone,
SSH host keys, WiFi credentials, user accounts, service sentinel files, and
arbitrary file provisioning — on every boot, from a single config file.`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("bootconf %s\n", Version)
		fmt.Printf("  Commit:    %s\n", Commit)
		fmt.Printf("  Built:     %s\n", BuildTime)
	},
}

// Execute runs the root Cobra command and returns any error.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "/boot/firmware/bootconf.yaml", "Path to configuration file")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(checkCmd)
}
