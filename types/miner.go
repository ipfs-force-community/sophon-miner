package types

import (
	"time"

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
	ID         string `json:"id"`
	Name       string `json:"name"`
	SourceType int    `json:"sourceType"`
	Comment    string `json:"comment"`
	State      int    `json:"state"`
	CreateTime int64  `json:"createTime"`
	UpdateTime int64  `json:"updateTime"`
}

type Miner struct {
	Miner, User          string
	CreatedAt, UpdatedAt time.Time
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
