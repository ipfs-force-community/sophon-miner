package modules

import (
	"context"

	"go.uber.org/fx"

	"github.com/ipfs-force-community/sophon-miner/lib/journal"
	"github.com/ipfs-force-community/sophon-miner/miner"
	"github.com/ipfs-force-community/sophon-miner/node/config"
	"github.com/ipfs-force-community/sophon-miner/node/modules/helpers"
	minermanager "github.com/ipfs-force-community/sophon-miner/node/modules/miner-manager"
	"github.com/ipfs-force-community/sophon-miner/node/modules/slashfilter"

	chainV1API "github.com/filecoin-project/venus/venus-shared/api/chain/v1"
)

func NewMinerProcessor(lc fx.Lifecycle,
	mCtx helpers.MetricsCtx,
	api chainV1API.FullNode,
	cfg *config.MinerConfig,
	sfAPI slashfilter.SlashFilterAPI,
	minerManager minermanager.MinerManageAPI,
	j journal.Journal,
) (miner.MiningAPI, error) {
	ctx := helpers.LifecycleCtx(mCtx, lc)
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
