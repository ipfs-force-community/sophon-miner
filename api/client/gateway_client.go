package client

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-jsonrpc"

	"github.com/filecoin-project/venus-miner/node/config"

	gateway "github.com/filecoin-project/venus/venus-shared/api/gateway/v1"
)

func NewGatewayRPC(ctx context.Context, cfg *config.GatewayNode) (gateway.IGateway, jsonrpc.ClientCloser, error) {
	var err error
	addrs, err := cfg.DialArgs()
	if err != nil {
		return nil, nil, fmt.Errorf("could not get DialArgs: %w", err)
	}

	var gatewayAPI gateway.IGateway = nil
	var closer jsonrpc.ClientCloser
	for _, addr := range addrs {
		gatewayAPI, closer, err = gateway.NewIGatewayRPC(ctx, addr, cfg.AuthHeader())
		if err == nil {
			return gatewayAPI, closer, err
		}
	}

	return gatewayAPI, closer, err
}
