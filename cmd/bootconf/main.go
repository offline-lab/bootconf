// Package main is the entry point for the bootconf binary.
package main

import (
	"os"

	"github.com/offline-lab/bootconf/cmd/bootconf/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
