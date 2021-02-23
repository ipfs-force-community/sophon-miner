package api

import (
	"github.com/filecoin-project/go-address"

	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
)

type MinerAPI interface {
	Common

	AddAddress(dtypes.MinerInfo) error
	RemoveAddress(address.Address) error
	ListAddress() ([]dtypes.MinerInfo, error)
	SetDefault(address.Address) error
	Default() (address.Address, error)
}
