package v0api

import (
	"github.com/filecoin-project/venus-miner/api"
)

type Common = api.Common
type CommonStruct = api.CommonStruct

type MinerAPI = api.MinerAPI

func PermissionedMinerAPI(m MinerAPI) MinerAPI {
	return api.PermissionedMinerAPI(m)
}

