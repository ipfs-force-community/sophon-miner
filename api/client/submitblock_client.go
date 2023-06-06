package client

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-jsonrpc"

	"github.com/ipfs-force-community/sophon-miner/node/config"

	v1 "github.com/filecoin-project/venus/venus-shared/api/chain/v1"
)

func NewSubmitBlockRPC(ctx context.Context, apiInfo *config.APIInfo) (v1.FullNode, jsonrpc.ClientCloser, error) {
	addr, err := apiInfo.DialArgs("v1")
	if err != nil {
		return nil, nil, fmt.Errorf("could not get DialArgs: %w", err)
	}

	return v1.NewFullNodeRPC(ctx, addr, apiInfo.AuthHeader())
}
