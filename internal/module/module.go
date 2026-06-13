// Package module defines the interface and runner for bootconf configuration
// modules. Each module (system, ssh, wifi, etc.) implements the Module
// interface and is executed sequentially by the Runner.
package module

import "context"

// Result holds the outcome of a single module execution.
type Result struct {
	Section  string `json:"section"`
	Success  bool   `json:"success"`
	Message  string `json:"message,omitempty"`
	Error    string `json:"error,omitempty"`
	Duration string `json:"duration,omitempty"`
}

// Module is the interface each configuration section must implement.
type Module interface {
	Name() string
	Run(ctx context.Context, dryRun bool, apply bool) Result
}
