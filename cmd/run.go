package main

import (
	"context"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/mux"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/tag"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-jsonrpc/auth"

	lapi "github.com/filecoin-project/venus-miner/api"
	"github.com/filecoin-project/venus-miner/build"
	lcli "github.com/filecoin-project/venus-miner/cli"
	"github.com/filecoin-project/venus-miner/lib/tracing"
	"github.com/filecoin-project/venus-miner/metrics"
	"github.com/filecoin-project/venus-miner/node"
	"github.com/filecoin-project/venus-miner/node/config"
	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
	"github.com/filecoin-project/venus-miner/node/repo"

	"github.com/filecoin-project/venus/venus-shared/api"
	"github.com/filecoin-project/venus/venus-shared/api/chain/v1"
)

var runCmd = &cli.Command{
	Name:  "run",
	Usage: "Start a venus miner process",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "miner-api",
			Usage: "12308",
		},
		&cli.StringFlag{
			Name:        "nettype",
			Usage:       "network type, one of: mainnet, debug, 2k, calibnet",
			Value:       "mainnet",
			DefaultText: "mainnet",
			Required:    false,
		},
		&cli.BoolFlag{
			Name:  "enable-gpu-proving",
			Usage: "enable use of GPU for mining operations",
			Value: true,
		},
		&cli.BoolFlag{
			Name:  "nosync",
			Usage: "don't check full-node sync status",
		},
		&cli.IntFlag{
			Name:  "api-max-req-size",
			Usage: "maximum API request size accepted by the JSON RPC server",
		},
		node.CLIFLAGBlockRecord,
	},
	Action: func(cctx *cli.Context) error {
		log.Info("Initializing build params")

		ctx := lcli.ReqContext(cctx)

		if err := build.InitNetWorkParams(cctx.String("nettype")); err != nil {
			return err
		}

		// default enlarge max os threads to 20000
		//maxOSThreads := 20000
		//if fMaxOSThreads := os.Getenv("FORCE_MAX_OS_THREADS"); fMaxOSThreads != "" {
		//	var err error
		//	maxOSThreads, err = strconv.Atoi(fMaxOSThreads)
		//	if err != nil {
		//		return err
		//	}
		//}
		//debug.SetMaxThreads(maxOSThreads)

		if !cctx.Bool("enable-gpu-proving") {
			err := os.Setenv("BELLMAN_NO_GPU", "true")
			if err != nil {
				return err
			}
		}

		minerRepoPath := cctx.String(FlagMinerRepo)
		r, err := repo.NewFS(minerRepoPath)
		if err != nil {
			return err
		}

		ok, err := r.Exists()
		if err != nil {
			return err
		}
		if !ok {
			return xerrors.Errorf("repo at '%s' is not initialized, run 'venus-miner init' to set it up", minerRepoPath)
		}

		//log.Info("Checking proof parameters")
		//
		//if err := fetchingProofParameters(ctx); err != nil {
		//	return xerrors.Errorf("fetching proof parameters: %w", err)
		//}

		lr, err := r.Lock(repo.Miner)
		if err != nil {
			return err
		}
		cfgV, err := lr.Config()
		if err != nil {
			return err
		}
		cfg := cfgV.(*config.MinerConfig)

		if err := checkV1ApiSupport(ctx, cctx, cfg.FullNode); err != nil {
			return err
		}

		nodeApi, ncloser, err := lcli.GetFullNodeAPIV1(cctx, cfg.FullNode)
		lr.Close() //nolint:errcheck
		if err != nil {
			return xerrors.Errorf("getting full node api: %w", err)
		}
		defer ncloser()

		v, err := nodeApi.Version(ctx)
		if err != nil {
			return err
		}

		if v.APIVersion != api.FullAPIVersion1 {
			return xerrors.Errorf("venus-daemon API version doesn't match: expected: %s", lapi.APIVersion{APIVersion: api.FullAPIVersion1})
		}

		log.Info("Checking full node sync status")

		if !cctx.Bool("nosync") {
			if err := SyncWait(ctx, nodeApi, false); err != nil {
				return xerrors.Errorf("sync wait: %w", err)
			}
		}

		shutdownChan := make(chan struct{})

		var minerAPI lapi.MinerAPI
		stop, err := node.New(ctx,
			node.MinerAPI(&minerAPI),
			node.Override(new(dtypes.ShutdownChan), shutdownChan),
			node.Online(),
			node.Repo(cctx, r),

			node.ApplyIf(func(s *node.Settings) bool { return cctx.IsSet("miner-api") },
				node.Override(new(dtypes.APIEndpoint), func() (dtypes.APIEndpoint, error) {
					return multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/" + cctx.String("miner-api"))
				})),
			node.Override(new(v1.FullNode), nodeApi),
		)
		if err != nil {
			return xerrors.Errorf("creating node: %w", err)
		}

		endpoint, err := r.APIEndpoint()
		if err != nil {
			return xerrors.Errorf("getting API endpoint: %w", err)
		}

		//// Bootstrap with full node
		//remoteAddrs, err := nodeApi.NetAddrsListen(ctx)
		//if err != nil {
		//	return xerrors.Errorf("getting full node libp2p address: %w", err)
		//}
		//
		//if err := minerAPI.NetConnect(ctx, remoteAddrs); err != nil {
		//	return xerrors.Errorf("connecting to full node (libp2p): %w", err)
		//}

		log.Infof("Remote version %s", v)

		// setup jaeger tracing
		jaeger := tracing.SetupJaegerTracing(cfg.Tracing)
		defer func() {
			if jaeger != nil {
				jaeger.Flush()
			}
		}()

		return serveRPC(minerAPI, stop, endpoint, shutdownChan, int64(cctx.Int("api-max-req-size")))
	},
}

func serveRPC(minerAPI lapi.MinerAPI, stop node.StopFunc, addr multiaddr.Multiaddr, shutdownChan chan struct{}, maxRequestSize int64) error {
	lst, err := manet.Listen(addr)
	if err != nil {
		return xerrors.Errorf("could not listen: %w", err)
	}

	serverOptions := make([]jsonrpc.ServerOption, 0)
	if maxRequestSize != 0 { // config set
		serverOptions = append(serverOptions, jsonrpc.WithMaxRequestSize(maxRequestSize))
	}

	rpcServer := jsonrpc.NewServer(serverOptions...)
	rpcServer.Register("Filecoin", lapi.PermissionedMinerAPI(minerAPI))
	// rpcServer.Register("Filecoin", minerAPI)

	mux := mux.NewRouter()
	mux.Handle("/rpc/v0", rpcServer)
	mux.Handle("/debug/metrics", metrics.Exporter())
	mux.PathPrefix("/").Handler(http.DefaultServeMux) // pprof

	ah := &auth.Handler{
		Verify: minerAPI.AuthVerify,
		Next:   mux.ServeHTTP,
	}

	srv := &http.Server{
		Handler: ah,
		BaseContext: func(listener net.Listener) context.Context {
			ctx, _ := tag.New(context.Background(), tag.Upsert(metrics.APIInterface, "venus-miner"))
			return ctx
		},
	}

	sigChan := make(chan os.Signal, 2)
	go func() {
		select {
		case sig := <-sigChan:
			log.Warnw("received shutdown", "signal", sig)
		case <-shutdownChan:
			log.Warn("received shutdown")
		}

		log.Warn("Shutting down...")
		if err := stop(context.TODO()); err != nil {
			log.Errorf("graceful shutting down failed: %s", err)
		}
		if err := srv.Shutdown(context.TODO()); err != nil {
			log.Errorf("shutting down RPC server failed: %s", err)
		}
		log.Warn("Graceful shutdown successful")
	}()
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	return srv.Serve(manet.NetListener(lst))
}
