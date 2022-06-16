package main

import (
	"context"
	"fmt"
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

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-jsonrpc/auth"

	lapi "github.com/filecoin-project/venus-miner/api"
	lcli "github.com/filecoin-project/venus-miner/cli"
	"github.com/filecoin-project/venus-miner/lib/metrics"
	"github.com/filecoin-project/venus-miner/lib/tracing"
	"github.com/filecoin-project/venus-miner/node"
	"github.com/filecoin-project/venus-miner/node/config"
	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
	"github.com/filecoin-project/venus-miner/node/repo"

	"github.com/filecoin-project/venus/pkg/constants"
	"github.com/filecoin-project/venus/venus-shared/api"
	v1 "github.com/filecoin-project/venus/venus-shared/api/chain/v1"
)

var runCmd = &cli.Command{
	Name:  "run",
	Usage: "Start a venus miner process",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "miner-api",
			Usage: "12308",
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
			return fmt.Errorf("repo at '%s' is not initialized, run 'venus-miner init' to set it up", minerRepoPath)
		}

		lr, err := r.Lock(repo.Miner)
		if err != nil {
			return err
		}
		cfgV, err := lr.Config()
		if err != nil {
			return err
		}
		cfg := cfgV.(*config.MinerConfig)

		nodeApi, ncloser, err := lcli.GetFullNodeAPI(cctx, cfg.FullNode, "v1")
		lr.Close() //nolint:errcheck
		if err != nil {
			return fmt.Errorf("getting full node api: %w", err)
		}
		defer ncloser()

		netName, err := nodeApi.StateNetworkName(ctx)
		if err != nil {
			return err
		}
		if netName == "mainnet" {
			constants.SetAddressNetwork(address.Mainnet)
		}

		v, err := nodeApi.Version(ctx)
		if err != nil {
			return err
		}

		if v.APIVersion != api.FullAPIVersion1 {
			return fmt.Errorf("venus-daemon API version doesn't match: expected: %s", lapi.APIVersion{APIVersion: api.FullAPIVersion1})
		}

		log.Info("Checking full node sync status")

		if !cctx.Bool("nosync") {
			if err := SyncWait(ctx, nodeApi, false); err != nil {
				return fmt.Errorf("sync wait: %w", err)
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
			return fmt.Errorf("creating node: %w", err)
		}

		endpoint, err := r.APIEndpoint()
		if err != nil {
			return fmt.Errorf("getting API endpoint: %w", err)
		}

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
		return fmt.Errorf("could not listen: %w", err)
	}

	serverOptions := make([]jsonrpc.ServerOption, 0)
	if maxRequestSize != 0 { // config set
		serverOptions = append(serverOptions, jsonrpc.WithMaxRequestSize(maxRequestSize))
	}

	rpcServer := jsonrpc.NewServer(serverOptions...)
	rpcServer.Register("Filecoin", lapi.PermissionedMinerAPI(minerAPI))

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
