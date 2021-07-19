package main

import (
	"fmt"

	"github.com/filecoin-project/venus-miner/api"
	"github.com/filecoin-project/venus-miner/build"
	lcli "github.com/filecoin-project/venus-miner/cli"
	"github.com/filecoin-project/venus-miner/lib/venuslog"
	"github.com/filecoin-project/venus-miner/node/repo"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
)

var log = logging.Logger("main")

const FlagMinerRepo = "miner-repo"

// TODO remove after deprecation period
const FlagMinerRepoDeprecation = "storagerepo"

func main() {
	api.RunningNodeType = api.NodeMiner

	venuslog.SetupLogLevels()

	local := []*cli.Command{
		initCmd,
		runCmd,
		stopCmd,
		addressCmd,
		winnerCmd,
		configCmd,
	}

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
				Aliases: []string{FlagMinerRepoDeprecation},
				EnvVars: []string{"VENUS_MINER_PATH"},
				Value:   "~/.venusminer", // TODO: Consider XDG_DATA_HOME
				Usage:   fmt.Sprintf("Specify miner repo path. flag(%s) and env(VENUS_MINER_PATH) are DEPRECATION, will REMOVE SOON", FlagMinerRepoDeprecation),
			},
		},

		Commands: append(local, lcli.CommonCommands...),
	}
	app.Setup()
	app.Metadata["repoType"] = repo.Miner

	lcli.RunApp(app)
}
