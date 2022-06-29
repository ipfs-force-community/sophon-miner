package miner_manager

import (
	"context"

	"github.com/filecoin-project/go-address"

	"github.com/filecoin-project/venus-miner/types"
)

type MinerManageAPI interface {
	Has(ctx context.Context, checkAddr address.Address) bool
	Get(ctx context.Context, checkAddr address.Address) *types.MinerInfo
	List(ctx context.Context) ([]types.MinerInfo, error)
	Update(ctx context.Context, skip, limit int64) ([]types.MinerInfo, error)
	Count(ctx context.Context) int
}