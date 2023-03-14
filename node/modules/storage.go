package modules

import (
	"context"

	types2 "github.com/filecoin-project/venus-miner/types"

	"go.uber.org/fx"

	"github.com/filecoin-project/venus-miner/node/modules/helpers"
	"github.com/filecoin-project/venus-miner/node/repo"
)

func LockedRepo(lr repo.LockedRepo) func(lc fx.Lifecycle) repo.LockedRepo {
	return func(lc fx.Lifecycle) repo.LockedRepo {
		lc.Append(fx.Hook{
			OnStop: func(_ context.Context) error {
				return lr.Close()
			},
		})

		return lr
	}
}

func Datastore(lc fx.Lifecycle, mctx helpers.MetricsCtx, r repo.LockedRepo) (types2.MetadataDS, error) {
	ctx := helpers.LifecycleCtx(mctx, lc)
	mds, err := r.Datastore(ctx, "/metadata")
	if err != nil {
		return nil, err
	}

	return mds, nil
}
