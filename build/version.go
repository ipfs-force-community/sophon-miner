package build

var CurrentCommit string

// BuildVersion is the local build version, set by build system
const BuildVersion = "1.8.0-rc1"
const Version = "180"

func UserVersion() string {
	return BuildVersion + CurrentCommit
}
