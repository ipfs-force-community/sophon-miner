package minermanage

import (
	"github.com/filecoin-project/go-address"

	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
)

const (
	Local = "local"
	MySQL = "mysql"
)

type MinerManageAPI interface {
	Put(addr dtypes.MinerInfo) error
	Set(addr dtypes.MinerInfo) error
	Has(checkAddr address.Address) bool
	Get(checkAddr address.Address) *dtypes.MinerInfo
	List() ([]dtypes.MinerInfo, error)
	Remove(rmAddr address.Address) error
	Count() int
}
