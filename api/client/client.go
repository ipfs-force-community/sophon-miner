package client

import (
	"context"
	"net/http"

	"github.com/filecoin-project/go-jsonrpc"

	venus_api "github.com/filecoin-project/venus/venus-shared/api"
	"github.com/ipfs-force-community/sophon-miner/api"
)

const MajorVersion = 0
const APINamespace = "miner.MinerAPI"
const MethodNamespace = "Filecoin"

// NewCommonRPC creates a new http jsonrpc client.
func NewCommonRPC(ctx context.Context, addr string, requestHeader http.Header) (api.Common, jsonrpc.ClientCloser, error) {
	var res api.CommonStruct
	closer, err := jsonrpc.NewMergeClient(ctx, addr, MethodNamespace,
		[]interface{}{
			&res.Internal,
		},
		requestHeader,
	)

	return &res, closer, err
}

// NewMinerRPC creates a new http jsonrpc client for miner
func NewMinerRPC(ctx context.Context, addr string, requestHeader http.Header, opts ...jsonrpc.Option) (api.MinerAPI, jsonrpc.ClientCloser, error) {
	requestHeader.Set(venus_api.VenusAPINamespaceHeader, APINamespace)
	var res api.MinerAPIStruct
	closer, err := jsonrpc.NewMergeClient(ctx, addr, MethodNamespace,
		[]interface{}{
			&res.CommonStruct.Internal,
			&res.Internal,
		},
		requestHeader,
		opts...,
	)

	return &res, closer, err
}
