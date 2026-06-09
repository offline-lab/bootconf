package module

import (
	"context"
	"time"

	"github.com/offline-lab/bootconf/internal/logging"
)

type Runner struct {
	modules []Module
	section string
}

func NewRunner(modules []Module) *Runner {
	return &Runner{
		modules: modules,
	}
}

func (runner *Runner) SetSection(name string) {
	runner.section = name
}

func (runner *Runner) Run(ctx context.Context, dryRun bool) []Result {
	var results []Result

	for _, mod := range runner.modules {
		if runner.section != "" && mod.Name() != runner.section {
			continue
		}

		start := time.Now()
		logging.Info(mod.Name(), "starting section")
		result := mod.Run(ctx, dryRun)
		result.Duration = time.Since(start).String()
		logging.Info(mod.Name(), "section completed in %s: success=%v", result.Duration, result.Success)
		results = append(results, result)
	}

	return results
}
