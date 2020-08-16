package version

import (
	"fmt"
	"runtime"
)

// Populated at build-time.
// go build -ldflags \
// “-X github.com/mq-to-db/internal/verion.Version=$(git rev-parse --abbrev-ref HEAD) \
// -X github.com/mq-to-db/internal/verion.Revision=$(git rev-parse --short HEAD) \
// -X github.com/mq-to-db/internal/verion.Branch=$(git rev-parse --abbrev-ref HEAD) \
// -X github.com/mq-to-db/internal/verion.BuildUser=$(git config --get user.name) \
// -X github.com/mq-to-db/internal/verion.BuildDate=$(date +”%Y-%m-%dT%H:%M:%S”)”
var (
	Version   string
	Revision  string
	Branch    string
	BuildUser string
	BuildDate string
	GoVersion = runtime.Version()
)

// GetVersionInfo returns application version
func GetVersionInfo() string {
	return fmt.Sprintf("(version=%s, branch=%s, revision=%s)",
		Version,
		Branch,
		Revision)
}

// GetVersionInfoExtended returns application version, branch and revision information.
func GetVersionInfoExtended() string {
	return fmt.Sprintf("(version=%s, branch=%s, revision=%s, go=%s, user=%s, date=%s)",
		Version,
		Branch,
		Revision,
		GoVersion,
		BuildUser,
		BuildDate)
}
