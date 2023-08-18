package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	lcli "github.com/ipfs-force-community/sophon-miner/cli"
	"github.com/ipfs-force-community/sophon-miner/types"
	"github.com/urfave/cli/v2"
)

var recordCmd = &cli.Command{
	Name:  "record",
	Usage: "the record about mine block",
	Subcommands: []*cli.Command{
		queryCmd,
	},
}

var queryCmd = &cli.Command{
	Name:      "query",
	Usage:     "query record",
	ArgsUsage: "<miner address> <epoch>",
	Flags: []cli.Flag{
		&cli.UintFlag{
			Name:  "limit",
			Usage: "query nums record of limit",
			Value: 1,
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.NArg() != 2 {
			return cli.ShowSubcommandHelp(cctx)
		}

		minerAddr, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return err
		}

		epoch, err := strconv.ParseUint(cctx.Args().Get(1), 10, 64)
		if err != nil {
			return err
		}

		minerAPI, closer, err := lcli.GetMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		res, err := minerAPI.QueryRecord(lcli.ReqContext(cctx), &types.QueryRecordParams{
			Epoch: abi.ChainEpoch(epoch),
			Limit: cctx.Uint("limit"),
			Miner: minerAddr,
		})
		if err != nil {
			return err
		}

		formatJson, err := json.MarshalIndent(res, "", "\t")
		if err != nil {
			return err
		}

		fmt.Println(string(formatJson))
		return nil
	},
}
