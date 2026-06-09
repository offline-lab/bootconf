// Package version holds build-time metadata injected via LDFLAGS.
package version

// Version, Commit, and BuildTime are set at build time via -ldflags.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)
