package cli

import (
	"context"
	"fmt"
	"github.com/filecoin-project/venus-miner/api/v1api"
	"github.com/filecoin-project/venus-miner/node/config"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	logging "github.com/ipfs/go-log/v2"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-jsonrpc"

	"github.com/filecoin-project/venus-miner/api"
	"github.com/filecoin-project/venus-miner/api/client"
	cliutil "github.com/filecoin-project/venus-miner/cli/util"
	"github.com/filecoin-project/venus-miner/node/repo"
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

func NewCliError(s string) error {
	return &ErrCmdFailed{s}
}

// ApiConnector returns API instance
type ApiConnector func() api.FullNode

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
		return cliutil.APIInfo{}, xerrors.Errorf("could not expand home dir (%s): %w", repoFlag, err)
	}

	r, err := repo.NewFS(p)
	if err != nil {
		return cliutil.APIInfo{}, xerrors.Errorf("could not open repo at path: %s; %w", p, err)
	}

	ma, err := r.APIEndpoint()
	if err != nil {
		return cliutil.APIInfo{}, xerrors.Errorf("could not get api endpoint: %w", err)
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
		return "", nil, xerrors.Errorf("could not get API info: %w", err)
	}

	addr, err := ainfo.DialArgs(version)
	if err != nil {
		return "", nil, xerrors.Errorf("could not get DialArgs: %w", err)
	}

	return addr, ainfo.AuthHeader(), nil
}

func GetAPI(ctx *cli.Context) (api.Common, jsonrpc.ClientCloser, error) {
	ti, ok := ctx.App.Metadata["repoType"]
	if !ok {
		log.Errorf("unknown repo type, are you sure you want to use GetAPI?")
		ti = repo.Miner
	}
	t, ok := ti.(repo.RepoType)
	if !ok {
		log.Errorf("repoType type does not match the type of repo.RepoType")
	}

	if tn, ok := ctx.App.Metadata["testnode-full"]; ok {
		return tn.(api.FullNode), func() {}, nil
	}

	addr, headers, err := GetRawAPI(ctx, t, "v0")
	if err != nil {
		return nil, nil, err
	}

	return client.NewCommonRPCV0(ctx.Context, addr, headers)
}

func GetFullNodeAPI(ctx *cli.Context, fn config.FullNode) (api.FullNode, jsonrpc.ClientCloser, error) {
	addr, err := fn.DialArgs("v0")
	if err != nil {
		return nil, nil, xerrors.Errorf("could not get DialArgs: %w", err)
	}

	return client.NewFullNodeRPCV0(ctx.Context, addr, fn.AuthHeader())
}

func GetFullNodeAPIV1(ctx *cli.Context, fn config.FullNode) (v1api.FullNode, jsonrpc.ClientCloser, error) {
	addr, err := fn.DialArgs("v1")
	if err != nil {
		return nil, nil, xerrors.Errorf("could not get DialArgs: %w", err)
	}

	return client.NewFullNodeRPCV1(ctx.Context, addr, fn.AuthHeader())
}

func GetMinerAPI(ctx *cli.Context) (api.MinerAPI, jsonrpc.ClientCloser, error) {
	addr, headers, err := GetRawAPI(ctx, repo.Miner, "v0")
	if err != nil {
		return nil, nil, err
	}

	return client.NewMinerRPCV0(ctx.Context, addr, headers)
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
}

var Commands = []*cli.Command{
	WithCategory("developer", logCmd),

	VersionCmd,
}

func WithCategory(cat string, cmd *cli.Command) *cli.Command {
	cmd.Category = cat
	return cmd
}
