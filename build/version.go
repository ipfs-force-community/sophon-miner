package build

var CurrentCommit string

// BuildVersion is the local build version, set by build system
const BuildVersion = "1.17.0"
const Version = "1170"

func UserVersion() string {
	return BuildVersion + CurrentCommit
}
