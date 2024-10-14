package node

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs-force-community/sophon-auth/jwtclient"

	"github.com/multiformats/go-multiaddr"

	logging "github.com/ipfs/go-log/v2"
	metricsi "github.com/ipfs/go-metrics-interface"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	"github.com/ipfs-force-community/sophon-miner/api"
	"github.com/ipfs-force-community/sophon-miner/f3participant"
	"github.com/ipfs-force-community/sophon-miner/lib/journal"
	"github.com/ipfs-force-community/sophon-miner/lib/metrics"
	"github.com/ipfs-force-community/sophon-miner/miner"
	"github.com/ipfs-force-community/sophon-miner/node/config"
	"github.com/ipfs-force-community/sophon-miner/node/impl"
	"github.com/ipfs-force-community/sophon-miner/node/impl/common"
	"github.com/ipfs-force-community/sophon-miner/node/modules"
	"github.com/ipfs-force-community/sophon-miner/node/modules/helpers"
	minerecorder "github.com/ipfs-force-community/sophon-miner/node/modules/mine-recorder"
	minermanager "github.com/ipfs-force-community/sophon-miner/node/modules/miner-manager"
	"github.com/ipfs-force-community/sophon-miner/node/modules/slashfilter"
	"github.com/ipfs-force-community/sophon-miner/node/repo"
	"github.com/ipfs-force-community/sophon-miner/types"
)

//nolint:deadcode,varcheck
var log = logging.Logger("builder")

// special is a type used to give keys to modules which
//
//	can't really be identified by the returned type
type special struct{ id int } //nolint

type invoke int

// Invokes are called in the order they are defined.
//
//nolint:golint
const (
	// InitJournal at position 0 initializes the journal global var as soon as
	// the system starts, so that it's available for all other components.
	InitJournalKey = invoke(iota)

	// daemon
	ExtractApiKey

	SetApiEndpointKey

	SetRecorderDatastoreKey // set recorder datastore

	LaunchMetricsSampleThread

	F3ParticipantKey

	_nInvokes // keep this last
)

type Settings struct {
	// modules is a map of constructors for DI
	//
	// In most cases the index will be a reflect. Type of element returned by
	// the constructor, but for some 'constructors' it's hard to specify what's
	// the return type should be (or the constructor returns fx group)
	modules map[interface{}]fx.Option

	// invokes are separate from modules as they can't be referenced by return
	// type, and must be applied in correct order
	invokes []fx.Option
}

func defaults() []Option {
	return []Option{
		// global system journal.
		Override(new(journal.DisabledEvents), journal.EnvDisabledEvents),
		Override(new(journal.Journal), modules.OpenFilesystemJournal),

		Override(new(helpers.MetricsCtx), func() context.Context {
			return metricsi.CtxScope(context.Background(), "sophon-miner")
		}),

		Override(new(types.ShutdownChan), make(chan struct{})),
	}
}

func Repo(cctx *cli.Context, r repo.Repo) Option {
	return func(settings *Settings) error {
		lr, err := r.Lock()
		if err != nil {
			return err
		}
		c, err := lr.Config()
		if err != nil {
			return err
		}
		return Options(
			Override(new(repo.LockedRepo), modules.LockedRepo(lr)),
			Override(new(types.MetadataDS), modules.Datastore),
			ConfigMinerOptions(c),
		)(settings)
	}
}

func ConfigMinerOptions(c interface{}) Option {
	cfg, ok := c.(*config.MinerConfig)
	if !ok {
		return Error(fmt.Errorf("invalid config from repo, got: %T", c))
	}

	configStr, _ := json.MarshalIndent(cfg, "", "\t")
	log.Infof("final config: \n%v", string(configStr))

	shareOps := Options(
		Override(new(*config.MinerConfig), cfg),
		Override(new(*config.MySQLConfig), &cfg.SlashFilter.MySQL),
		Override(new(types.APIEndpoint), func() (types.APIEndpoint, error) {
			return multiaddr.NewMultiaddr(cfg.API.ListenAddress)
		}),
		Override(new(api.Common), From(new(common.CommonAPI))),
	)

	minerOps := Options(
		If(cfg.SlashFilter.Type == string(slashfilter.Local), Override(new(slashfilter.SlashFilterAPI), slashfilter.NewLocal)),
		If(cfg.SlashFilter.Type == string(slashfilter.MySQL), Override(new(slashfilter.SlashFilterAPI), slashfilter.NewMysql)),

		Override(new(jwtclient.IAuthClient), minermanager.NewVenusAuth(cfg.Auth.Addr, cfg.Auth.Token)),
		Override(new(minermanager.MinerManageAPI), minermanager.NewMinerManager),
		Override(SetRecorderDatastoreKey, func(ds types.MetadataDS) {
			if cfg.Recorder != nil && cfg.Recorder.Enable {
				if cfg.Recorder.MaxRecordPerQuery > 0 {
					minerecorder.MaxRecordPerQuery = cfg.Recorder.MaxRecordPerQuery
				}
				if cfg.Recorder.ExpireEpoch > 0 {
					minerecorder.ExpireEpoch = abi.ChainEpoch(cfg.Recorder.ExpireEpoch)
				}
				minerecorder.SetDatastore(ds)
			}
		}),
		Override(new(miner.MiningAPI), modules.NewMinerProcessor),
		Override(F3ParticipantKey, f3participant.NewMultiParticipant),

		Override(LaunchMetricsSampleThread, func(ctx helpers.MetricsCtx, api minermanager.MinerManageAPI) {
			go func() {
				// Record metrics
				tm := time.NewTicker(time.Second * 60)
				defer tm.Stop()

				for {
					select {
					case <-ctx.Done():
						log.Warnf("stop record metrics: %v", ctx.Err())
						return
					case <-tm.C:
						miners, err := api.List(ctx)
						if err != nil {
							log.Warnf("record metrics: list miner: %s", err)
						}
						minerInState := map[bool]int64{true: 0, false: 0}
						for _, miner := range miners {
							minerInState[miner.OpenMining] += 1
						}
						for state, num := range minerInState {
							stateStr := "open_for_mining"
							if !state {
								stateStr = "close_for_mining"
							}
							metrics.MinerNumInState.Set(ctx, stateStr, num)
						}
					}
				}
			}()
		}),
	)

	return Options(
		shareOps,
		minerOps,
		Override(SetApiEndpointKey, func(lr repo.LockedRepo, e types.APIEndpoint) error {
			return lr.SetAPIEndpoint(e)
		}),
	)
}

func MinerAPI(out *api.MinerAPI) Option {
	return Options(
		func(s *Settings) error {
			resAPI := &impl.MinerAPI{}
			s.invokes[ExtractApiKey] = fx.Populate(resAPI)
			*out = resAPI
			return nil
		},
	)
}

type FullOption = Option

type StopFunc func(context.Context) error

// New builds and starts new Filecoin node
func New(ctx context.Context, opts ...Option) (StopFunc, error) {
	settings := Settings{
		modules: map[interface{}]fx.Option{},
		invokes: make([]fx.Option, _nInvokes),
	}

	// apply module options in the right order
	if err := Options(Options(defaults()...), Options(opts...))(&settings); err != nil {
		return nil, fmt.Errorf("applying node options failed: %w", err)
	}

	// gather constructors for fx.Options
	ctors := make([]fx.Option, 0, len(settings.modules))
	for _, opt := range settings.modules {
		ctors = append(ctors, opt)
	}

	// fill holes in invokes for use in fx.Options
	for i, opt := range settings.invokes {
		if opt == nil {
			settings.invokes[i] = fx.Options()
		}
	}

	app := fx.New(
		fx.Options(ctors...),
		fx.Options(settings.invokes...),

		fx.NopLogger,
	)

	// TODO: we probably should have a 'firewall' for Closing signal
	//  on this context, and implement closing logic through lifecycles
	//  correctly
	if err := app.Start(ctx); err != nil {
		// comment fx.NopLogger few lines above for easier debugging
		return nil, fmt.Errorf("starting node: %w", err)
	}

	return app.Stop, nil
}
