package modules

import (
	"context"

	"go.uber.org/fx"

	"github.com/filecoin-project/venus-miner/lib/journal"
	"github.com/filecoin-project/venus-miner/miner"
	"github.com/filecoin-project/venus-miner/node/config"
	minermanager "github.com/filecoin-project/venus-miner/node/modules/miner-manager"
	"github.com/filecoin-project/venus-miner/node/modules/slashfilter"

	fullnode "github.com/filecoin-project/venus/venus-shared/api/chain/v1"
)

func NewMinerProcessor(lc fx.Lifecycle,
	api fullnode.FullNode,
	gtNode *config.GatewayNode,
	sfAPI slashfilter.SlashFilterAPI,
	minerManager minermanager.MinerManageAPI,
	j journal.Journal,
) (miner.MiningAPI, error) {
	m := miner.NewMiner(api, gtNode, minerManager, sfAPI, j)

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

	return m, nil
}
