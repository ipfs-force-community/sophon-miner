package config

import (
	"net/http"
	"net/url"

	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
)

type GatewayNode struct {
	ListenAPI []string
	Token     string
}

func (gw *GatewayNode) DialArgs() ([]string, error) {
	var mAddrs []string

	for _, apiAddr := range gw.ListenAPI {
		ma, err := multiaddr.NewMultiaddr(apiAddr)
		if err == nil {
			_, addr, err := manet.DialArgs(ma)
			if err != nil {
				log.Errorf("dial ma err: %s", err.Error())
				continue
			}

			mAddrs = append(mAddrs, "ws://"+addr+"/rpc/v0")
		}

		_, err = url.Parse(apiAddr)
		if err != nil {
			log.Errorf("parse [%s] err: %s", apiAddr, err.Error())
			continue
		}

		mAddrs = append(mAddrs, apiAddr+"/rpc/v0")
	}

	return mAddrs, nil
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
		ListenAPI: []string{},
		Token:     "",
	}
}
