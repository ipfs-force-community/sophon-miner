package types

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
)

type NetworkName string

type MinerInfo struct {
	Addr       address.Address
	Id         string
	Name       string
	OpenMining bool
}

type MinerState struct {
	Addr     address.Address
	IsMining bool
	Err      []string
}

type SimpleWinInfo struct {
	Epoch    abi.ChainEpoch `json:"epoch"`
	WinCount int64          `json:"winCount"`
}

type CountWinners struct {
	Miner         address.Address `json:"miner"`
	TotalWinCount int64           `json:"totalWinCount"`
	Msg           string          `json:"msg"`
	WinEpochList  []SimpleWinInfo `json:"winEpochList"`
}
