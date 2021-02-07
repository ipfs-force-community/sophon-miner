package node

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	metricsi "github.com/ipfs/go-metrics-interface"

	"github.com/filecoin-project/go-address"
	logging "github.com/ipfs/go-log"
	ci "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-peerstore/pstoremem"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	record "github.com/libp2p/go-libp2p-record"
	"github.com/libp2p/go-libp2p/p2p/net/conngater"
	"github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/venus-miner/api"
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
	"github.com/filecoin-project/venus-miner/node/modules/lp2p"
	"github.com/filecoin-project/venus-miner/node/modules/posterkeymgr"
	"github.com/filecoin-project/venus-miner/node/modules/prover_ctor"
	"github.com/filecoin-project/venus-miner/node/repo"
	"github.com/filecoin-project/venus-miner/sector-storage/ffiwrapper"
	"github.com/filecoin-project/venus-miner/sector-storage/stores"
	"github.com/filecoin-project/venus-miner/system"
)

//nolint:deadcode,varcheck
var log = logging.Logger("builder")

// special is a type used to give keys to modules which
//  can't really be identified by the returned type
type special struct{ id int }

//nolint:golint
var (
	DefaultTransportsKey = special{0}  // Libp2p option
	DiscoveryHandlerKey  = special{2}  // Private type
	AddrsFactoryKey      = special{3}  // Libp2p option
	SmuxTransportKey     = special{4}  // Libp2p option
	RelayKey             = special{5}  // Libp2p option
	SecurityKey          = special{6}  // Libp2p option
	BaseRoutingKey       = special{7}  // fx groups + multiret
	NatPortMapKey        = special{8}  // Libp2p option
	ConnectionManagerKey = special{9}  // Libp2p option
	AutoNATSvcKey        = special{10} // Libp2p option
	BandwidthReporterKey = special{11} // Libp2p option
	ConnGaterKey         = special{12} // libp2p option
)

type invoke int

// Invokes are called in the order they are defined.
//nolint:golint
const (
	// InitJournal at position 0 initializes the journal global var as soon as
	// the system starts, so that it's available for all other components.
	InitJournalKey = invoke(iota)

	// System processes.
	InitMemoryWatchdog

	// libp2p
	PstoreAddSelfKeysKey
	StartListeningKey

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

		Override(new(record.Validator), modules.RecordValidator),
		Override(new(dtypes.Bootstrapper), dtypes.Bootstrapper(false)),
		Override(new(dtypes.ShutdownChan), make(chan struct{})),

		// Filecoin modules

	}
}

func libp2p() Option {
	return Options(
		Override(new(peerstore.Peerstore), pstoremem.NewPeerstore),

		Override(DefaultTransportsKey, lp2p.DefaultTransports),

		Override(new(lp2p.RawHost), lp2p.Host),
		Override(new(host.Host), lp2p.RoutedHost),
		Override(new(lp2p.BaseIpfsRouting), lp2p.DHTRouting(dht.ModeAuto)),

		Override(DiscoveryHandlerKey, lp2p.DiscoveryHandler),
		Override(AddrsFactoryKey, lp2p.AddrsFactory(nil, nil)),
		Override(SmuxTransportKey, lp2p.SmuxTransport(true)),
		Override(RelayKey, lp2p.NoRelay()),
		Override(SecurityKey, lp2p.Security(true, false)),

		Override(BaseRoutingKey, lp2p.BaseRouting),
		Override(new(routing.Routing), lp2p.Routing),

		Override(NatPortMapKey, lp2p.NatPortMap),
		Override(BandwidthReporterKey, lp2p.BandwidthCounter),

		Override(ConnectionManagerKey, lp2p.ConnectionManager(50, 200, 20*time.Second, nil)),
		Override(AutoNATSvcKey, lp2p.AutoNATService),

		Override(new(*dtypes.ScoreKeeper), lp2p.ScoreKeeper),
		Override(new(*pubsub.PubSub), lp2p.GossipSub),
		Override(new(*config.Pubsub), func(bs dtypes.Bootstrapper) *config.Pubsub {
			return &config.Pubsub{
				Bootstrapper: bool(bs),
			}
		}),

		Override(PstoreAddSelfKeysKey, lp2p.PstoreAddSelfKeys),

		Override(new(*conngater.BasicConnectionGater), lp2p.ConnGater),
		Override(ConnGaterKey, lp2p.ConnGaterOption),
	)
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

		libp2p(),

		// miner
		ApplyIf(isType(repo.Miner),
			Override(new(dtypes.NetworkName), modules.MinerNetworkName),
			Override(new(*stores.Index), stores.NewIndex),
			Override(new(stores.SectorIndex), From(new(*stores.Index))),
			Override(new(stores.LocalStorage), From(new(repo.LockedRepo))),
		),
	)
}

// Config sets up constructors based on the provided Config
func ConfigCommon( cfg *config.Common) Option {
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
		ApplyIf(func(s *Settings) bool { return s.Online },
			Override(StartListeningKey, lp2p.StartListening(cfg.Libp2p.ListenAddresses)),
			Override(ConnectionManagerKey, lp2p.ConnectionManager(
				cfg.Libp2p.ConnMgrLow,
				cfg.Libp2p.ConnMgrHigh,
				time.Duration(cfg.Libp2p.ConnMgrGrace),
				cfg.Libp2p.ProtectedPeers)),
			Override(new(*pubsub.PubSub), lp2p.GossipSub),
			Override(new(*config.Pubsub), &cfg.Pubsub),
		),
		Override(AddrsFactoryKey, lp2p.AddrsFactory(
			cfg.Libp2p.AnnounceAddresses,
			cfg.Libp2p.NoAnnounceAddresses)),
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

			Override(new(ci.PrivKey), lp2p.PrivKey),
			Override(new(ci.PubKey), ci.PrivKey.GetPublic),
			Override(new(peer.ID), peer.IDFromPublicKey),

			Override(new(types.KeyStore), modules.KeyStore),

			Override(new(*dtypes.APIAlg), modules.APISecret),

			ApplyIf(isType(repo.Miner), ConfigPostOptions(cctx, c)),
		)(settings)
	}
}

var (
	CLIFlagMinerAddress = &cli.StringFlag{
		Name:     "addr",
		Usage:    "设置挖矿的地址",
		Value:    "",
		Required: false,
	}

	CLIFLAGBlockRecord = &cli.StringFlag{
		Name:    "block_record",
		Usage:   "记录已经产生的区块用于防止重复出块的问题 取值（localdb, cache）mysql待完成",
		EnvVars: []string{"FORCE_Block_Record"},
		Value:   "localdb",
	}
)

func ConfigPostConfig(cctx *cli.Context, cfg *config.MinerConfig) (*config.MinerConfig, error) {
	if cctx.IsSet(CLIFlagMinerAddress.Name) {
		addrs := cctx.StringSlice(CLIFlagMinerAddress.Name)
		for _, addrStr := range addrs {
			posterAddr, err := config.ParserPosterAddr(addrStr)
			if err != nil {
				return nil, xerrors.Errorf("invalid miner address in addr flag")
			}
			if posterAddr.Addr.Protocol() != address.ID {
				return nil, xerrors.Errorf("not the expect address type")
			}
			cfg.PosterAddrs = append(cfg.PosterAddrs, posterAddr)
		}
	}

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

		Override(new(dtypes.NetworkName), modules.MinerNetworkName),
		Override(new(dtypes.MinerAddress), modules.MinerAddress),

		Override(new(posterkeymgr.IActorMgr), posterkeymgr.NewActorMgr),
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

	return Options(
		blockRecordOp,
		Override(new(prover_ctor.WinningPostConstructor), prover_ctor.WinningPostProverCCTor),
		Override(new(miner.BlockMinerApi), modules.SetupPostBlockProducer),
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

func newBlockRecord(t string) (Option, error) {
	if t == block_recorder.LocalDb {
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
