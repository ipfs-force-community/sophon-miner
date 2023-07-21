package build

var CurrentCommit string

// BuildVersion is the local build version, set by build system
const BuildVersion = "1.12.1-rc1"
const Version = "1121"

func UserVersion() string {
	return BuildVersion + CurrentCommit
}
