package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-paramfetch"
	"github.com/filecoin-project/go-state-types/abi"

	miner0 "github.com/filecoin-project/specs-actors/actors/builtin/miner"

	"github.com/filecoin-project/venus/venus-shared/api"
	"github.com/filecoin-project/venus/venus-shared/api/chain/v1"
	types2 "github.com/filecoin-project/venus/venus-shared/types"

	"github.com/filecoin-project/venus-miner/build"
	"github.com/filecoin-project/venus-miner/chain/types"
	lcli "github.com/filecoin-project/venus-miner/cli"
	"github.com/filecoin-project/venus-miner/node/config"
	"github.com/filecoin-project/venus-miner/node/repo"
)

var initCmd = &cli.Command{
	Name:  "init",
	Usage: "Initialize a venus miner repo",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "nettype",
			Usage:       "network type, one of: mainnet, debug, 2k, calibnet, butterfly",
			Value:       "mainnet",
			DefaultText: "mainnet",
			Required:    false,
		},
		&cli.StringFlag{
			Name:     "api",
			Usage:    "full node api",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "token",
			Usage:    "full node token",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "auth-api",
			Usage:    "auth node api",
			Required: true,
		},
		&cli.StringFlag{
			Name:  "auth-token",
			Usage: "auth node token",
			Value: "",
		},
		&cli.StringSliceFlag{
			Name:  "gateway-api",
			Usage: "gateway api",
		},
		&cli.StringFlag{
			Name:  "gateway-token",
			Usage: "gateway token",
			Value: "",
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

		log.Info("Initializing build params")

		if err := build.InitNetWorkParams(cctx.String("nettype")); err != nil {
			return err
		}

		log.Info("Trying to connect to full node RPC")

		fullnode := config.FullNode{}
		if cctx.String("api") != "" && cctx.String("token") != "" {
			fullnode.ListenAPI = cctx.String("api")
			fullnode.Token = cctx.String("token")
		}

		fullNodeAPI, closer, err := lcli.GetFullNodeAPIV1(cctx, fullnode)
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
			return xerrors.Errorf("repo at '%s' is already initialized", cctx.String(FlagMinerRepo))
		}

		log.Info("Checking full node version")

		v, err := fullNodeAPI.Version(ctx)
		if err != nil {
			return err
		}

		if !v.APIVersion.EqMajorMinor(api.FullAPIVersion1) {
			return xerrors.Errorf("Remote API version didn't match (expected %s, remote %s)", api.FullAPIVersion1, v.APIVersion)
		}

		log.Info("Initializing repo")

		if err := r.Init(repo.Miner); err != nil {
			return err
		}

		if err := storageMinerInit(cctx, r, fullnode); err != nil {
			log.Errorf("Failed to initialize venus-miner: %+v", err)
			path, err := homedir.Expand(repoPath)
			if err != nil {
				return err
			}
			log.Infof("Cleaning up %s after attempt...", path)
			if err := os.RemoveAll(path); err != nil {
				log.Errorf("Failed to clean up failed storage repo: %s", err)
			}
			return xerrors.Errorf("Storage-miner init failed")
		}

		log.Info("Miner successfully init, you can now start it with 'venus-miner run'")

		return nil
	},
}

func storageMinerInit(cctx *cli.Context, r repo.Repo, fn config.FullNode) error {
	lr, err := r.Lock(repo.Miner)
	if err != nil {
		return err
	}
	defer lr.Close() //nolint:errcheck

	//log.Info("Initializing libp2p identity")
	//
	//p2pSk, err := makeHostKey(lr)
	//if err != nil {
	//	return xerrors.Errorf("make host key: %w", err)
	//}
	//
	//peerID, err := peer.IDFromPrivateKey(p2pSk)
	//if err != nil {
	//	return xerrors.Errorf("peer ID from private key: %w", err)
	//}
	//log.Infow("init new peer: %s", peerID)

	//mds, err := lr.Datastore(context.TODO(), "/metadata")
	//if err != nil {
	//	return err
	//}

	// modify config
	log.Info("modify config ...")
	sfType := cctx.String("slash-filter")
	if sfType != "local" && sfType != "mysql" {
		return xerrors.Errorf("wrong slash filter type")
	}

	if err := lr.SetConfig(func(i interface{}) {
		cfg := i.(*config.MinerConfig)
		cfg.FullNode = fn

		if cctx.IsSet("gateway-api") {
			gt := cctx.String("gateway-token")
			if gt == "" {
				gt = cctx.String("token")
			}
			cfg.Gateway = &config.GatewayNode{
				ListenAPI: cctx.StringSlice("gateway-api"),
				Token:     gt,
			}
		}

		if cctx.String("auth-api") != "" {
			cfg.Db.Type = "auth"
			cfg.Db.Auth = &config.AuthConfig{
				ListenAPI: cctx.String("auth-api"),
				Token:     cctx.String("auth-token"),
			}
		}

		cfg.Db.SFType = sfType
		if cfg.Db.SFType == "mysql" {
			cfg.Db.MySQL.Conn = cctx.String("mysql-conn")
		}
	}); err != nil {
		return xerrors.Errorf("modify config failed: %w", err)
	}

	return nil
}

func makeHostKey(ctx context.Context, lr repo.LockedRepo) (crypto.PrivKey, error) { //nolint
	pk, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, err
	}

	ks, err := lr.KeyStore()
	if err != nil {
		return nil, err
	}

	kbytes, err := crypto.MarshalPrivateKey(pk)
	if err != nil {
		return nil, err
	}

	if err := ks.Put("libp2p-host", types.KeyInfo{
		Type:       "libp2p-host",
		PrivateKey: kbytes,
	}); err != nil {
		return nil, err
	}

	return pk, nil
}

func SyncWait(ctx context.Context, napi v1.FullNode, watch bool) error {
	tick := time.Second / 4

	lastLines := 0
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	samples := 8
	i := 0
	var firstApp, app, lastApp uint64

	state, err := napi.SyncState(ctx)
	if err != nil {
		return err
	}
	firstApp = state.VMApplied

	for {
		state, err := napi.SyncState(ctx)
		if err != nil {
			return err
		}

		if len(state.ActiveSyncs) == 0 {
			time.Sleep(time.Second)
			continue
		}

		head, err := napi.ChainHead(ctx)
		if err != nil {
			return err
		}

		working := -1
		for i, ss := range state.ActiveSyncs {
			switch ss.Stage {
			case types2.StageSyncComplete:
			default:
				working = i
			case types2.StageIdle:
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

		if !watch && time.Now().Unix()-int64(head.MinTimestamp()) < int64(build.BlockDelaySecs) {
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

func fetchingProofParameters(ctx context.Context) error { // nolint
	ss := make([]uint64, 0)

	log.Info("SupportedProofTypes: ", miner0.SupportedProofTypes)
	for spf := range miner0.SupportedProofTypes {
		switch spf {
		case abi.RegisteredSealProof_StackedDrg2KiBV1:
			ss = append(ss, 2048)
		case abi.RegisteredSealProof_StackedDrg8MiBV1:
			ss = append(ss, 8<<20)
		case abi.RegisteredSealProof_StackedDrg512MiBV1:
			ss = append(ss, 512<<20)
		case abi.RegisteredSealProof_StackedDrg32GiBV1:
			ss = append(ss, 32<<30)
		case abi.RegisteredSealProof_StackedDrg64GiBV1:
			ss = append(ss, 64<<30)
		default:

		}
	}

	for _, ssize := range ss {
		if err := paramfetch.GetParams(ctx, build.ParametersJSON(), build.SrsJSON(), uint64(ssize)); err != nil {
			return xerrors.Errorf("fetching proof parameters: %w", err)
		}
	}

	return nil
}
