package impl

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/filecoin-project/venus-miner/miner"
	"github.com/filecoin-project/venus-miner/node/impl/common"
	"github.com/filecoin-project/venus-miner/types"
)

type MinerAPI struct {
	common.CommonAPI

	miner.MiningAPI
}

func (m *MinerAPI) UpdateAddress(ctx context.Context, skip int64, limit int64) ([]types.MinerInfo, error) {
	return m.MiningAPI.UpdateAddress(ctx, skip, limit)
}

func (m *MinerAPI) ListAddress(ctx context.Context) ([]types.MinerInfo, error) {
	return m.MiningAPI.ListAddress(ctx)
}

func (m *MinerAPI) StatesForMining(ctx context.Context, addrs []address.Address) ([]types.MinerState, error) {
	return m.MiningAPI.StatesForMining(ctx, addrs)
}

func (m *MinerAPI) CountWinners(ctx context.Context, addrs []address.Address, start abi.ChainEpoch, end abi.ChainEpoch) ([]types.CountWinners, error) {
	return m.MiningAPI.CountWinners(ctx, addrs, start, end)
}

func (m *MinerAPI) Start(ctx context.Context, addrs []address.Address) error {
	return m.MiningAPI.ManualStart(ctx, addrs)
}

func (m *MinerAPI) Stop(ctx context.Context, addrs []address.Address) error {
	return m.MiningAPI.ManualStop(ctx, addrs)
}
