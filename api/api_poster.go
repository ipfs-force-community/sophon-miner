package api

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
)

type MinerAPI interface {
	Common

	UpdateAddress(context.Context, int64, int64) ([]dtypes.MinerInfo, error)
	ListAddress(context.Context) ([]dtypes.MinerInfo, error)
	StatesForMining(context.Context, []address.Address) ([]dtypes.MinerState, error)
	CountWinners(context.Context, []address.Address, abi.ChainEpoch, abi.ChainEpoch) ([]dtypes.CountWinners, error)
	Start(context.Context, address.Address) error
	Stop(context.Context, address.Address) error
}
