package build

var CurrentCommit string

// BuildVersion is the local build version, set by build system
const BuildVersion = "1.12.0-rc1"
const Version = "1120"

func UserVersion() string {
	return BuildVersion + CurrentCommit
}
