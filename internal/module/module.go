package module

import "context"

// Result holds the outcome of a single module execution.
type Result struct {
	// Section identifies which module produced this result.
	Section string

	// Success indicates whether the module completed without errors.
	Success bool

	// Message contains a human-readable summary of what was done.
	Message string

	// Error describes any failure that occurred, empty on success.
	Error string

	// Duration records how long the module took to execute.
	Duration string
}

// Module is the interface each configuration section must implement.
type Module interface {
	Name() string
	Run(ctx context.Context, dryRun bool) Result
}
