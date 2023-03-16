package build

var CurrentCommit string

// BuildVersion is the local build version, set by build system
const BuildVersion = "1.10.1"
const Version = "1100"

func UserVersion() string {
	return BuildVersion + CurrentCommit
}
