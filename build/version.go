package build

var CurrentCommit string

// BuildVersion is the local build version, set by build system
const (
	BuildVersion = "1.20.0-rc1"
	Version      = "1200"
)

func UserVersion() string {
	return BuildVersion + CurrentCommit
}
