package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/venus/venus-shared/api/chain"
	v1 "github.com/filecoin-project/venus/venus-shared/api/chain/v1"
	"github.com/filecoin-project/venus/venus-shared/types"
	manet "github.com/multiformats/go-multiaddr/net"

	"github.com/filecoin-project/venus-miner/build"
	lcli "github.com/filecoin-project/venus-miner/cli"
	"github.com/filecoin-project/venus-miner/node/config"
	"github.com/filecoin-project/venus-miner/node/repo"
)

var initCmd = &cli.Command{
	Name:  "init",
	Usage: "Initialize a venus miner repo",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "listen",
			Usage: "host address and port",
			Value: "0.0.0.0:12308",
		},
		&cli.StringFlag{
			Name:     "api",
			Usage:    "full node api",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "auth-api",
			Usage:    "auth node api",
			Required: true,
		},
		&cli.StringSliceFlag{
			Name:  "gateway-api",
			Usage: "gateway api",
		},
		&cli.StringFlag{
			Name:     "token",
			Usage:    "full node token",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "slash-filter",
			Usage:    "the type of db type used to store block info required by slash filter, optional: local, mysql",
			Required: true,
		},
		&cli.StringFlag{
			Name:  "mysql-conn",
			Usage: "mysql conn info, eg. [user]:[password]@tcp([ip]:[port])/[db_name]?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s",
			Value: "",
		},
	},
	Action: func(cctx *cli.Context) error {
		log.Info("Initializing venus miner")

		ctx := lcli.ReqContext(cctx)

		log.Info("Trying to connect to full node RPC")

		fullnode := config.APIInfo{}
		if cctx.String("api") != "" && cctx.String("token") != "" {
			fullnode.Addr = cctx.String("api")
			fullnode.Token = cctx.String("token")
		}

		fullNodeAPI, closer, err := lcli.GetFullNodeAPI(cctx, &fullnode, "v1")
		if err != nil {
			return err
		}
		defer closer()

		log.Info("Checking if repo exists")

		repoPath := cctx.String(FlagMinerRepo)
		r, err := repo.NewFS(repoPath)
		if err != nil {
			return err
		}

		ok, err := r.Exists()
		if err != nil {
			return err
		}
		if ok {
			return fmt.Errorf("repo at '%s' is already initialized", cctx.String(FlagMinerRepo))
		}

		log.Info("Checking full node version")

		v, err := fullNodeAPI.Version(ctx)
		if err != nil {
			return err
		}

		if !v.APIVersion.EqMajorMinor(chain.FullAPIVersion1) {
			return fmt.Errorf("RemoteAPI version didn't match (expected %s, remote %s)", chain.FullAPIVersion1, v.APIVersion)
		}

		log.Info("Initializing repo")

		if err := r.Init(); err != nil {
			return err
		}

		if err := storageMinerInit(cctx, r, &fullnode); err != nil {
			log.Errorf("Failed to initialize venus-miner: %+v", err)
			path, err := homedir.Expand(repoPath)
			if err != nil {
				return err
			}
			log.Infof("Cleaning up %s after attempt...", path)
			if err := os.RemoveAll(path); err != nil {
				log.Errorf("Failed to clean up failed storage repo: %s", err)
			}
			return fmt.Errorf("init failed")
		}

		log.Info("Miner successfully init, you can now start it with 'venus-miner run'")

		return nil
	},
}

func storageMinerInit(cctx *cli.Context, r repo.Repo, fn *config.APIInfo) error {
	lr, err := r.Lock()
	if err != nil {
		return err
	}
	defer lr.Close() //nolint:errcheck

	// modify config
	log.Info("modify config ...")
	sfType := cctx.String("slash-filter")
	if sfType != "local" && sfType != "mysql" {
		return fmt.Errorf("wrong slash filter type")
	}
	log.Info("generate API ...")
	address := cctx.String("listen")
	a, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return fmt.Errorf("parsing address: %w", err)
	}

	ma, err := manet.FromNetAddr(a)
	if err != nil {
		return fmt.Errorf("creating api multiaddress: %w", err)
	}

	if err := lr.SetConfig(func(i interface{}) {
		cfg := i.(*config.MinerConfig)
		cfg.FullNode = fn

		gt := cctx.String("token")
		if cctx.IsSet("gateway-api") {
			cfg.Gateway = &config.GatewayNode{
				ListenAPI: cctx.StringSlice("gateway-api"),
				Token:     gt,
			}
		}

		if cctx.String("auth-api") != "" {
			cfg.Auth = &config.APIInfo{
				Addr:  cctx.String("auth-api"),
				Token: gt,
			}
		}

		cfg.API.ListenAddress = ma.String()
		cfg.SlashFilter.Type = sfType
		cfg.SlashFilter.MySQL.Conn = cctx.String("mysql-conn")
	}); err != nil {
		return fmt.Errorf("modify config failed: %w", err)
	}

	if err := lr.SetVersion(build.Version); err != nil {
		return fmt.Errorf("setting version: %w", err)
	}
	return nil
}

func SyncWait(ctx context.Context, fullNode v1.FullNode, watch bool) error {
	tick := time.Second / 4
	netParams, err := fullNode.StateGetNetworkParams(ctx)
	if err != nil {
		return err
	}

	lastLines := 0
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	samples := 8
	i := 0
	var firstApp, app, lastApp uint64

	state, err := fullNode.SyncState(ctx)
	if err != nil {
		return err
	}
	firstApp = state.VMApplied

	for {
		state, err := fullNode.SyncState(ctx)
		if err != nil {
			return err
		}

		if len(state.ActiveSyncs) == 0 {
			time.Sleep(time.Second)
			continue
		}

		head, err := fullNode.ChainHead(ctx)
		if err != nil {
			return err
		}

		working := -1
		for i, ss := range state.ActiveSyncs {
			switch ss.Stage {
			case types.StageSyncComplete:
			default:
				working = i
			case types.StageIdle:
				// not complete, not actively working
			}
		}

		if working == -1 {
			working = len(state.ActiveSyncs) - 1
		}

		ss := state.ActiveSyncs[working]
		workerID := ss.WorkerID

		var baseHeight abi.ChainEpoch
		var target []cid.Cid
		var theight abi.ChainEpoch
		var heightDiff int64

		if ss.Base != nil {
			baseHeight = ss.Base.Height()
			heightDiff = int64(ss.Base.Height())
		}
		if ss.Target != nil {
			target = ss.Target.Cids()
			theight = ss.Target.Height()
			heightDiff = int64(ss.Target.Height()) - heightDiff
		} else {
			heightDiff = 0
		}

		for i := 0; i < lastLines; i++ {
			fmt.Print("\r\x1b[2K\x1b[A")
		}

		fmt.Printf("Worker: %d; Base: %d; Target: %d (diff: %d)\n", workerID, baseHeight, theight, heightDiff)
		fmt.Printf("State: %s; Current Epoch: %d; Todo: %d\n", ss.Stage, ss.Height, theight-ss.Height)
		lastLines = 2

		if i%samples == 0 {
			lastApp = app
			app = state.VMApplied - firstApp
		}
		if i > 0 {
			fmt.Printf("Validated %d messages (%d per second)\n", state.VMApplied-firstApp, (app-lastApp)*uint64(time.Second/tick)/uint64(samples))
			lastLines++
		}

		_ = target // todo: maybe print? (creates a bunch of line wrapping issues with most tipsets)

		if !watch && time.Now().Unix()-int64(head.MinTimestamp()) < int64(netParams.BlockDelaySecs) {
			fmt.Println("\nDone!")
			return nil
		}

		select {
		case <-ctx.Done():
			fmt.Println("\nExit by user")
			return nil
		case <-ticker.C:
		}

		i++
	}
}
