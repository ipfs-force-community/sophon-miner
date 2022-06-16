package modules

import (
	"context"

	"go.uber.org/fx"

	"github.com/filecoin-project/venus-miner/lib/journal"
	"github.com/filecoin-project/venus-miner/miner"
	"github.com/filecoin-project/venus-miner/miner/slashfilter"
	"github.com/filecoin-project/venus-miner/node/config"
	"github.com/filecoin-project/venus-miner/node/modules/block_recorder"
	"github.com/filecoin-project/venus-miner/node/modules/minermanage"

	"github.com/filecoin-project/venus/pkg/util/ffiwrapper"
	v1api "github.com/filecoin-project/venus/venus-shared/api/chain/v1"
)

func NewWiningPoster(lc fx.Lifecycle,
	api v1api.FullNode,
	gtNode *config.GatewayNode,
	sfAPI slashfilter.SlashFilterAPI,
	verifier ffiwrapper.Verifier,
	minerManager minermanage.MinerManageAPI,
	j journal.Journal,
	blockRecord block_recorder.IBlockRecord,
) (miner.MiningAPI, error) {
	m := miner.NewMiner(api, gtNode, verifier, minerManager, sfAPI, j, blockRecord)

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
