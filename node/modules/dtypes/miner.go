package dtypes

import (
	"net/http"
	"net/url"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/prometheus/common/log"
)

type NetworkName string

type MinerAddress address.Address
type MinerID abi.ActorID

type WalletNode struct {
	ListenAPI string
	Token     string
}

type SealerNode struct {
	ListenAPI string
	Token     string
}

type MinerInfo struct {
	Addr   address.Address
	Sealer SealerNode
	Wallet WalletNode
}

type MinerState struct {
	Addr     address.Address
	IsMining bool
	Err      []string
}

func (sn SealerNode) DialArgs() (string, error) {
	ma, err := multiaddr.NewMultiaddr(sn.ListenAPI)
	if err == nil {
		_, addr, err := manet.DialArgs(ma)
		if err != nil {
			return "", err
		}

		return "ws://" + addr + "/rpc/v0", nil
	}

	_, err = url.Parse(sn.ListenAPI)
	if err != nil {
		return "", err
	}
	return sn.ListenAPI + "/rpc/v0", nil
}

func (sn SealerNode) AuthHeader() http.Header {
	if len(sn.Token) != 0 {
		headers := http.Header{}
		headers.Add("Authorization", "Bearer "+string(sn.Token))
		return headers
	}
	log.Warn("Sealer API Token not set and requested, capabilities might be limited.")
	return nil
}

func (wn WalletNode) DialArgs() (string, error) {
	ma, err := multiaddr.NewMultiaddr(wn.ListenAPI)
	if err == nil {
		_, addr, err := manet.DialArgs(ma)
		if err != nil {
			return "", err
		}

		return "ws://" + addr + "/rpc/v0", nil
	}

	_, err = url.Parse(wn.ListenAPI)
	if err != nil {
		return "", err
	}
	return wn.ListenAPI + "/rpc/v0", nil
}

func (wn WalletNode) AuthHeader() http.Header {
	if len(wn.Token) != 0 {
		headers := http.Header{}
		headers.Add("Authorization", "Bearer "+string(wn.Token))
		return headers
	}
	log.Warn("Sealer API Token not set and requested, capabilities might be limited.")
	return nil
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
