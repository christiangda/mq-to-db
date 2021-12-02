package version

import (
	"fmt"
	"runtime"
)

// Populated at build-time.
// go build \
// -ldflags "-X github.com/mq-to-db/internal/verion.Version=$(git rev-parse --abbrev-ref HEAD) \
//           -X github.com/mq-to-db/internal/verion.Revision=$(git rev-parse --short HEAD) \
//           -X github.com/mq-to-db/internal/verion.Branch=$(git rev-parse --abbrev-ref HEAD) \
//           -X github.com/mq-to-db/internal/verion.BuildUser=$(git config --get user.name) \
//           -X github.com/mq-to-db/internal/verion.BuildDate=$(date +'%Y-%m-%dT%H:%M:%S')"
var (
	Version   string
	Revision  string
	Branch    string
	BuildUser string
	BuildDate string
	GoVersion = runtime.Version()
)

// GetVersion returns the version string.
func GetVersion() string {
	if Version == "" {
		Version = "0.0.0"
	}

	return Version
}

// GetVersionInfo returns the version string.
func GetVersionInfo() string {
	if Version == "" {
		Version = "0.0.0"
	}
	if Revision == "" {
		Revision = "0"
	}
	if Branch == "" {
		Branch = "unknown"
	}

	return fmt.Sprintf("(version=%s, revision=%s, branch=%s)",
		Version,
		Revision,
		Branch,
	)
}

// GetVersionInfoExtended returns the version string.
func GetVersionInfoExtended() string {
	if Version == "" {
		Version = "0.0.0"
	}
	if Revision == "" {
		Revision = "0"
	}
	if Branch == "" {
		Branch = "unknown"
	}
	if BuildUser == "" {
		BuildUser = "unknown"
	}
	if BuildDate == "" {
		BuildDate = "unknown"
	}

	return fmt.Sprintf("(version=%s, revision=%s, branch=%s, go=%s, user=%s, date=%s)",
		Version,
		Revision,
		Branch,
		GoVersion,
		BuildUser,
		BuildDate)
}
