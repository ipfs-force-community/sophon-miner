package build

var CurrentCommit string

// BuildVersion is the local build version, set by build system
const BuildVersion = "1.14.0"
const Version = "1140"

func UserVersion() string {
	return BuildVersion + CurrentCommit
}
