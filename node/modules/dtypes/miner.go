package dtypes

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
)

type NetworkName string

type MinerInfo struct {
	Addr address.Address
	Id   string
	Name string
}

type MinerState struct {
	Addr     address.Address
	IsMining bool
	Err      []string
}

type User struct {
	ID         string
	Name       string
	Miner      string
	Comment    string
	State      int //0 for init, 1 for active
	CreateTime uint64
	UpdateTime uint64
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
