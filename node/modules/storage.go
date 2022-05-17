package modules

import (
	"context"

	"go.uber.org/fx"

	"github.com/filecoin-project/venus-miner/chain/types"
	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
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

func KeyStore(lr repo.LockedRepo) (types.KeyStore, error) {
	return lr.KeyStore()
}

func Datastore(lc fx.Lifecycle, mctx helpers.MetricsCtx, r repo.LockedRepo) (dtypes.MetadataDS, error) {
	ctx := helpers.LifecycleCtx(mctx, lc)
	mds, err := r.Datastore(ctx, "/metadata")
	if err != nil {
		return nil, err
	}

	return mds, nil
}
