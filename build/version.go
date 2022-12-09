package build

var CurrentCommit string

// BuildVersion is the local build version, set by build system
const BuildVersion = "1.8.1"
const Version = "181"

func UserVersion() string {
	return BuildVersion + CurrentCommit
}
