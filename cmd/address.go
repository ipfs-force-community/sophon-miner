package main

import (
	"encoding/json"
	"fmt"

	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"

	miner0 "github.com/filecoin-project/specs-actors/actors/builtin/miner"

	"github.com/filecoin-project/venus-miner/chain/types"
	lcli "github.com/filecoin-project/venus-miner/cli"
	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
)

func isSupportedSectorSize(ssize abi.SectorSize) bool {
	for spf := range miner0.SupportedProofTypes {
		switch spf {
		case abi.RegisteredSealProof_StackedDrg2KiBV1:
			if ssize == 2048 {
				return true
			}
		case abi.RegisteredSealProof_StackedDrg8MiBV1:
			if ssize == 8<<20 {
				return true
			}
		case abi.RegisteredSealProof_StackedDrg512MiBV1:
			if ssize == 512<<20 {
				return true
			}
		case abi.RegisteredSealProof_StackedDrg32GiBV1:
			if ssize == 32<<30 {
				return true
			}
		case abi.RegisteredSealProof_StackedDrg64GiBV1:
			if ssize == 64<<30 {
				return true
			}
		default:

		}
	}

	return false
}

var addressCmd = &cli.Command{
	Name:  "address",
	Usage: "manage the miner address",
	Subcommands: []*cli.Command{
		addCmd,
		removeCmd,
		listCmd,
		setdefaultCmd,
		defaultCmd,
		startMiningCmd,
		stopMiningCmd,
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
		nodeApi, ncloser, err := lcli.GetFullNodeAPI(cctx)
		if err != nil {
			return xerrors.Errorf("getting full node api: %w", err)
		}
		defer ncloser()

		// check actor
		addrStr := cctx.String("addr")
		addr, err := address.NewFromString(addrStr)
		if err != nil {
			return err
		}

		ctx := lcli.DaemonContext(cctx)
		mi, err := nodeApi.StateMinerInfo(ctx, addr, types.EmptyTSK)
		if err != nil {
			return xerrors.Errorf("looking up actor: %w", err)
		}

		if !isSupportedSectorSize(mi.SectorSize) {
			return xerrors.New("Sector-Size not supported")
		}

		postApi, closer, err := lcli.GetMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		posterAddr := dtypes.MinerInfo{
			Addr:      addr,
			ListenAPI: cctx.String("listen-api"),
			Token:     cctx.String("token"),
		}

		err = postApi.AddAddress(posterAddr)
		if err != nil {
			return err
		}

		fmt.Println("add miner: ", posterAddr)
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

		fmt.Println("remove miner: ", minerAddr)
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

		fmt.Println("default miner: ", addr)
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

		fmt.Println("default miner: ", addr)
		return nil
	},
}

var startMiningCmd = &cli.Command{
	Name:      "start",
	Usage:     "start mining for specified miner",
	Flags:     []cli.Flag{},
	ArgsUsage: "[address]",
	Action: func(cctx *cli.Context) error {
		postApi, closer, err := lcli.GetMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := lcli.ReqContext(cctx)

		addrStr := cctx.Args().First()
		addr, err := address.NewFromString(addrStr)
		if err != nil {
			return err
		}

		err = postApi.Start(ctx, addr)
		if err != nil {
			return err
		}

		fmt.Println("start mining for: ", addr)
		return nil
	},
}

var stopMiningCmd = &cli.Command{
	Name:      "stop",
	Usage:     "stop mining for specified miner",
	Flags:     []cli.Flag{},
	ArgsUsage: "[address]",
	Action: func(cctx *cli.Context) error {
		postApi, closer, err := lcli.GetMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := lcli.ReqContext(cctx)

		addrStr := cctx.Args().First()
		addr, err := address.NewFromString(addrStr)
		if err != nil {
			return err
		}

		err = postApi.Stop(ctx, addr)
		if err != nil {
			return err
		}

		fmt.Println("stop mining for: ", addr)
		return nil
	},
}
