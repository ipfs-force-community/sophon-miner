package client

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/specs-actors/actors/runtime/proof"

	"github.com/filecoin-project/venus-wallet/core"
	"github.com/ipfs-force-community/venus-gateway/types"

	"github.com/filecoin-project/venus-miner/node/config"
)

type GatewayAPI struct {
	WalletSign             func(ctx context.Context, account string, addr address.Address, toSign []byte, meta core.MsgMeta) (*crypto.Signature, error)
	ComputeProof           func(ctx context.Context, miner address.Address, reqBody *types.ComputeProofRequest) ([]proof.PoStProof, error)
}

func NewGatewayRPC(cfg *config.GatewayNode) (*GatewayAPI, jsonrpc.ClientCloser, error) {
	addr, err := cfg.DialArgs()
	if err != nil {
		return nil, nil, xerrors.Errorf("could not get DialArgs: %w", err)
	}

	var gatewayAPI = &GatewayAPI{}
	closer, err := jsonrpc.NewMergeClient(context.Background(), addr, "Filecoin",
		[]interface{}{
			gatewayAPI,
		},
		cfg.AuthHeader(),
	)

	return gatewayAPI, closer, err
}
