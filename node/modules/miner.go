package modules

import (
	"context"

	miner_manager "github.com/filecoin-project/venus-miner/node/modules/miner-manager"

	"go.uber.org/fx"

	"github.com/filecoin-project/venus-miner/lib/journal"
	"github.com/filecoin-project/venus-miner/miner"
	"github.com/filecoin-project/venus-miner/node/config"
	"github.com/filecoin-project/venus-miner/node/modules/slashfilter"

	"github.com/filecoin-project/venus/pkg/util/ffiwrapper"
	fullnode "github.com/filecoin-project/venus/venus-shared/api/chain/v1"
)

func NewMinerProcessor(lc fx.Lifecycle,
	api fullnode.FullNode,
	gtNode *config.GatewayNode,
	sfAPI slashfilter.SlashFilterAPI,
	verifier ffiwrapper.Verifier,
	minerManager miner_manager.MinerManageAPI,
	j journal.Journal,
) (miner.MiningAPI, error) {
	m := miner.NewMiner(api, gtNode, verifier, minerManager, sfAPI, j)

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
