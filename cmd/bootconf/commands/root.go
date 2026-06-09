// Package commands defines the Cobra CLI commands for bootconf.
package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/offline-lab/bootconf/internal/version"
)

var (
	configPath string
	verbose    bool
)

var rootCmd = &cobra.Command{
	Use:   "bootconf",
	Short: "Configure a readonly OS at boot time",
	Long: `Bootconf reads a YAML configuration file and prepares a readonly Linux
system at boot time, before other services start. It applies network settings,
hostname, SSH keys, and other system configuration from a single config file.`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("bootconf %s\n", version.Version)
		fmt.Printf("  Commit:    %s\n", version.Commit)
		fmt.Printf("  Built:     %s\n", version.BuildTime)
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
