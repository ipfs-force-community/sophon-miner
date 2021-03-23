package client

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-jsonrpc"

	"github.com/ipfs-force-community/venus-wallet/api/remotecli"
	"github.com/ipfs-force-community/venus-wallet/storage/wallet"

	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
)

func NewWalletRPC(ctx context.Context, w *dtypes.WalletNode) (wallet.IWallet, jsonrpc.ClientCloser, error) {
	addr, err := w.DialArgs()
	if err != nil {
		return nil, nil, xerrors.Errorf("could not get DialArgs: %w", err)
	}

	return remotecli.NewWalletRPC(ctx, addr,  w.AuthHeader())
}
