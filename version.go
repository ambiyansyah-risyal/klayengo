package klayengo

import (
	"fmt"
	"runtime"
)

var (
	Version = "v1.1.0"

	GitCommit = "unknown"

	BuildDate = "unknown"

	GoVersion = runtime.Version()
)

func GetVersion() string {
	return fmt.Sprintf("Klayengo v%s (commit: %s, built: %s, go: %s)",
		Version, GitCommit, BuildDate, GoVersion)
}

func GetVersionInfo() map[string]string {
	return map[string]string{
		"version":    Version,
		"commit":     GitCommit,
		"build_date": BuildDate,
		"go_version": GoVersion,
	}
}
