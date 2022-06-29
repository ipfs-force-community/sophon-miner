package build

var CurrentCommit string

// BuildVersion is the local build version, set by build system
const BuildVersion = "1.7.0-rc1"

func UserVersion() string {
	return BuildVersion + CurrentCommit
}
