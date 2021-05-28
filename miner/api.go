package miner

import (
	"context"
	"github.com/filecoin-project/go-state-types/abi"

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
	ManualStart(context.Context, address.Address) error
	ManualStop(context.Context, address.Address) error
}

type IMinerManager interface {
	AddAddress(dtypes.MinerInfo) error
	UpdateAddress(dtypes.MinerInfo) error
	ListAddress() ([]dtypes.MinerInfo, error)
	StatesForMining([]address.Address) ([]dtypes.MinerState, error)
	CountWinners([]address.Address, abi.ChainEpoch, abi.ChainEpoch) ([]dtypes.CountWinners, error)
	RemoveAddress(address.Address) error
}

type MockMinerMgr struct {
}

var _ IMinerManager = &MockMinerMgr{}

func (m MockMinerMgr) AddAddress(dtypes.MinerInfo) error {
	return nil
}

func (m MockMinerMgr) UpdateAddress(dtypes.MinerInfo) error {
	return nil
}

func (m MockMinerMgr) ListAddress() ([]dtypes.MinerInfo, error) {
	return nil, nil
}

func (m MockMinerMgr) StatesForMining([]address.Address) ([]dtypes.MinerState, error) {
	return nil, nil
}

func (m MockMinerMgr) CountWinners([]address.Address, abi.ChainEpoch, abi.ChainEpoch) ([]dtypes.CountWinners, error) {
	return nil, nil
}

func (m MockMinerMgr) RemoveAddress(a address.Address) error {
	return nil
}
