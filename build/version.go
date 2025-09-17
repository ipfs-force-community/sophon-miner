package build

var CurrentCommit string

// BuildVersion is the local build version, set by build system
const BuildVersion = "1.19.0"
const Version = "1190"

func UserVersion() string {
	return BuildVersion + CurrentCommit
}
