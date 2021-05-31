package node

import (
	"context"
	"encoding/json"
	"errors"

	logging "github.com/ipfs/go-log"
	metricsi "github.com/ipfs/go-metrics-interface"
	"github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/venus-miner/api"
	"github.com/filecoin-project/venus-miner/chain/gen/slashfilter"
	"github.com/filecoin-project/venus-miner/chain/types"
	"github.com/filecoin-project/venus-miner/journal"
	_ "github.com/filecoin-project/venus-miner/lib/sigs/bls"
	_ "github.com/filecoin-project/venus-miner/lib/sigs/secp"
	"github.com/filecoin-project/venus-miner/miner"
	"github.com/filecoin-project/venus-miner/node/config"
	"github.com/filecoin-project/venus-miner/node/impl"
	"github.com/filecoin-project/venus-miner/node/impl/common"
	"github.com/filecoin-project/venus-miner/node/modules"
	"github.com/filecoin-project/venus-miner/node/modules/block_recorder"
	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
	"github.com/filecoin-project/venus-miner/node/modules/helpers"
	"github.com/filecoin-project/venus-miner/node/modules/minermanage"
	"github.com/filecoin-project/venus-miner/node/modules/minermanage/auth"
	"github.com/filecoin-project/venus-miner/node/modules/minermanage/local"
	"github.com/filecoin-project/venus-miner/node/modules/minermanage/mysql"
	"github.com/filecoin-project/venus-miner/node/repo"
	"github.com/filecoin-project/venus-miner/sector-storage/ffiwrapper"
	"github.com/filecoin-project/venus-miner/system"
)

//nolint:deadcode,varcheck
var log = logging.Logger("builder")

// special is a type used to give keys to modules which
//  can't really be identified by the returned type
type special struct{ id int } //nolint

type invoke int

// Invokes are called in the order they are defined.
//nolint:golint
const (
	// InitJournal at position 0 initializes the journal global var as soon as
	// the system starts, so that it's available for all other components.
	InitJournalKey = invoke(iota)

	// System processes.
	InitMemoryWatchdog

	// daemon
	ExtractApiKey

	SetApiEndpointKey

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

	nodeType repo.RepoType

	Online bool // Online option applied
	Config bool // Config option applied
}

func defaults() []Option {
	return []Option{
		// global system journal.
		Override(new(journal.DisabledEvents), journal.EnvDisabledEvents),
		Override(new(journal.Journal), modules.OpenFilesystemJournal),

		Override(new(system.MemoryConstraints), modules.MemoryConstraints),
		Override(InitMemoryWatchdog, modules.MemoryWatchdog),

		Override(new(helpers.MetricsCtx), func() context.Context {
			return metricsi.CtxScope(context.Background(), "venus-miner")
		}),

		Override(new(dtypes.ShutdownChan), make(chan struct{})),

		// Filecoin modules
	}
}

func isType(t repo.RepoType) func(s *Settings) bool {
	return func(s *Settings) bool { return s.nodeType == t }
}

// Online sets up basic libp2p node
func Online() Option {

	return Options(
		// make sure that online is applied before Config.
		// This is important because Config overrides some of Online units
		func(s *Settings) error { s.Online = true; return nil },
		ApplyIf(func(s *Settings) bool { return s.Config },
			Error(errors.New("the Online option must be set before Config option")),
		),

		// miner
		ApplyIf(isType(repo.Miner)),
	)
}

// Config sets up constructors based on the provided Config
func ConfigCommon(cfg *config.Common) Option {
	return Options(
		func(s *Settings) error { s.Config = true; return nil },
		Override(new(dtypes.APIEndpoint), func() (dtypes.APIEndpoint, error) {
			return multiaddr.NewMultiaddr(cfg.API.ListenAddress)
		}),
		Override(SetApiEndpointKey, func(lr repo.LockedRepo, e dtypes.APIEndpoint) error {
			return lr.SetAPIEndpoint(e)
		}),
		Override(new(ffiwrapper.URLs), func(e dtypes.APIEndpoint) (ffiwrapper.URLs, error) {
			ip := cfg.API.RemoteListenAddress

			var urls ffiwrapper.URLs
			urls = append(urls, "http://"+ip+"/remote") // TODO: This makes no assumptions, and probably could...
			return urls, nil
		}),
	)
}

func Repo(cctx *cli.Context, r repo.Repo) Option {
	return func(settings *Settings) error {
		lr, err := r.Lock(settings.nodeType)
		if err != nil {
			return err
		}
		c, err := lr.Config()
		if err != nil {
			return err
		}

		return Options(
			Override(new(repo.LockedRepo), modules.LockedRepo(lr)), // module handles closing

			Override(new(dtypes.MetadataDS), modules.Datastore),

			Override(new(types.KeyStore), modules.KeyStore),

			Override(new(*dtypes.APIAlg), modules.APISecret),

			ApplyIf(isType(repo.Miner), ConfigPostOptions(cctx, c)),
		)(settings)
	}
}

var (
	CLIFLAGBlockRecord = &cli.StringFlag{
		Name:    "block_record",
		Usage:   "记录已经产生的区块用于防止重复出块的问题 取值（localdb, cache）mysql待完成",
		EnvVars: []string{"FORCE_Block_Record"},
		Value:   "localdb",
	}
)

func ConfigPostConfig(cctx *cli.Context, cfg *config.MinerConfig) (*config.MinerConfig, error) {
	if cctx.IsSet(CLIFLAGBlockRecord.Name) {
		cfg.BlockRecord = cctx.String(CLIFLAGBlockRecord.Name)
	}

	configStr, _ := json.MarshalIndent(cfg, "", "\t")
	log.Warnf("final config: \n%v", string(configStr))
	return cfg, nil
}

func ConfigPostOptions(cctx *cli.Context, c interface{}) Option {
	postCfg, ok := c.(*config.MinerConfig)
	if !ok {
		return Error(xerrors.Errorf("invalid config from repo, got: %T", c))
	}

	scfg, err := ConfigPostConfig(cctx, postCfg)
	if err != nil {
		return Error(xerrors.Errorf("error to parse config and flag %v", err))
	}
	shareOps := Options(
		Override(new(*config.MinerConfig), scfg),

		Override(new(api.Common), From(new(common.CommonAPI))),
		Override(new(ffiwrapper.Verifier), ffiwrapper.ProofVerifier),
	)

	opt, err := PostWinningOptions(scfg)
	if err != nil {
		return Error(xerrors.Errorf("error to constructor poster %v", err))
	}

	return Options(
		ConfigCommon(&postCfg.Common),
		shareOps,
		opt,
	)
}

func PostWinningOptions(postCfg *config.MinerConfig) (Option, error) {
	blockRecordOp, err := newBlockRecord(postCfg.BlockRecord)
	if err != nil {
		return nil, err
	}

	minerManageAPIOp, err := newMinerManageAPI(postCfg.Db)
	if err != nil {
		return nil, err
	}

	slashFilterAPIOp, err := newSlashFilterAPI(postCfg.Db)
	if err != nil {
		return nil, err
	}
	
	return Options(
		blockRecordOp,
		minerManageAPIOp,
		slashFilterAPIOp,
		Override(new(*config.GatewayNode), postCfg.Gateway),
		Override(new(miner.MiningAPI), modules.NewWiningPoster),
	), nil
}

func MinerAPI(out *api.MinerAPI) Option {
	return Options(
		ApplyIf(func(s *Settings) bool { return s.Config },
			Error(errors.New("the Poster option must be set before Config option")),
		),
		ApplyIf(func(s *Settings) bool { return s.Online },
			Error(errors.New("the Poster option must be set before Online option")),
		),

		func(s *Settings) error {
			s.nodeType = repo.Miner
			return nil
		},

		func(s *Settings) error {
			resAPI := &impl.MinerAPI{}
			s.invokes[ExtractApiKey] = fx.Populate(resAPI)
			*out = resAPI
			return nil
		},
	)
}

func newMinerManageAPI(dbConfig *config.MinerDbConfig) (Option, error) {
	switch dbConfig.Type {
	case minermanage.Local:
		return Override(new(minermanage.MinerManageAPI), local.NewMinerManger), nil
	case minermanage.MySQL:
		return Override(new(minermanage.MinerManageAPI), mysql.NewMinerManger(&dbConfig.MySQL)), nil
	case minermanage.Auth:
		return Override(new(minermanage.MinerManageAPI), auth.NewMinerManager(dbConfig.Auth.ListenAPI, "venus-miner", dbConfig.Auth.Token)), nil
	default:

	}

	return nil, xerrors.Errorf("unsupport db type")
}

func newSlashFilterAPI(dbConfig *config.MinerDbConfig) (Option, error) {
	switch dbConfig.Type{
	case minermanage.Local:
		return Override(new(slashfilter.SlashFilterAPI), slashfilter.NewLocal), nil
	case minermanage.MySQL:
		return Override(new(slashfilter.SlashFilterAPI), slashfilter.NewMysqlSlashFilter(&dbConfig.MySQL)), nil
	default:

	}

	return nil, xerrors.Errorf("unsupport db type")
}

func newBlockRecord(t string) (Option, error) {
	if t == block_recorder.Local {
		return Override(new(block_recorder.IBlockRecord), block_recorder.NewLocalDBRecord), nil
	} else if t == block_recorder.Cache {
		return Override(new(block_recorder.IBlockRecord), block_recorder.NewCacheRecord), nil
	} else {
		return nil, xerrors.Errorf("unsupport block record type")
	}
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
		return nil, xerrors.Errorf("applying node options failed: %w", err)
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
		return nil, xerrors.Errorf("starting node: %w", err)
	}

	return app.Stop, nil
}
