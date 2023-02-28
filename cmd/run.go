package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/gorilla/mux"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/venus-auth/jwtclient"

	lapi "github.com/filecoin-project/venus-miner/api"
	lcli "github.com/filecoin-project/venus-miner/cli"
	"github.com/filecoin-project/venus-miner/lib/metrics"
	"github.com/filecoin-project/venus-miner/lib/tracing"
	"github.com/filecoin-project/venus-miner/node"
	"github.com/filecoin-project/venus-miner/node/config"
	"github.com/filecoin-project/venus-miner/node/repo"
	"github.com/filecoin-project/venus-miner/types"

	"github.com/filecoin-project/venus/pkg/constants"
	"github.com/filecoin-project/venus/venus-shared/api/chain"
	v1 "github.com/filecoin-project/venus/venus-shared/api/chain/v1"
	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
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

		lr, err := r.Lock()
		if err != nil {
			return err
		}
		err = lr.Migrate() //nolint: errcheck
		if err != nil {
			log.Errorf("Migrate failed: %v", err.Error())
		}
		cfgV, err := lr.Config()
		if err != nil {
			return err
		}
		cfg := cfgV.(*config.MinerConfig)

		nodeApi, ncloser, err := lcli.GetFullNodeAPI(cctx, cfg.FullNode, "v1")
		if err != nil {
			return err
		}

		localJwtClient, token, err := jwtclient.NewLocalAuthClient()
		if err != nil {
			return fmt.Errorf("unable to generate local jwt client: %w", err)
		}

		err = lr.SetAPIToken(token)
		if err != nil {
			return err
		}

		lr.Close() //nolint:errcheck

		if err != nil {
			return fmt.Errorf("getting full node api: %w", err)
		}
		defer ncloser()

		var remoteJwtAuthClient jwtclient.IJwtAuthClient
		var authClient jwtclient.IAuthClient
		if len(cfg.Auth.Addr) == 0 {
			return fmt.Errorf("auth addr is empty")
		}
		client, err := jwtclient.NewAuthClient(cfg.Auth.Addr)
		if err != nil {
			return fmt.Errorf("failed to create remote jwt auth client: %w", err)
		}
		remoteJwtAuthClient = jwtclient.WarpIJwtAuthClient(client)
		authClient = client

		// TODO: delete this when relative issue is fixed in lotus https://github.com/filecoin-project/venus/issues/5247
		log.Info("wait for height of chain bigger than zero ...")
		ticker := time.NewTicker(10 * time.Second)
		for {
			head, err := nodeApi.ChainHead(ctx)
			if err != nil {
				return err
			}
			if head.Height() > 0 {
				break
			}
			select {
			case <-ctx.Done():
				fmt.Println("\nExit by user")
				return nil
			case <-ticker.C:
			}
		}
		ticker.Stop()

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

		if v.APIVersion != chain.FullAPIVersion1 {
			return fmt.Errorf("venus-daemon API version doesn't match: expected: %s", sharedTypes.Version{APIVersion: chain.FullAPIVersion1})
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
			node.Repo(cctx, r),
			node.Override(new(types.ShutdownChan), shutdownChan),
			node.Override(new(jwtclient.IAuthClient), authClient),

			node.ApplyIf(func(s *node.Settings) bool { return cctx.IsSet("miner-api") },
				node.Override(new(types.APIEndpoint), func() (types.APIEndpoint, error) {
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

		// metrics
		err = metrics.SetupMetrics(ctx, cfg.Metrics)
		if err != nil {
			return err
		}

		var remoteJwtAuthClient jwtclient.IJwtAuthClient
		if len(cfg.Auth.Addr) > 0 {
			client, err := jwtclient.NewAuthClient(cfg.Auth.Addr, cfg.Auth.Token)
			if err != nil {
				return fmt.Errorf("failed to create remote jwt auth client: %w", err)
			}
			remoteJwtAuthClient = jwtclient.WarpIJwtAuthClient(client)
		}

		return serveRPC(minerAPI, stop, endpoint, shutdownChan, int64(cctx.Int("api-max-req-size")), localJwtClient, remoteJwtAuthClient)
	},
}

func serveRPC(minerAPI lapi.MinerAPI, stop node.StopFunc, addr multiaddr.Multiaddr, shutdownChan chan struct{}, maxRequestSize int64, localJwtClient, remoteJwtAuthClient jwtclient.IJwtAuthClient) error {
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
	mux.PathPrefix("/").Handler(http.DefaultServeMux) // pprof

	ah := jwtclient.NewAuthMux(localJwtClient, remoteJwtAuthClient, mux)

	srv := &http.Server{
		Handler: ah,
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
