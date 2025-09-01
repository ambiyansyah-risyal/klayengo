package klayengo

import (
	"fmt"
	"runtime"
)

// Version information - these will be set at build time
var (
	// Version is the current version of Klayengo
	Version = "1.0.0"

	// GitCommit is the git commit hash
	GitCommit = "unknown"

	// BuildDate is the date when the binary was built
	BuildDate = "unknown"

	// GoVersion is the Go version used to build
	GoVersion = runtime.Version()
)

// GetVersion returns formatted version information
func GetVersion() string {
	return fmt.Sprintf("Klayengo v%s (commit: %s, built: %s, go: %s)",
		Version, GitCommit, BuildDate, GoVersion)
}

// GetVersionInfo returns detailed version information as a map
func GetVersionInfo() map[string]string {
	return map[string]string{
		"version":    Version,
		"commit":     GitCommit,
		"build_date": BuildDate,
		"go_version": GoVersion,
	}
}
