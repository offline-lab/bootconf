package module

import (
	"context"
	"time"

	"github.com/offline-lab/bootconf/internal/logging"
)

// Runner executes modules sequentially and collects their results.
type Runner struct {
	modules []Module
}

// NewRunner creates a Runner for the given module list.
func NewRunner(modules []Module) *Runner {
	return &Runner{modules: modules}
}

// Run executes each module in order, one at a time. Results are returned in
// the same order as the module list. If section is non-empty, only the
// matching module runs.
func (runner *Runner) Run(ctx context.Context, dryRun bool, apply bool, section string) []Result {
	var active []Module

	for _, mod := range runner.modules {
		if section == "" || mod.Name() == section {
			active = append(active, mod)
		}
	}

	results := make([]Result, len(active))

	for index, mod := range active {
		start := time.Now()
		logging.Debug(mod.Name(), "starting")
		result := mod.Run(ctx, dryRun, apply)
		result.Duration = time.Since(start).String()
		logging.Debug(mod.Name(), "done in %s success=%v", result.Duration, result.Success)
		results[index] = result
	}

	return results
}
