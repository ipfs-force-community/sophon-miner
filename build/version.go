package build

var CurrentCommit string

// BuildVersion is the local build version, set by build system
const BuildVersion = "1.18.0-rc2"
const Version = "1180"

func UserVersion() string {
	return BuildVersion + CurrentCommit
}
