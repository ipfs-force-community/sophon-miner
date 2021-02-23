package main

import (
	"encoding/json"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/urfave/cli/v2"

	lcli "github.com/filecoin-project/venus-miner/cli"
	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
)

var addressCmd = &cli.Command{
	Name:  "address",
	Usage: "manage the miner address",
	Subcommands: []*cli.Command{
		addCmd,
		removeCmd,
		listCmd,
		setdefaultCmd,
		defaultCmd,
	},
}

var addCmd = &cli.Command{
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
		&cli.StringFlag{
			Name:     "token",
			Usage:    "rpc token",
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

		posterAddr := dtypes.MinerInfo{
			Addr:      addr,
			ListenAPI: cctx.String("listen-api"),
			Token:     cctx.String("token"),
		}

		err = postApi.AddAddress(posterAddr)
		if err != nil {
			return err
		}
		return nil
	},
}

var removeCmd = &cli.Command{
	Name:      "rm",
	Usage:     "remove the specified miner from the miners",
	ArgsUsage: "[address]",
	Flags:     []cli.Flag{},
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

var listCmd = &cli.Command{
	Name:  "list",
	Usage: "print miners",
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

var setdefaultCmd = &cli.Command{
	Name:      "set-default",
	Usage:     "set default address",
	Flags:     []cli.Flag{},
	ArgsUsage: "[address]",
	Action: func(cctx *cli.Context) error {
		postApi, closer, err := lcli.GetMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		addrStr := cctx.Args().First()
		addr, err := address.NewFromString(addrStr)
		if err != nil {
			return err
		}

		err = postApi.SetDefault(addr)
		if err != nil {
			return err
		}

		return nil
	},
}

var defaultCmd = &cli.Command{
	Name:  "default",
	Usage: "get default address",
	Flags: []cli.Flag{},
	Action: func(cctx *cli.Context) error {
		postApi, closer, err := lcli.GetMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		addr, err := postApi.Default()
		if err != nil {
			return err
		}

		fmt.Println(addr.String())
		return nil
	},
}
