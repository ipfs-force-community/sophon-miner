package client

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/go-state-types/network"

	"github.com/filecoin-project/venus-wallet/core"

	"github.com/filecoin-project/venus-miner/node/config"

	"github.com/filecoin-project/venus/venus-shared/actors/builtin"
)

type GatewayAPI struct {
	WalletSign   func(context.Context, string, address.Address, []byte, core.MsgMeta) (*crypto.Signature, error)
	ComputeProof func(ctx context.Context, miner address.Address, sectorInfos []builtin.ExtendedSectorInfo, rand abi.PoStRandomness, height abi.ChainEpoch, nwVersion network.Version) ([]builtin.PoStProof, error)
}

func NewGatewayRPC(cfg *config.GatewayNode) (*GatewayAPI, jsonrpc.ClientCloser, error) {
	var err error
	addrs, err := cfg.DialArgs()
	if err != nil {
		return nil, nil, xerrors.Errorf("could not get DialArgs: %w", err)
	}

	var gatewayAPI = &GatewayAPI{}
	var closer jsonrpc.ClientCloser
	for _, addr := range addrs {
		closer, err = jsonrpc.NewMergeClient(context.Background(), addr, "Gateway",
			[]interface{}{
				gatewayAPI,
			},
			cfg.AuthHeader(),
		)
		if err == nil {
			return gatewayAPI, closer, err
		}
	}

	return gatewayAPI, closer, err
}
