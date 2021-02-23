package impl

import (
	"github.com/filecoin-project/go-address"

	"github.com/filecoin-project/venus-miner/miner"
	"github.com/filecoin-project/venus-miner/node/impl/common"
	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
	"github.com/filecoin-project/venus-miner/node/modules/minermanage"
)

type MinerAPI struct {
	common.CommonAPI

	MinerManager minermanage.MinerManageAPI
	miner.MiningAPI
}

func (m *MinerAPI) AddAddress(minerInfo dtypes.MinerInfo) error {

	return m.MiningAPI.AddAddress(minerInfo)
}

func (m *MinerAPI) RemoveAddress(addr address.Address) error {
	return m.MiningAPI.RemoveAddress(addr)
}

func (m *MinerAPI) ListAddress() ([]dtypes.MinerInfo, error) {
	return m.MinerManager.List()
}

func (m *MinerAPI) SetDefault(addr address.Address) error {
	return m.MinerManager.SetDefault(addr)
}

func (m *MinerAPI) Default() (address.Address, error) {
	return m.MinerManager.Default()
}
