package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	v1 "github.com/filecoin-project/venus/venus-shared/api/chain/v1"
	lcli "github.com/ipfs-force-community/sophon-miner/cli"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
)

var log = logging.Logger("main")

func main() {
	logging.SetAllLoggers(logging.LevelInfo)
	app := &cli.App{
		Name:                 "",
		Usage:                "",
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:     "url-token",
				Usage:    "venus url and token, eg token:url",
				Required: true,
			},
			&cli.Int64Flag{
				Name:  "interval",
				Value: 5,
			},
		},
		Action: run,
	}
	app.Setup()

	lcli.RunApp(app)
}

func run(cctx *cli.Context) error {
	urlTokens := cctx.StringSlice("url-token")
	log.Info("url-tokens: ", urlTokens)

	interval := cctx.Int64("interval")
	log.Info("interval: ", interval)

	ticker := time.NewTicker(time.Second * time.Duration(interval))
	defer ticker.Stop()

	ctx, cancel := context.WithCancel(cctx.Context)
	defer cancel()

	var apis []v1.FullNode

	for _, urlToken := range urlTokens {
		arr := strings.Split(urlToken, ":")
		if len(arr) != 2 {
			log.Info("invalid url and token: ", arr)
			continue
		}
		api, closer, err := v1.DialFullNodeRPC(ctx, arr[1], arr[0], nil)
		if err != nil {
			return err
		}
		defer closer()

		apis = append(apis, api)
	}

	for {
		select {
		case <-ctx.Done():
			fmt.Println("context done")
			return ctx.Err()
		case <-ticker.C:
			var wg sync.WaitGroup
			for _, api := range apis {
				api := api
				wg.Add(1)
				go func() {
					defer wg.Done()

					head, err := api.ChainHead(ctx)
					if err != nil {
						log.Info("got head failed: ", err)
						return
					}
					start := time.Now()
					round := head.Height() + 1
					if _, err = api.StateGetBeaconEntry(ctx, round); err != nil {
						log.Info("call StateGetBeaconEntry failed: ", err)
					}

					log.Infof("call StateGetBeaconEntry at %d took: %v", round, time.Since(start))
				}()
			}
			wg.Wait()
		}
	}
}
