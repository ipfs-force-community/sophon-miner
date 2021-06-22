package api

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
)

type MinerAPI interface {
	Common

	UpdateAddress(context.Context, int64, int64) ([]dtypes.MinerInfo, error)                                        //perm:write
	ListAddress(context.Context) ([]dtypes.MinerInfo, error)                                                        //perm:read
	StatesForMining(context.Context, []address.Address) ([]dtypes.MinerState, error)                                //perm:read
	CountWinners(context.Context, []address.Address, abi.ChainEpoch, abi.ChainEpoch) ([]dtypes.CountWinners, error) //perm:read
	Start(context.Context, address.Address) error                                                                   //perm:admin
	Stop(context.Context, address.Address) error                                                                    //perm:admin
}
