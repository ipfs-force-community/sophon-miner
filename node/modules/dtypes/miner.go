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

type MinerInfo struct {
	Addr      address.Address
	ListenAPI string
	Token     string
}

func (m MinerInfo) DialArgs() (string, error) {
	ma, err := multiaddr.NewMultiaddr(m.ListenAPI)
	if err == nil {
		_, addr, err := manet.DialArgs(ma)
		if err != nil {
			return "", err
		}

		return "ws://" + addr + "/rpc/v0", nil
	}

	_, err = url.Parse(m.ListenAPI)
	if err != nil {
		return "", err
	}
	return m.ListenAPI + "/rpc/v0", nil
}

func (m MinerInfo) AuthHeader() http.Header {
	if len(m.Token) != 0 {
		headers := http.Header{}
		headers.Add("Authorization", "Bearer "+string(m.Token))
		return headers
	}
	log.Warn("API Token not set and requested, capabilities might be limited.")
	return nil
}
