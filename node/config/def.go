package config

import (
	"encoding"
	"net/http"
	"net/url"
	"time"

	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/prometheus/common/log"
)

// Common is common config between full node and miner
type Common struct {
	API API
}

// API contains configs for API endpoint
type API struct {
	ListenAddress       string
	RemoteListenAddress string
	Timeout             Duration
}

type FullNode struct {
	ListenAPI string
	Token     string
}

func (sn FullNode) DialArgs() (string, error) {
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

func (sn FullNode) AuthHeader() http.Header {
	if len(sn.Token) != 0 {
		headers := http.Header{}
		headers.Add("Authorization", "Bearer "+string(sn.Token))
		return headers
	}
	log.Warn("Sealer API Token not set and requested, capabilities might be limited.")
	return nil
}

func defFullNode() FullNode {
	return FullNode{
		ListenAPI: "/ip4/127.0.0.1/tcp/1234/http",
		Token:     "",
	}
}

type MinerConfig struct {
	Common

	FullNode

	BlockRecord string
}

func defCommon() Common {
	return Common{
		API: API{
			ListenAddress: "/ip4/127.0.0.1/tcp/1234/http",
			Timeout:       Duration(30 * time.Second),
		},
	}
}

func DefaultMinerConfig() *MinerConfig {
	minerCfg := &MinerConfig{
		Common:      defCommon(),
		FullNode:    defFullNode(),
		BlockRecord: "cache",
	}

	minerCfg.Common.API.ListenAddress = "/ip4/0.0.0.0/tcp/12308/http" //change default address

	return minerCfg
}

var _ encoding.TextMarshaler = (*Duration)(nil)
var _ encoding.TextUnmarshaler = (*Duration)(nil)

// Duration is a wrapper type for time.Duration
// for decoding and encoding from/to TOML
type Duration time.Duration

// UnmarshalText implements interface for TOML decoding
func (dur *Duration) UnmarshalText(text []byte) error {
	d, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	*dur = Duration(d)
	return err
}

func (dur Duration) MarshalText() ([]byte, error) {
	d := time.Duration(dur)
	return []byte(d.String()), nil
}
