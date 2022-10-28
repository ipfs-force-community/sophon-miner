package miner_manager

import (
	"context"

	"github.com/filecoin-project/go-address"

	"github.com/filecoin-project/venus-miner/types"
)

//go:generate go run github.com/golang/mock/mockgen -destination=mock/miner_manager.go -package=mock . MinerManageAPI

type MinerManageAPI interface {
	Has(ctx context.Context, mAddr address.Address) bool
	Get(ctx context.Context, mAddr address.Address) (*types.MinerInfo, error)
	IsOpenMining(ctx context.Context, mAddr address.Address) bool
	OpenMining(ctx context.Context, mAddr address.Address) (*types.MinerInfo, error)
	CloseMining(ctx context.Context, mAddr address.Address) error
	List(ctx context.Context) (map[address.Address]*types.MinerInfo, error)
	Update(ctx context.Context, skip, limit int64) (map[address.Address]*types.MinerInfo, error)
}
