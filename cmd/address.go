package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/urfave/cli/v2"

	lcli "github.com/filecoin-project/venus-miner/cli"
	"github.com/filecoin-project/venus-miner/node/config"
)

var addressCmd = &cli.Command{
	Name:  "address",
	Usage: "manage the miner address",
	Subcommands: []*cli.Command{
		addAddrCmd,
		removeAddrCmd,
		listAddrCmd,
		setdefaultCmd,
	},
}

var setdefaultCmd = &cli.Command{
	Name:  "set-default",
	Usage: "set default address for poster",
	Flags: []cli.Flag{},
	Action: func(cctx *cli.Context) error {
		postApi, closer, err := lcli.GetMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		addrStr := cctx.Args().Get(0)
		addr, err := address.NewFromString(addrStr)
		if err != nil {
			return err
		}
		err = postApi.SetDefault(context.TODO(), addr)
		if err != nil {
			return err
		}
		return nil
	},
}

var addAddrCmd = &cli.Command{
	Name:  "add",
	Usage: "add address for poster",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "addr",
			Usage:    "miner address",
			Value:    "",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "listen-api",
			Usage:    "rpc api",
			Value:    "",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		postApi, closer, err := lcli.GetMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		addrStr := cctx.String("addr")
		addr, err := address.NewFromString(addrStr)
		if err != nil {
			return err
		}

		posterAddr := config.MinerInfo{
			Addr:      addr,
			ListenAPI: cctx.String("listen-api"),
		}

		err = postApi.AddAddress(posterAddr)
		if err != nil {
			return err
		}
		return nil
	},
}

var removeAddrCmd = &cli.Command{
	Name:  "rm",
	Usage: "remove addr to poster",
	Flags: []cli.Flag{},
	Action: func(cctx *cli.Context) error {
		postApi, closer, err := lcli.GetMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		minerAddr, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return err
		}
		err = postApi.RemoveAddress(minerAddr)
		if err != nil {
			return err
		}
		return nil
	},
}

var listAddrCmd = &cli.Command{
	Name:  "list",
	Usage: "addrs to poster",
	Flags: []cli.Flag{},
	Action: func(cctx *cli.Context) error {

		postApi, closer, err := lcli.GetMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		addrs, err := postApi.ListAddress()
		if err != nil {
			return err
		}

		formatJson, err := json.MarshalIndent(addrs, "", "\t")
		if err != nil {
			return err
		}
		fmt.Println(string(formatJson))
		return nil

	},
}
