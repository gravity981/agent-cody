package buildinfo

import "os"

var (
	Version = "0.0.0+unknown"
	Commit  = "unknown"
)

func Current() (string, string) {
	version, commit := Version, Commit
	if value := os.Getenv("CODY_VERSION"); value != "" {
		version = value
	}
	if value := os.Getenv("CODY_COMMIT"); value != "" {
		commit = value
	}
	return version, commit
}
