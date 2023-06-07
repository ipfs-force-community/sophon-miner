package main

import (
	_ "net/http/pprof"

	"github.com/urfave/cli/v2"

	lcli "github.com/ipfs-force-community/sophon-miner/cli"
)

var stopCmd = &cli.Command{
	Name:  "stop",
	Usage: "Stop running sophon-miner daemon",
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
