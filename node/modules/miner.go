package modules

import (
	"context"

	"github.com/filecoin-project/venus/pkg/constants"
	chainV1API "github.com/filecoin-project/venus/venus-shared/api/chain/v1"
	"github.com/ipfs-force-community/sophon-miner/f3participant"
	"github.com/ipfs-force-community/sophon-miner/lib/journal"
	"github.com/ipfs-force-community/sophon-miner/miner"
	"github.com/ipfs-force-community/sophon-miner/node/config"
	"github.com/ipfs-force-community/sophon-miner/node/modules/helpers"
	minermanager "github.com/ipfs-force-community/sophon-miner/node/modules/miner-manager"
	"github.com/ipfs-force-community/sophon-miner/node/modules/slashfilter"
	logging "github.com/ipfs/go-log/v2"
	"go.uber.org/fx"
)

var log = logging.Logger("modules/miner")

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

func NewMultiParticipant(lc fx.Lifecycle,
	mCtx helpers.MetricsCtx,
	cfg *config.MinerConfig,
	minerManager minermanager.MinerManageAPI,
) (*f3participant.MultiParticipant, error) {
	if cfg.F3Node == nil || len(cfg.F3Node.Addr) == 0 {
		log.Warnf("f3 node is not configured")
		return nil, nil
	}
	if constants.DisableF3 {
		log.Warnf("f3 is disabled")
		return nil, nil
	}

	node, close, err := chainV1API.DialFullNodeRPC(mCtx, cfg.F3Node.Addr, cfg.F3Node.Token, nil)
	if err != nil {
		return nil, err
	}

	running, err := node.F3IsRunning(mCtx)
	if err != nil || !running {
		log.Warnf("f3 is not running: %v", err)
		return nil, nil
	}

	mp, err := f3participant.NewMultiParticipant(mCtx, node, minerManager)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return mp.Start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			close()
			return mp.Stop(ctx)
		},
	})

	return mp, nil
}
