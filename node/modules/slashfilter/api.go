package slashfilter

import (
	"context"
	"time"

	"github.com/filecoin-project/go-state-types/abi"

	vtypes "github.com/filecoin-project/venus/venus-shared/types"
)

type BlockStoreType string

const (
	Local BlockStoreType = "local"
	MySQL BlockStoreType = "mysql"
)

type StateMining int

const (
	Mining StateMining = iota
	Success
	Timeout
	ChainForked
	Error
)

func (sm StateMining) String() string {
	switch sm {
	case Mining:
		return "Mining"
	case Success:
		return "Success"
	case Timeout:
		return "TimeOut"
	case ChainForked:
		return "ChainForked"
	case Error:
		return "Error"
	default:
		return "unknown"
	}
}

type SlashFilterAPI interface {
	HasBlock(ctx context.Context, bh *vtypes.BlockHeader) (bool, error)
	MinedBlock(ctx context.Context, bh *vtypes.BlockHeader, parentEpoch abi.ChainEpoch) error
	PutBlock(ctx context.Context, bh *vtypes.BlockHeader, parentEpoch abi.ChainEpoch, t time.Time, state StateMining) error
}
