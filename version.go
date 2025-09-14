package klayengo

import (
	"fmt"
	"runtime"
)

var (
	// Version is the library semantic version (injected at build time optionally).
	Version = "v1.1.0"
	// GitCommit is the git SHA (inject via -ldflags at build time).
	GitCommit = "unknown"
	// BuildDate is the build timestamp (inject via -ldflags).
	BuildDate = "unknown"
	// GoVersion records the Go toolchain version used.
	GoVersion = runtime.Version()
)

// GetVersion returns a human-readable version string.
func GetVersion() string {
	return fmt.Sprintf("Klayengo v%s (commit: %s, built: %s, go: %s)",
		Version, GitCommit, BuildDate, GoVersion)
}

// GetVersionInfo returns version metadata as a map for logging / metrics.
func GetVersionInfo() map[string]string {
	return map[string]string{
		"version":    Version,
		"commit":     GitCommit,
		"build_date": BuildDate,
		"go_version": GoVersion,
	}
}
