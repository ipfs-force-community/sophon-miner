package main

import (
	_ "net/http/pprof"

	"github.com/urfave/cli/v2"

	lcli "github.com/filecoin-project/venus-miner/cli"
)

var stopCmd = &cli.Command{
	Name:  "stop",
	Usage: "Stop a running venus miner",
	Flags: []cli.Flag{},
	Action: func(cctx *cli.Context) error {
		api, closer, err := lcli.GetMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		err = api.Shutdown(lcli.ReqContext(cctx))
		if err != nil {
			return err
		}

		return nil
	},
}
