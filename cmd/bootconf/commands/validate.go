package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/offline-lab/bootconf/internal/config"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the configuration file",
	Long: `Parse and validate the bootconf configuration file without
making any changes to the system. Works fully offline.`,
	Run: validateConfig,
}

// validateConfig parses and validates the config file without making any system changes. Used for offline verification (e.g. in CI).
func validateConfig(_ *cobra.Command, _ []string) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Println("no config file found")
		os.Exit(0)
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

	fmt.Println("configuration is valid")
}
