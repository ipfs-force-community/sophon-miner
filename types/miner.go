package types

import (
	"fmt"
	"time"

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
	Msg      string         `json:"msg"`
}

type CountWinners struct {
	Miner         address.Address `json:"miner"`
	TotalWinCount int64           `json:"totalWinCount"`
	Msg           string          `json:"msg"`
	WinEpochList  []SimpleWinInfo `json:"winEpochList"`
}

type StateMining int

const (
	Mining StateMining = iota
	Success
	Timeout
	ChainForked
	Error
)

func (sm StateMining) String() string {
	switch sm {
	case Mining:
		return "Mining"
	case Success:
		return "Success"
	case Timeout:
		return "TimeOut"
	case ChainForked:
		return "ChainForked"
	case Error:
		return "Error"
	default:
		return "unknown"
	}
}

type MinedBlock struct {
	ParentEpoch int64  `gorm:"column:parent_epoch;type:bigint(20);default:0;NOT NULL"`
	ParentKey   string `gorm:"column:parent_key;type:varchar(2048);default:'';NOT NULL"`

	Epoch int64  `gorm:"column:epoch;type:bigint(20);NOT NULL;primary_key"`
	Miner string `gorm:"column:miner;type:varchar(256);NOT NULL;primary_key"`
	Cid   string `gorm:"column:cid;type:varchar(256);default:''"`

	WinningAt time.Time   `gorm:"column:winning_at;type:datetime"`
	MineState StateMining `gorm:"column:mine_state;type:tinyint(4);default:0;comment:0-mining,1-success,2-timeout,3-chain forked,4-error;NOT NULL"`
	Consuming int64       `gorm:"column:consuming;type:bigint(10);default:0;NOT NULL"` // reserved
}

func (m *MinedBlock) TableName() string {
	return "miner_blocks"
}

type BlocksQueryParams struct {
	Miners []address.Address
	Limit  int
	Offset int
}

type QueryRecordParams struct {
	Miner address.Address
	Epoch abi.ChainEpoch
	Limit uint
}

type ErrorCode int

const (
	ConnectGatewayError ErrorCode = iota
	CallNodeRPCError
	WalletSignError
)

func (e ErrorCode) String() string {
	switch e {
	case ConnectGatewayError:
		return "ConnectGatewayError"
	case CallNodeRPCError:
		return "CallNodeRPCError"
	case WalletSignError:
		return "WalletSignError"
	default:
		return "unknown"
	}
}

func (e ErrorCode) Error() string {
	return fmt.Sprintf("%d", e)
}
