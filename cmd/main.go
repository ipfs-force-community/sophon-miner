package main

import (
	"context"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/trace"

	"github.com/filecoin-project/venus-miner/build"
	lcli "github.com/filecoin-project/venus-miner/cli"
	"github.com/filecoin-project/venus-miner/lib/logger"
)

var log = logging.Logger("main")

const FlagMinerRepo = "repo"

func main() {
	logger.SetupLogLevels()

	local := []*cli.Command{
		initCmd,
		runCmd,
		stopCmd,
		addressCmd,
		winnerCmd,
		configCmd,
	}

	ctx, span := trace.StartSpan(context.Background(), "/cli")
	defer span.End()

	app := &cli.App{
		Name:                 "venus-miner",
		Usage:                "Filecoin decentralized storage network miner",
		Version:              build.UserVersion(),
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name: "color",
			},
			&cli.StringFlag{
				Name:    FlagMinerRepo,
				EnvVars: []string{"VENUS_MINER_PATH"},
				Aliases: []string{"miner-repo"},
				Value:   "~/.venusminer",
				Usage:   "Specify miner repo path, env VENUS_MINER_PATH",
			},
		},

		Commands: append(local, lcli.CommonCommands...),
	}
	app.Setup()
	app.Metadata["traceContext"] = ctx

	lcli.RunApp(app)
}
