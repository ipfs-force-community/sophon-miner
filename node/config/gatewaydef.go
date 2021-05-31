package config

import (
	"net/http"
	"net/url"

	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/prometheus/common/log"
)

type GatewayNode struct {
	ListenAPI string
	Token     string
}

func (gw *GatewayNode) DialArgs() (string, error) {
	ma, err := multiaddr.NewMultiaddr(gw.ListenAPI)
	if err == nil {
		_, addr, err := manet.DialArgs(ma)
		if err != nil {
			return "", err
		}

		return "ws://" + addr + "/rpc/v0", nil
	}

	_, err = url.Parse(gw.ListenAPI)
	if err != nil {
		return "", err
	}
	return gw.ListenAPI + "/rpc/v0", nil
}

func (gw *GatewayNode) AuthHeader() http.Header {
	if len(gw.Token) != 0 {
		headers := http.Header{}
		headers.Add("Authorization", "Bearer "+string(gw.Token))
		return headers
	}
	log.Warn("Sealer API Token not set and requested, capabilities might be limited.")
	return nil
}

func newDefaultGatewayNode() *GatewayNode {
	return &GatewayNode{
		ListenAPI: "",
		Token:     "",
	}
}
