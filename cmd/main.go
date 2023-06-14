package main

import (
	"context"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/trace"

	"github.com/ipfs-force-community/sophon-miner/build"
	lcli "github.com/ipfs-force-community/sophon-miner/cli"
	"github.com/ipfs-force-community/sophon-miner/lib/logger"
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
	}

	ctx, span := trace.StartSpan(context.Background(), "/cli")
	defer span.End()

	app := &cli.App{
		Name:                 "sophon-miner",
		Usage:                "Filecoin decentralized storage network miner",
		Version:              build.UserVersion(),
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name: FlagMinerRepo,
				EnvVars: []string{
					"SOPHON_MINER_PATH",
					"VENUS_MINER_PATH", // TODO Deprecated after V1.13.*
				},
				Aliases: []string{"miner-repo"},
				Value:   "~/.sophon-miner",
				Usage:   "Specify miner repo path, env SOPHON_MINER_PATH",
			},
		},

		Commands: append(local, lcli.CommonCommands...),
	}
	app.Setup()
	app.Metadata["traceContext"] = ctx

	lcli.RunApp(app)
}
