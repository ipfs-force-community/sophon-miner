package build

var CurrentCommit string

// BuildVersion is the local build version, set by build system
const BuildVersion = "1.15.0-rc1"
const Version = "1150"

func UserVersion() string {
	return BuildVersion + CurrentCommit
}
