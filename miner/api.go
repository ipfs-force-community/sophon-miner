package miner

import (
	"context"

	"github.com/filecoin-project/go-address"

	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
)

type MiningAPI interface {
	IMinerMining
	IMinerManager
}

type IMinerMining interface {
	Start(context.Context) error
	Stop(context.Context) error
}

type IMinerManager interface {
	AddAddress(dtypes.MinerInfo) error
	ListAddress() ([]dtypes.MinerInfo, error)
	RemoveAddress(address.Address) error
}

type MockMinerMgr struct {
}

func (m MockMinerMgr) AddAddress(dtypes.MinerInfo) error {
	return nil
}

func (m MockMinerMgr) ListAddress() ([]dtypes.MinerInfo, error) {
	return nil, nil
}

func (m MockMinerMgr) RemoveAddress(a address.Address) error {
	return nil
}
