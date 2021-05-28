package main

import (
	"encoding/json"
	"fmt"

	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"

	lcli "github.com/filecoin-project/venus-miner/cli"
)

var winnerCmd = &cli.Command{
	Name:  "winner",
	Usage: "block right management",
	Subcommands: []*cli.Command{
		countCmd,
	},
}

var countCmd = &cli.Command{
	Name:  "count",
	Usage: "Count the block rights of the specified miner",
	Flags: []cli.Flag{
		&cli.Int64Flag{
			Name:     "epoch-start",
			Required: true,
		},
		&cli.Int64Flag{
			Name:     "epoch-end",
			Required: true,
		},
	},
	ArgsUsage: "[address ...]",
	Action: func(cctx *cli.Context) error {
		minerAPI, closer, err := lcli.GetMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		var addrs []address.Address
		for i, s := range cctx.Args().Slice() {
			minerAddr, err := address.NewFromString(s)
			if err != nil {
				return xerrors.Errorf("parsing %d-th miner: %w", i, err)
			}

			addrs = append(addrs, minerAddr)
		}

		epochStart := cctx.Int64("epoch-start")
		epochEnd := cctx.Int64("epoch-end")

		winners, err := minerAPI.CountWinners(addrs, abi.ChainEpoch(epochStart), abi.ChainEpoch(epochEnd))
		if err != nil {
			return err
		}

		formatJson, err := json.MarshalIndent(winners, "", "\t")
		if err != nil {
			return err
		}
		fmt.Println(string(formatJson))
		return nil

	},
}
