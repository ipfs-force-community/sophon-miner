package miner

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/filecoin-project/venus-miner/types"
)

type MiningAPI interface {
	IMinerMining
	IMinerManager
}

type IMinerMining interface {
	Start(context.Context) error
	Stop(context.Context) error
	ManualStart(context.Context, []address.Address) error
	ManualStop(context.Context, []address.Address) error
}

type IMinerManager interface {
	UpdateAddress(context.Context, int64, int64) ([]types.MinerInfo, error)
	ListAddress(context.Context) ([]types.MinerInfo, error)
	StatesForMining(context.Context, []address.Address) ([]types.MinerState, error)
	CountWinners(context.Context, []address.Address, abi.ChainEpoch, abi.ChainEpoch) ([]types.CountWinners, error)
}
