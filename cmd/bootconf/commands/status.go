package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/offline-lab/bootconf/internal/config"
	"github.com/offline-lab/bootconf/internal/status"
)

var (
	statusSection string
	showFailed    bool
	showFull      bool
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of the last configuration run",
	Long: `Display the results of the most recent bootconf run.
By default shows a summary. Use flags to filter or show details.`,
	Run: func(_ *cobra.Command, _ []string) {
		basedir := "/data/config/bootconf"

		if _, err := os.Stat(configPath); err == nil {
			cfg, err := config.Load(configPath)
			if err == nil && cfg.Bootconf.Directory != "" {
				basedir = cfg.Bootconf.Directory
			}
		}

		runStatus, err := status.Read(basedir)

		if err != nil {
			fmt.Println("no status found")
			os.Exit(0)
		}

		if statusSection != "" {
			filtered := filterBySection(runStatus.Sections, statusSection)

			if len(filtered) == 0 {
				fmt.Printf("no status found for section %q\n", statusSection)
				os.Exit(0)
			}

			printSections(filtered, showFull)

			return
		}

		if showFailed {
			failed := filterFailed(runStatus.Sections)

			if len(failed) == 0 {
				fmt.Println("all sections passed")

				return
			}

			printSections(failed, showFull)

			return
		}

		printSummary(runStatus)

		if showFull {
			printSections(runStatus.Sections, true)
		}
	},
}

func filterBySection(sections []status.SectionStatus, name string) []status.SectionStatus {
	var filtered []status.SectionStatus

	for _, section := range sections {
		if section.Section == name {
			filtered = append(filtered, section)
		}
	}

	return filtered
}

func filterFailed(sections []status.SectionStatus) []status.SectionStatus {
	var failed []status.SectionStatus

	for _, section := range sections {
		if !section.Success {
			failed = append(failed, section)
		}
	}

	return failed
}

func printSummary(runStatus *status.RunStatus) {
	passCount := 0
	failCount := 0

	for _, section := range runStatus.Sections {
		if section.Success {
			passCount++
		} else {
			failCount++
		}
	}

	fmt.Printf("Timestamp: %s\n", runStatus.Timestamp.Format("2006-01-02 15:04:05 UTC"))
	fmt.Printf("Overall:   %s\n", boolStr(runStatus.Overall))
	fmt.Printf("Sections:  %d passed, %d failed\n", passCount, failCount)
}

func printSections(sections []status.SectionStatus, full bool) {
	for _, section := range sections {
		label := "OK"

		if !section.Success {
			label = "FAILED"
		}

		fmt.Printf("  %-12s %s\n", section.Section, label)

		if full {
			if section.Message != "" {
				fmt.Printf("    message:  %s\n", section.Message)
			}

			if section.Error != "" {
				fmt.Printf("    error:    %s\n", section.Error)
			}

			if section.Duration != "" {
				fmt.Printf("    duration: %s\n", section.Duration)
			}
		}
	}
}

func boolStr(success bool) string {
	if success {
		return "PASS"
	}

	return "FAIL"
}

func init() {
	statusCmd.Flags().StringVar(&statusSection, "section", "", "Show status for a specific section")
	statusCmd.Flags().BoolVar(&showFailed, "failed", false, "Only show failed sections")
	statusCmd.Flags().BoolVar(&showFull, "full", false, "Show all details including duration and messages")
}
