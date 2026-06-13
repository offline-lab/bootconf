// Package run provides a thin wrapper around exec for running system commands.
// All module code should use this instead of calling exec directly so that
// command invocations are consistently formatted in errors.
package run

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Command runs the named binary with the given arguments under ctx.
// The returned error includes the full command line for quick diagnosis.
func Command(ctx context.Context, name string, args ...string) error {

	if err := exec.CommandContext(ctx, name, args...).Run(); err != nil {
		cmdLine := strings.Join(append([]string{name}, args...), " ")

		return fmt.Errorf("%s: %w", cmdLine, err)
	}

	return nil
}
