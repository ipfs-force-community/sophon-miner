package build

var CurrentCommit string

// BuildVersion is the local build version, set by build system
const BuildVersion = "1.13.1"
const Version = "1131"

func UserVersion() string {
	return BuildVersion + CurrentCommit
}
