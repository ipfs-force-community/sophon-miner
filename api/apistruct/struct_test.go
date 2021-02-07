package apistruct

import "testing"

func TestPermTags(t *testing.T) {
	_ = PermissionedFullAPI(&FullNodeStruct{})
	_ = PermissionedMinerAPI(&MinerStruct{})
}
