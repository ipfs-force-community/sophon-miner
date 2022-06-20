package config

import (
	"net/http"
	"net/url"

	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
)

type APIInfo struct {
	Addr  string
	Token string
}

func defaultAPIInfo() *APIInfo {
	return &APIInfo{
		Addr: "/ip4/0.0.0.0/tcp/12308/http",
		Token:     "",
	}
}

func (a APIInfo) DialArgs(version string) (string, error) {
	ma, err := multiaddr.NewMultiaddr(a.Addr)
	if err == nil {
		_, addr, err := manet.DialArgs(ma)
		if err != nil {
			return "", err
		}

		return "ws://" + addr + "/rpc/" + version, nil
	}

	_, err = url.Parse(a.Addr)
	if err != nil {
		return "", err
	}
	return a.Addr + "/rpc/" + version, nil
}

func (a APIInfo) Host() (string, error) {
	ma, err := multiaddr.NewMultiaddr(a.Addr)
	if err == nil {
		_, addr, err := manet.DialArgs(ma)
		if err != nil {
			return "", err
		}

		return addr, nil
	}

	spec, err := url.Parse(a.Addr)
	if err != nil {
		return "", err
	}
	return spec.Host, nil
}

func (a APIInfo) AuthHeader() http.Header {
	if len(a.Token) != 0 {
		headers := http.Header{}
		headers.Add("Authorization", "Bearer "+string(a.Token))
		return headers
	}
	log.Warn("Sealer API Token not set and requested, capabilities might be limited.")
	return nil
}
