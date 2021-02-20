package api

import (
	"context"

	"github.com/filecoin-project/go-address"

	"github.com/filecoin-project/venus-miner/node/config"
)

type MinerAPI interface {
	Common

	AddAddress(addr config.MinerInfo) error
	RemoveAddress(addr address.Address) error
	ListAddress() ([]config.MinerInfo, error)
	SetDefault(ctx context.Context, addr address.Address) error
}
