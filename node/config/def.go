package config

import (
	"encoding"
	"strings"
	"time"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
)

// Common is common config between full node and miner
type Common struct {
	API    API
	Libp2p Libp2p
	Pubsub Pubsub
}

// API contains configs for API endpoint
type API struct {
	ListenAddress       string
	RemoteListenAddress string
	Timeout             Duration
}

// Libp2p contains configs for libp2p
type Libp2p struct {
	ListenAddresses     []string
	AnnounceAddresses   []string
	NoAnnounceAddresses []string
	BootstrapPeers      []string
	ProtectedPeers      []string

	ConnMgrLow   uint
	ConnMgrHigh  uint
	ConnMgrGrace Duration
}

type Pubsub struct {
	Bootstrapper bool
	DirectPeers  []string
	RemoteTracer string
}

type PosterAddr struct {
	Addr      address.Address
	ListenAPI string
}

// FullNode is a full node config
type FullNode struct {
	Common
}

type MinerConfig struct {
	Common
	PosterAddrs []PosterAddr
	BlockRecord string
}

func ParserPosterAddr(str string) (PosterAddr, error) {
	addrPath := strings.Split(str, "@")
	if len(addrPath) != 2 {
		return PosterAddr{}, xerrors.Errorf("not support PosterAddr format eg f01001|/ip4/127.0.0.1/tcp/1234/http")
	}
	addr, err := address.NewFromString(addrPath[0])
	if err != nil {
		return PosterAddr{}, xerrors.Errorf("fail to parser poster addr in config", err)
	}
	return PosterAddr{
		Addr:      addr,
		ListenAPI: addrPath[1],
	}, nil
}

func defCommon() Common {
	return Common{
		API: API{
			ListenAddress: "/ip4/127.0.0.1/tcp/1234/http",
			Timeout:       Duration(30 * time.Second),
		},
		Libp2p: Libp2p{
			ListenAddresses: []string{
				"/ip4/0.0.0.0/tcp/0",
				"/ip6/::/tcp/0",
			},
			AnnounceAddresses:   []string{},
			NoAnnounceAddresses: []string{},

			ConnMgrLow:   150,
			ConnMgrHigh:  180,
			ConnMgrGrace: Duration(20 * time.Second),
		},
		Pubsub: Pubsub{
			Bootstrapper: false,
			DirectPeers:  nil,
			RemoteTracer: "/dns4/pubsub-tracer.filecoin.io/tcp/4001/p2p/QmTd6UvR47vUidRNZ1ZKXHrAFhqTJAD27rKL9XYghEKgKX",
		},
	}

}

// DefaultFullNode returns the default config
func DefaultFullNode() *FullNode {
	return &FullNode{
		Common: defCommon(),
	}
}

func DefaultMinerConfig() *MinerConfig {
	minerCfg := &MinerConfig{
		Common:      defCommon(),
		PosterAddrs: []PosterAddr{},
		BlockRecord: "localdb",
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
