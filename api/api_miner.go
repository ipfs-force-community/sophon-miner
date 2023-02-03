package api

import (
	"context"

	"github.com/filecoin-project/venus-miner/types"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
)

type MinerAPI interface {
	Common

	UpdateAddress(context.Context, int64, int64) ([]types.MinerInfo, error)                                        //perm:admin
	ListAddress(context.Context) ([]types.MinerInfo, error)                                                        //perm:read
	StatesForMining(context.Context, []address.Address) ([]types.MinerState, error)                                //perm:read
	CountWinners(context.Context, []address.Address, abi.ChainEpoch, abi.ChainEpoch) ([]types.CountWinners, error) //perm:read
	WarmupForMiner(context.Context, address.Address) error                                                         //perm:write
	Start(context.Context, []address.Address) error                                                                //perm:write
	Stop(context.Context, []address.Address) error                                                                 //perm:write
}
