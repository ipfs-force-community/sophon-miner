package modules

import (
	"context"

	"github.com/filecoin-project/venus-miner/node/modules/helpers"

	"go.uber.org/fx"

	"github.com/filecoin-project/venus-miner/lib/journal"
	"github.com/filecoin-project/venus-miner/miner"
	"github.com/filecoin-project/venus-miner/node/config"
	minermanager "github.com/filecoin-project/venus-miner/node/modules/miner-manager"
	"github.com/filecoin-project/venus-miner/node/modules/slashfilter"

	fullnode "github.com/filecoin-project/venus/venus-shared/api/chain/v1"
)

func NewMinerProcessor(lc fx.Lifecycle,
	mctx helpers.MetricsCtx,
	api fullnode.FullNode,
	cfg *config.MinerConfig,
	sfAPI slashfilter.SlashFilterAPI,
	minerManager minermanager.MinerManageAPI,
	j journal.Journal,
) (miner.MiningAPI, error) {
	ctx := helpers.LifecycleCtx(mctx, lc)
	m, err := miner.NewMiner(ctx, api, cfg, minerManager, sfAPI, j)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := m.Start(ctx); err != nil {
				return err
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return m.Stop(ctx)
		},
	})

	return m, err
}
