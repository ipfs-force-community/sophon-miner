package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	logging "github.com/ipfs/go-log/v2"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/go-jsonrpc"

	"github.com/ipfs-force-community/sophon-miner/api"
	"github.com/ipfs-force-community/sophon-miner/api/client"
	"github.com/ipfs-force-community/sophon-miner/node/config"
	"github.com/ipfs-force-community/sophon-miner/node/repo"

	v1 "github.com/filecoin-project/venus/venus-shared/api/chain/v1"
)

var log = logging.Logger("cli")

const (
	metadataTraceContext = "traceContext"
)

func GetAPIInfo(ctx *cli.Context) (config.APIInfo, error) {
	p, err := homedir.Expand(ctx.String("repo"))
	if err != nil {
		return config.APIInfo{}, fmt.Errorf("could not expand home dir: %w", err)
	}

	r, err := repo.NewFS(p)
	if err != nil {
		return config.APIInfo{}, fmt.Errorf("could not open repo at path: %s; %w", p, err)
	}

	// todo: rm compatibility for repo when appropriate
	exist, _ := r.Exists()
	if !exist {
		r, err = repo.NewFS("~/.venusminer")
		if err != nil {
			return config.APIInfo{}, err
		}
	}

	ma, err := r.APIEndpoint()
	if err != nil {
		return config.APIInfo{}, fmt.Errorf("could not get api endpoint: %w", err)
	}

	token, err := r.APIToken()
	if err != nil {
		log.Warnf("Couldn't load CLI token, capabilities may be limited: %v", err)
	}

	return config.APIInfo{
		Addr:  ma.String(),
		Token: string(token),
	}, nil
}

func GetRawAPI(ctx *cli.Context, version string) (string, http.Header, error) {
	ainfo, err := GetAPIInfo(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("could not get API info: %w", err)
	}

	addr, err := ainfo.DialArgs(version)
	if err != nil {
		return "", nil, fmt.Errorf("could not get DialArgs: %w", err)
	}

	return addr, ainfo.AuthHeader(), nil
}

func GetMinerAPI(ctx *cli.Context) (api.MinerAPI, jsonrpc.ClientCloser, error) {
	addr, headers, err := GetRawAPI(ctx, "v0")
	if err != nil {
		return nil, nil, err
	}

	return client.NewMinerRPC(ctx.Context, addr, headers)
}

func GetFullNodeAPI(ctx *cli.Context, fn *config.APIInfo, version string) (v1.FullNode, jsonrpc.ClientCloser, error) {
	addr, err := fn.DialArgs(version)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get DialArgs: %w", err)
	}

	return v1.NewFullNodeRPC(ctx.Context, addr, fn.AuthHeader())
}

func DaemonContext(cctx *cli.Context) context.Context {
	if mtCtx, ok := cctx.App.Metadata[metadataTraceContext]; ok {
		return mtCtx.(context.Context)
	}

	return context.Background()
}

// ReqContext returns context for cli execution. Calling it for the first time
// installs SIGTERM handler that will close returned context.
// Not safe for concurrent execution.
func ReqContext(cctx *cli.Context) context.Context {
	tCtx := DaemonContext(cctx)

	ctx, done := context.WithCancel(tCtx)
	sigChan := make(chan os.Signal, 2)
	go func() {
		<-sigChan
		done()
	}()
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

	return ctx
}

var CommonCommands = []*cli.Command{
	logCmd,
	VersionCmd,
}
