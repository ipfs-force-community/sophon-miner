package v0api

import (
	"context"

	"github.com/filecoin-project/venus-miner/api"
	"github.com/filecoin-project/venus-miner/api/v1api"
)

type WrapperV1Full struct {
	v1api.FullNode
}

func (w *WrapperV1Full) Version(ctx context.Context) (api.APIVersion, error) {
	ver, err := w.FullNode.Version(ctx)
	if err != nil {
		return api.APIVersion{}, err
	}

	ver.APIVersion = api.FullAPIVersion0

	return ver, nil
}

var _ FullNode = &WrapperV1Full{}
