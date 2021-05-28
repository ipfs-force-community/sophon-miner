package impl

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"

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

func (m *MinerAPI) UpdateAddress(minerInfo dtypes.MinerInfo) error {

	return m.MiningAPI.UpdateAddress(minerInfo)
}

func (m *MinerAPI) RemoveAddress(addr address.Address) error {
	return m.MiningAPI.RemoveAddress(addr)
}

func (m *MinerAPI) ListAddress() ([]dtypes.MinerInfo, error) {
	return m.MiningAPI.ListAddress()
}

func (m *MinerAPI) StatesForMining(addrs []address.Address) ([]dtypes.MinerState, error) {
	return m.MiningAPI.StatesForMining(addrs)
}

func (m *MinerAPI) CountWinners(addrs []address.Address, start abi.ChainEpoch, end abi.ChainEpoch) ([]dtypes.CountWinners, error) {
	return m.MiningAPI.CountWinners(addrs, start, end)
}

func (m *MinerAPI) Start(ctx context.Context, addr address.Address) error {
	return m.MiningAPI.ManualStart(ctx, addr)
}

func (m *MinerAPI) Stop(ctx context.Context, addr address.Address) error {
	return m.MiningAPI.ManualStop(ctx, addr)
}
