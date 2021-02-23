package dtypes

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
)

type MinerAddress address.Address
type MinerID abi.ActorID

type MinerInfo struct {
	Addr      address.Address
	ListenAPI string
	Token     string
}
