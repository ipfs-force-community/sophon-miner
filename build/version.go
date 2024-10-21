package build

var CurrentCommit string

// BuildVersion is the local build version, set by build system
const BuildVersion = "1.17.0-rc1"
const Version = "1160"

func UserVersion() string {
	return BuildVersion + CurrentCommit
}
