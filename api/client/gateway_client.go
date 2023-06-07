package client

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-jsonrpc"

	"github.com/ipfs-force-community/sophon-miner/node/config"

	gatewayAPIV2 "github.com/filecoin-project/venus/venus-shared/api/gateway/v2"
)

func NewGatewayRPC(ctx context.Context, cfg *config.GatewayNode) (gatewayAPIV2.IGateway, jsonrpc.ClientCloser, error) {
	var err error
	addrs, err := cfg.DialArgs()
	if err != nil {
		return nil, nil, fmt.Errorf("could not get DialArgs: %w", err)
	}

	var gatewayAPI gatewayAPIV2.IGateway = nil
	var closer jsonrpc.ClientCloser
	for _, addr := range addrs {
		gatewayAPI, closer, err = gatewayAPIV2.NewIGatewayRPC(ctx, addr, cfg.AuthHeader())
		if err == nil {
			return gatewayAPI, closer, err
		}
	}

	return gatewayAPI, closer, err
}
