package skillhubd

import "fmt"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func VersionString() string {
	return fmt.Sprintf("skill-hubd version %s (commit: %s, built: %s)", version, commit, date)
}
