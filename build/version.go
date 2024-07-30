package build

var CurrentCommit string

// BuildVersion is the local build version, set by build system
const BuildVersion = "1.16.0"
const Version = "1160"

func UserVersion() string {
	return BuildVersion + CurrentCommit
}
