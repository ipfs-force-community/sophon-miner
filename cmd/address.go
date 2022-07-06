package main

import (
	"encoding/json"
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/go-address"
	lcli "github.com/filecoin-project/venus-miner/cli"
)

var addressCmd = &cli.Command{
	Name:  "address",
	Usage: "manage the miner address",
	Subcommands: []*cli.Command{
		updateCmd,
		listCmd,
		stateCmd,
		startMiningCmd,
		stopMiningCmd,
		warmupCmd,
	},
}

var updateCmd = &cli.Command{
	Name:  "update",
	Usage: "reacquire address from venus-auth",
	Flags: []cli.Flag{
		&cli.Int64Flag{
			Name:     "skip",
			Required: false,
		},
		&cli.Int64Flag{
			Name:     "limit",
			Required: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		postApi, closer, err := lcli.GetMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		skip := cctx.Int64("skip")
		limit := cctx.Int64("limit")

		miners, err := postApi.UpdateAddress(cctx.Context, skip, limit)
		if err != nil {
			return err
		}

		formatJson, err := json.MarshalIndent(miners, "", "\t")
		if err != nil {
			return err
		}
		fmt.Println(string(formatJson))

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

		miners, err := postApi.ListAddress(cctx.Context)
		if err != nil {
			return err
		}

		formatJson, err := json.MarshalIndent(miners, "", "\t")
		if err != nil {
			return err
		}
		fmt.Println(string(formatJson))
		return nil

	},
}

var stateCmd = &cli.Command{
	Name:      "state",
	Usage:     "print state of mining",
	Flags:     []cli.Flag{},
	ArgsUsage: "[address ...]",
	Action: func(cctx *cli.Context) error {
		postApi, closer, err := lcli.GetMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		var addrs []address.Address
		for i, s := range cctx.Args().Slice() {
			minerAddr, err := address.NewFromString(s)
			if err != nil {
				return fmt.Errorf("parsing %d-th miner: %w", i, err)
			}

			addrs = append(addrs, minerAddr)
		}

		states, err := postApi.StatesForMining(cctx.Context, addrs)
		if err != nil {
			return err
		}

		formatJson, err := json.MarshalIndent(states, "", "\t")
		if err != nil {
			return err
		}
		fmt.Println(string(formatJson))
		return nil

	},
}

var startMiningCmd = &cli.Command{
	Name:      "start",
	Usage:     "start mining for specified miner, if not specified, it means all",
	Flags:     []cli.Flag{},
	ArgsUsage: "[address ...]",
	Action: func(cctx *cli.Context) error {
		postApi, closer, err := lcli.GetMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := lcli.ReqContext(cctx)

		var addrs []address.Address
		for i, s := range cctx.Args().Slice() {
			minerAddr, err := address.NewFromString(s)
			if err != nil {
				return fmt.Errorf("parsing %d-th miner: %w", i, err)
			}

			addrs = append(addrs, minerAddr)
		}

		err = postApi.Start(ctx, addrs)
		if err != nil {
			return err
		}

		fmt.Println("start mining success.")
		return nil
	},
}

var stopMiningCmd = &cli.Command{
	Name:      "stop",
	Usage:     "stop mining for specified miner, if not specified, it means all",
	Flags:     []cli.Flag{},
	ArgsUsage: "[address ...]",
	Action: func(cctx *cli.Context) error {
		postApi, closer, err := lcli.GetMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := lcli.ReqContext(cctx)

		var addrs []address.Address
		for i, s := range cctx.Args().Slice() {
			minerAddr, err := address.NewFromString(s)
			if err != nil {
				return fmt.Errorf("parsing %d-th miner: %w", i, err)
			}

			addrs = append(addrs, minerAddr)
		}

		err = postApi.Stop(ctx, addrs)
		if err != nil {
			return err
		}

		fmt.Println("stop mining success.")
		return nil
	},
}

var warmupCmd = &cli.Command{
	Name:      "warmup",
	Usage:     "winPoSt warmup for miner",
	Flags:     []cli.Flag{},
	ArgsUsage: "<miner address>",
	Action: func(cctx *cli.Context) error {
		if count := cctx.Args().Len(); count < 1 {
			return cli.ShowSubcommandHelp(cctx)
		}

		minerApi, closer, err := lcli.GetMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		minerAddr, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return fmt.Errorf("parsing miner: %w", err)
		}

		err = minerApi.WarmupForMiner(cctx.Context, minerAddr)
		if err == nil {
			fmt.Println("warmup success.")
		} else {
			fmt.Println("warmup failed: ", err.Error())
		}

		return nil
	},
}
