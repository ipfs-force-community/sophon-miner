package main

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/venus-miner/node/config"
)

var configCmd = &cli.Command{
	Name:  "config",
	Usage: "Output default configuration",
	Action: func(cctx *cli.Context) error {
		comm, err := config.ConfigComment(config.DefaultMinerConfig())
		if err != nil {
			return err
		}
		fmt.Println(string(comm))
		return nil
	},
}
