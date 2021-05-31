package minermanage

import (
	"context"

	"github.com/filecoin-project/go-address"

	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
)

const (
	Local = "local"
	MySQL = "mysql"
	Auth  = "auth"
)

type MinerManageAPI interface {
	//Put(ctx context.Context, addr dtypes.MinerInfo) error
	//Set(ctx context.Context, addr dtypes.MinerInfo) error
	Has(ctx context.Context, checkAddr address.Address) bool
	Get(ctx context.Context, checkAddr address.Address) *dtypes.MinerInfo
	List(ctx context.Context) ([]dtypes.MinerInfo, error)
	Update(ctx context.Context, skip, limit int64) ([]dtypes.MinerInfo, error)
	//Remove(ctx context.Context, rmAddr address.Address) error
	Count(ctx context.Context) int
}
