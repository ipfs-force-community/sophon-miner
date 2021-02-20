package impl

import (
	"context"

	"github.com/filecoin-project/go-address"

	"github.com/filecoin-project/venus-miner/miner"
	"github.com/filecoin-project/venus-miner/node/config"
	"github.com/filecoin-project/venus-miner/node/impl/common"
	"github.com/filecoin-project/venus-miner/node/modules/posterkeymgr"
)

type MinerAPI struct {
	common.CommonAPI

	ActorMgr posterkeymgr.IActorMgr
	Miner    miner.BlockMinerApi `optional:"true"`
}

func (m *MinerAPI) AddAddress(posterAddr config.MinerInfo) error {
	return m.Miner.AddAddress(posterAddr)
}

func (m *MinerAPI) RemoveAddress(addr address.Address) error {
	return m.Miner.RemoveAddress(addr)
}

func (m *MinerAPI) ListAddress() ([]config.MinerInfo, error) {
	return m.ActorMgr.ListKey()
}

func (m *MinerAPI) SetDefault(ctx context.Context, addr address.Address) error {
	return m.ActorMgr.SetDefault(addr)
}
