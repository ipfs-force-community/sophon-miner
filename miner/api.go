package miner

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/filecoin-project/go-address"

	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
)

type MiningAPI interface {
	IMinerMining
	IMinerManager
}

type IMinerMining interface {
	Start(context.Context) error
	Stop(context.Context) error
	ManualStart(context.Context, address.Address) error
	ManualStop(context.Context, address.Address) error
}

type IMinerManager interface {
	UpdateAddress(context.Context, int64, int64) ([]dtypes.MinerInfo, error)
	ListAddress(context.Context) ([]dtypes.MinerInfo, error)
	StatesForMining(context.Context, []address.Address) ([]dtypes.MinerState, error)
	CountWinners(context.Context, []address.Address, abi.ChainEpoch, abi.ChainEpoch) ([]dtypes.CountWinners, error)
}

type MockMinerMgr struct {
}

var _ IMinerManager = &MockMinerMgr{}

func (m MockMinerMgr) UpdateAddress(context.Context, int64, int64) ([]dtypes.MinerInfo, error) {
	return nil, nil
}

func (m MockMinerMgr) ListAddress(context.Context) ([]dtypes.MinerInfo, error) {
	return nil, nil
}

func (m MockMinerMgr) StatesForMining(context.Context, []address.Address) ([]dtypes.MinerState, error) {
	return nil, nil
}

func (m MockMinerMgr) CountWinners(context.Context, []address.Address, abi.ChainEpoch, abi.ChainEpoch) ([]dtypes.CountWinners, error) {
	return nil, nil
}
