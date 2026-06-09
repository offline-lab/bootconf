package module

import (
	"context"
	"time"

	"github.com/offline-lab/bootconf/internal/logging"
)

// Runner executes a list of modules sequentially and collects results.
type Runner struct {
	modules []Module
}

// NewRunner creates a Runner for the given module list.
func NewRunner(modules []Module) *Runner {
	return &Runner{modules: modules}
}

// Run executes each module sequentially, filtering by section name if
// specified. It collects results with timing information and returns them
// all — callers decide how to handle failures.
func (r *Runner) Run(ctx context.Context, dryRun bool, section string) []Result {
	var results []Result

	for _, mod := range r.modules {
		if section != "" && mod.Name() != section {
			continue
		}

		start := time.Now()
		logging.Debug(mod.Name(), "starting section")
		result := mod.Run(ctx, dryRun)
		result.Duration = time.Since(start).String()
		logging.Debug(mod.Name(), "section completed in %s: success=%v", result.Duration, result.Success)
		results = append(results, result)
	}

	return results
}
