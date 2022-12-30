package build

var CurrentCommit string

// BuildVersion is the local build version, set by build system
const BuildVersion = "1.9.0"
const Version = "190"

func UserVersion() string {
	return BuildVersion + CurrentCommit
}
