package version

import (
	"fmt"
	"runtime"
)

// Populated at build-time.
// go build -ldflags \
// “-X github.com/mq-to-db/internal/verion.Version=$(git rev-parse --abbrev-ref HEAD) \
// -X github.com/mq-to-db/internal/verion.Revision=$(git log –pretty=format:’%h’ -n 1) \
// -X github.com/mq-to-db/internal/verion.Branch=$(git log –pretty=format:’%h’ -n 1) \
// -X github.com/mq-to-db/internal/verion.BuildUser=$(git log –pretty=format:’%h’ -n 1) \
// -X github.com/mq-to-db/internal/verion.BuildDate=$(date +”%Y-%m-%dT%H:%M:%S”)”
var (
	Version   string
	Revision  string
	Branch    string
	BuildUser string
	BuildDate string
	GoVersion = runtime.Version()
)

// GetVersion returns application version
func GetVersion() string {
	return Version
}

// GetVerionExtended returns application version, branch and revision information.
func GetVerionExtended() string {
	return fmt.Sprintf("(version=%s, branch=%s, revision=%s, go=%s, user=%s, date=%s)",
		Version,
		Branch,
		Revision,
		GoVersion,
		BuildUser,
		BuildDate)
}
