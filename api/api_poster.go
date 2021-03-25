package api

import (
	"context"

	"github.com/filecoin-project/go-address"

	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
)

type MinerAPI interface {
	Common

	AddAddress(dtypes.MinerInfo) error
	UpdateAddress(dtypes.MinerInfo) error
	RemoveAddress(address.Address) error
	ListAddress() ([]dtypes.MinerInfo, error)
	Start(context.Context, address.Address) error
	Stop(context.Context, address.Address) error
}
