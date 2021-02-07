package miner

import (
	"context"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/venus-miner/node/config"
)

type BlockMinerApi interface {
	IMiner
	IMinerMgr
}

type IMiner interface {
	Start(context.Context) error
	Stop(context.Context) error
}

type IMinerMgr interface {
	AddAddress(config.PosterAddr) error
	ListAddress() ([]config.PosterAddr, error)
	RemoveAddress(address.Address) error
}

type MockMinerMgr struct {
}

func (m MockMinerMgr) AddAddress(a config.PosterAddr) error {
	return nil
}

func (m MockMinerMgr) ListAddress() ([]config.PosterAddr, error) {
	return nil, nil
}

func (m MockMinerMgr) RemoveAddress(a address.Address) error {
	return nil
}
