package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/coreos/go-systemd/v22/daemon"
	"github.com/spf13/cobra"

	"github.com/offline-lab/bootconf/internal/config"
	"github.com/offline-lab/bootconf/internal/logging"
	"github.com/offline-lab/bootconf/internal/module"
	"github.com/offline-lab/bootconf/internal/output"
	"github.com/offline-lab/bootconf/internal/registry"
	"github.com/offline-lab/bootconf/internal/status"
)

var (
	dryRun     bool
	runSection string
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Apply boot configuration",
	Long: `Read the bootconf configuration file and apply all configured
sections. Creates status files, sentinel files, host keys, user accounts,
and service configurations as needed.`,
	Run: runBootconf,
}

func runBootconf(_ *cobra.Command, _ []string) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		os.Exit(0)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if verbose {
		logging.SetLevel(logging.DEBUG)
	}

	registry.ApplyDefaults(cfg)

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "validation error: %v\n", err)
		os.Exit(1)
	}

	if err := registry.Validate(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "validation error: %v\n", err)
		os.Exit(1)
	}

	if !cfg.Bootconf.Enabled {
		fmt.Println("bootconf disabled, skipping")
		os.Exit(0)
	}

	_, _ = daemon.SdNotify(false, "STATUS=Applying boot configuration")

	modules := registry.Build(cfg)

	ctx := context.Background()
	start := time.Now()
	results := module.NewRunner(modules).Run(ctx, dryRun, runSection)
	totalDuration := time.Since(start)

	overall := writeRunStatus(cfg.Bootconf.Directory, results)
	printResults(results, dryRun, totalDuration)
	if !overall {
		os.Exit(1)
	}

	_, _ = daemon.SdNotify(false, daemon.SdNotifyReady)
}

func writeRunStatus(statusDir string, results []module.Result) bool {
	overall := true
	for _, result := range results {
		if !result.Success {
			overall = false
		}
	}
	if err := status.Write(statusDir, &status.RunStatus{
		Timestamp: time.Now().UTC(),
		Overall:   overall,
		Sections:  results,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to write status: %v\n", err)
	}
	return overall
}

func printResults(results []module.Result, dryRun bool, totalDuration time.Duration) {
	if dryRun {
		fmt.Println("[dry-run] No changes were applied.")
		fmt.Println()
	}

	statusLabel := "OK"
	if dryRun {
		statusLabel = "DRY-RUN"
	}

	table := output.NewTable("Section", "Status", "Detail")
	for _, result := range results {
		sectionLabel := statusLabel
		if !result.Success {
			sectionLabel = "FAIL"
		}

		detail := result.Message
		if !result.Success {
			detail = result.Error
		}

		table.AddRow(result.Section, sectionLabel, detail)
	}
	table.Render()

	fmt.Printf("\n%d section(s) completed in %s\n", len(results), totalDuration.Round(time.Microsecond))
}

func init() {
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")
	runCmd.Flags().StringVar(&runSection, "section", "", "Only run a specific section")
}
