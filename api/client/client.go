package client

import (
	"context"
	"net/http"

	"github.com/filecoin-project/go-jsonrpc"

	"github.com/ipfs-force-community/sophon-miner/api"
)

// NewCommonRPC creates a new http jsonrpc client.
func NewCommonRPC(ctx context.Context, addr string, requestHeader http.Header) (api.Common, jsonrpc.ClientCloser, error) {
	var res api.CommonStruct
	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Filecoin",
		[]interface{}{
			&res.Internal,
		},
		requestHeader,
	)

	return &res, closer, err
}

// NewMinerRPC creates a new http jsonrpc client for miner
func NewMinerRPC(ctx context.Context, addr string, requestHeader http.Header, opts ...jsonrpc.Option) (api.MinerAPI, jsonrpc.ClientCloser, error) {
	var res api.MinerAPIStruct
	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Filecoin",
		[]interface{}{
			&res.CommonStruct.Internal,
			&res.Internal,
		},
		requestHeader,
		opts...,
	)

	return &res, closer, err
}
