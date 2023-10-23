package slashfilter

import (
	"context"
	"errors"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs-force-community/sophon-miner/types"

	vtypes "github.com/filecoin-project/venus/venus-shared/types"
)

var TimeOffsetMiningFaults = errors.New("time-offset mining faults")
var ParentGrindingFaults = errors.New("parent-grinding fault")

type BlockStoreType string

const (
	Local BlockStoreType = "local"
	MySQL BlockStoreType = "mysql"
)

type SlashFilterAPI interface {
	HasBlock(ctx context.Context, bh *vtypes.BlockHeader) (bool, error)
	MinedBlock(ctx context.Context, bh *vtypes.BlockHeader, parentEpoch abi.ChainEpoch) error
	PutBlock(ctx context.Context, bh *vtypes.BlockHeader, parentEpoch abi.ChainEpoch, t time.Time, state types.StateMining) error
	ListBlock(ctx context.Context, params *types.BlocksQueryParams) ([]MinedBlock, error)
}
