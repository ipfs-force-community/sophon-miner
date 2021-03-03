package modules

import (
	"context"

	"go.uber.org/fx"

	"github.com/filecoin-project/venus-miner/api"
	"github.com/filecoin-project/venus-miner/chain/gen/slashfilter"
	"github.com/filecoin-project/venus-miner/journal"
	"github.com/filecoin-project/venus-miner/miner"
	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
	"github.com/filecoin-project/venus-miner/node/modules/minermanage"
	"github.com/filecoin-project/venus-miner/sector-storage/ffiwrapper"
)

func NewWiningPoster(lc fx.Lifecycle,
	api api.FullNode,
	ds dtypes.MetadataDS,
	verifier ffiwrapper.Verifier,
	minerManager minermanage.MinerManageAPI,
	j journal.Journal,
	//blockRecord block_recorder.IBlockRecord
) (miner.MiningAPI, error) {
	m := miner.NewMiner(api, verifier, minerManager, slashfilter.New(ds), j)

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
