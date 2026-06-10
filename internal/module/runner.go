package module

import (
	"context"
	"sync"
	"time"

	"github.com/offline-lab/bootconf/internal/logging"
)

// Runner executes all modules concurrently and collects results in declaration order.
type Runner struct {
	modules []Module
}

// NewRunner creates a Runner for the given module list.
func NewRunner(modules []Module) *Runner {
	return &Runner{modules: modules}
}

// Run executes each module concurrently in its own goroutine. Results are
// returned in the same order as the module list regardless of completion order.
// If section is non-empty, only the matching module runs.
func (runner *Runner) Run(ctx context.Context, dryRun bool, section string) []Result {
	var active []Module
	for _, mod := range runner.modules {
		if section == "" || mod.Name() == section {
			active = append(active, mod)
		}
	}

	results := make([]Result, len(active))
	var wg sync.WaitGroup

	for index, mod := range active {
		wg.Add(1)
		go func(index int, currentModule Module) {
			defer wg.Done()
			start := time.Now()
			logging.Debug(currentModule.Name(), "starting")
			result := currentModule.Run(ctx, dryRun)
			result.Duration = time.Since(start).String()
			logging.Debug(currentModule.Name(), "done in %s success=%v", result.Duration, result.Success)
			results[index] = result
		}(index, mod)
	}

	wg.Wait()
	return results
}
