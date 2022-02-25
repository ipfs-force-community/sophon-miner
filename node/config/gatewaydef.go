package config

import (
	"net/http"

	"github.com/ipfs-force-community/venus-common-utils/apiinfo"
)

type GatewayNode struct {
	ListenAPI []string
	Token     string
}

func (gw *GatewayNode) DialArgs() ([]string, error) {
	var mAddrs []string

	for _, apiAddr := range gw.ListenAPI {
		apiInfo := apiinfo.NewAPIInfo(apiAddr, gw.Token)
		addr, err := apiInfo.DialArgs("v1")
		if err != nil {
			log.Errorf("dial ma err: %s", err.Error())
			continue
		}


		mAddrs = append(mAddrs, addr)
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
