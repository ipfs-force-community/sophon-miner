package build

var CurrentCommit string

// BuildVersion is the local build version, set by build system
const (
	BuildVersion = "1.21.0"
	Version      = "1210"
)

func UserVersion() string {
	return BuildVersion + CurrentCommit
}
