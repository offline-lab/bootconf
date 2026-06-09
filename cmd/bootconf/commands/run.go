package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/offline-lab/bootconf/internal/config"
	"github.com/offline-lab/bootconf/internal/logging"
	"github.com/offline-lab/bootconf/internal/module"
	"github.com/offline-lab/bootconf/internal/module/files"
	"github.com/offline-lab/bootconf/internal/module/services"
	"github.com/offline-lab/bootconf/internal/module/ssh"
	"github.com/offline-lab/bootconf/internal/module/system"
	"github.com/offline-lab/bootconf/internal/module/users"
	"github.com/offline-lab/bootconf/internal/module/wifi"
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

func runBootconf(cmd *cobra.Command, args []string) {
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

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "validation error: %v\n", err)
		os.Exit(1)
	}

	if !cfg.Bootconf.Enabled {
		fmt.Println("bootconf disabled, skipping")
		os.Exit(0)
	}

	modules := []module.Module{
		system.NewSystemModule(cfg.System, cfg.Bootconf.Directory),
		ssh.NewSSHModule(cfg.SSH, cfg.Services.Directory),
		wifi.NewWifiModule(cfg.Wifi, cfg.Services.Directory),
		services.NewServicesModule(cfg.Services),
		users.NewUsersModule(cfg.Users, 2000),
		files.NewFilesModule(cfg.Files),
	}

	runner := module.NewRunner(modules)
	if runSection != "" {
		runner.SetSection(runSection)
	}

	ctx := context.Background()
	results := runner.Run(ctx, dryRun)

	statusDir := filepath.Join(cfg.Bootconf.Directory, ".bootconf")
	writeRunStatus(statusDir, results)

	printResults(results)

	overall := true
	for _, section := range results {
		if !section.Success {
			overall = false
			break
		}
	}

	if !overall {
		os.Exit(1)
	}
}

func writeRunStatus(statusDir string, results []module.Result) {
	overall := true
	sections := make([]status.SectionStatus, len(results))
	for idx, section := range results {
		if !section.Success {
			overall = false
		}
		sections[idx] = status.SectionStatus{
			Section:  section.Section,
			Success:  section.Success,
			Message:  section.Message,
			Error:    section.Error,
			Duration: section.Duration,
		}
	}

	runStatus := &status.RunStatus{
		Timestamp: time.Now().UTC(),
		Overall:   overall,
		Sections:  sections,
	}

	if err := status.Write(statusDir, runStatus); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to write status: %v\n", err)
	}
}

func printResults(results []module.Result) {
	for _, section := range results {
		if section.Success {
			fmt.Printf("  %-12s OK\n", section.Section)
		} else {
			fmt.Printf("  %-12s FAILED: %s\n", section.Section, section.Error)
		}
	}
}

func init() {
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")
	runCmd.Flags().StringVar(&runSection, "section", "", "Only run a specific section")
}
