package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	logging "github.com/ipfs/go-log/v2"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/go-jsonrpc"

	"github.com/filecoin-project/venus-miner/api"
	"github.com/filecoin-project/venus-miner/api/client"
	cliutil "github.com/filecoin-project/venus-miner/cli/util"
	"github.com/filecoin-project/venus-miner/node/config"
	"github.com/filecoin-project/venus-miner/node/repo"

	"github.com/filecoin-project/venus/venus-shared/api/chain/v1"
)

var log = logging.Logger("cli")

const (
	metadataTraceContext = "traceContext"
)

// custom CLI error

type ErrCmdFailed struct {
	msg string
}

func (e *ErrCmdFailed) Error() string {
	return e.msg
}

// The flag passed on the command line with the listen address of the API
// server (only used by the tests)
func flagForAPI(t repo.RepoType) string {
	switch t {
	case repo.Miner:
		return "miner-api-url"
	default:
		panic(fmt.Sprintf("Unknown repo type: %v", t))
	}
}

func flagForRepo(t repo.RepoType) string {
	switch t {
	case repo.Miner:
		return "miner-repo"
	default:
		panic(fmt.Sprintf("Unknown repo type: %v", t))
	}
}

func GetAPIInfo(ctx *cli.Context, t repo.RepoType) (cliutil.APIInfo, error) {
	// Check if there was a flag passed with the listen address of the API
	// server (only used by the tests)
	apiFlag := flagForAPI(t)
	if ctx.IsSet(apiFlag) {
		strma := ctx.String(apiFlag)
		strma = strings.TrimSpace(strma)

		return cliutil.APIInfo{Addr: strma}, nil
	}

	repoFlag := flagForRepo(t)

	p, err := homedir.Expand(ctx.String(repoFlag))
	if err != nil {
		return cliutil.APIInfo{}, fmt.Errorf("could not expand home dir (%s): %w", repoFlag, err)
	}

	r, err := repo.NewFS(p)
	if err != nil {
		return cliutil.APIInfo{}, fmt.Errorf("could not open repo at path: %s; %w", p, err)
	}

	ma, err := r.APIEndpoint()
	if err != nil {
		return cliutil.APIInfo{}, fmt.Errorf("could not get api endpoint: %w", err)
	}

	token, err := r.APIToken()
	if err != nil {
		log.Warnf("Couldn't load CLI token, capabilities may be limited: %v", err)
	}

	return cliutil.APIInfo{
		Addr:  ma.String(),
		Token: token,
	}, nil
}

func GetRawAPI(ctx *cli.Context, t repo.RepoType, version string) (string, http.Header, error) {
	ainfo, err := GetAPIInfo(ctx, t)
	if err != nil {
		return "", nil, fmt.Errorf("could not get API info: %w", err)
	}

	addr, err := ainfo.DialArgs(version)
	if err != nil {
		return "", nil, fmt.Errorf("could not get DialArgs: %w", err)
	}

	return addr, ainfo.AuthHeader(), nil
}

func GetFullNodeAPI(ctx *cli.Context, fn config.FullNode, version string) (v1.FullNode, jsonrpc.ClientCloser, error) {
	addr, err := fn.DialArgs(version)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get DialArgs: %w", err)
	}

	return v1.NewFullNodeRPC(ctx.Context, addr, fn.AuthHeader())
}

func GetMinerAPI(ctx *cli.Context) (api.MinerAPI, jsonrpc.ClientCloser, error) {
	addr, headers, err := GetRawAPI(ctx, repo.Miner, "v0")
	if err != nil {
		return nil, nil, err
	}

	return client.NewMinerRPC(ctx.Context, addr, headers)
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
