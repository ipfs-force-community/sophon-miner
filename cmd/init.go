package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	paramfetch "github.com/filecoin-project/go-paramfetch"
	"github.com/filecoin-project/go-state-types/abi"

	miner0 "github.com/filecoin-project/specs-actors/actors/builtin/miner"

	"github.com/filecoin-project/venus-miner/api"
	"github.com/filecoin-project/venus-miner/build"
	"github.com/filecoin-project/venus-miner/chain/types"
	lcli "github.com/filecoin-project/venus-miner/cli"
	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
	"github.com/filecoin-project/venus-miner/node/repo"
)

var initCmd = &cli.Command{
	Name:  "init",
	Usage: "Initialize a venus miner repo",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "actor",
			Usage: "specify the address of an already created miner actor",
		},
		&cli.StringFlag{
			Name:  "sealer-listen-api",
			Usage: "sealer rpc api",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "sealer-token",
			Usage: "sealer rpc token",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "wallet-listen-api",
			Usage: "wallet rpc api",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "wallet-token",
			Usage: "wallet rpc token",
			Value: "",
		},
	},
	Action: func(cctx *cli.Context) error {
		log.Info("Initializing venus miner")

		ctx := lcli.ReqContext(cctx)

		//log.Info("Checking proof parameters")

		//if err := fetchingProofParameters(ctx); err != nil {
		//	return xerrors.Errorf("fetching proof parameters: %w", err)
		//}

		log.Info("Trying to connect to full node RPC")

		api, closer, err := lcli.GetFullNodeAPI(cctx)
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

		v, err := api.Version(ctx)
		if err != nil {
			return err
		}

		if !v.APIVersion.EqMajorMinor(build.FullAPIVersion) {
			return xerrors.Errorf("Remote API version didn't match (expected %s, remote %s)", build.FullAPIVersion, v.APIVersion)
		}

		log.Info("Initializing repo")

		if err := r.Init(repo.Miner); err != nil {
			return err
		}

		if err := storageMinerInit(cctx, r); err != nil {
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

func storageMinerInit(cctx *cli.Context, r repo.Repo) error {
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

	mds, err := lr.Datastore(context.TODO(), "/metadata")
	if err != nil {
		return err
	}

	var actor address.Address
	if cctx.String("actor") != "" {
		actor, err = address.NewFromString(cctx.String("actor"))
		if err != nil {
			return err
		}

		if actor.Protocol() == address.ID {
			if cctx.String("sealer-listen-api") == "" || cctx.String("sealer-token") == "" {
				return xerrors.New("the actor's api & token cannot be empty")
			}

			posterAddr := dtypes.MinerInfo{
				Addr: actor,
				Sealer: dtypes.SealerNode{
					ListenAPI: cctx.String("sealer-listen-api"),
					Token:     cctx.String("sealer-token"),
				},
			}
			if cctx.String("wallet-listen-api") != "" && cctx.String("wallet-token") != "" {
				posterAddr.Wallet = dtypes.WalletNode{
					ListenAPI: cctx.String("wallet-listen-api"),
					Token:     cctx.String("wallet-token"),
				}
			}

			log.Infof("init new miner: %v", posterAddr)

			miners := make([]dtypes.MinerInfo, 0)
			miners = append(miners, posterAddr)
			addrBytes, err := json.Marshal(miners)
			if err != nil {
				return err
			}
			if err := mds.Put(datastore.NewKey("miner-actors"), addrBytes); err != nil {
				return err
			}
		} else {
			return xerrors.New("the actor's Protocol is not ID")
		}
	}

	return nil
}

func makeHostKey(lr repo.LockedRepo) (crypto.PrivKey, error) { //nolint
	pk, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, err
	}

	ks, err := lr.KeyStore()
	if err != nil {
		return nil, err
	}

	kbytes, err := pk.Bytes()
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

func SyncWait(ctx context.Context, napi api.FullNode, watch bool) error {
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
			case api.StageSyncComplete:
			default:
				working = i
			case api.StageIdle:
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
		if err := paramfetch.GetParams(ctx, build.ParametersJSON(), uint64(ssize)); err != nil {
			return xerrors.Errorf("fetching proof parameters: %w", err)
		}
	}

	return nil
}
