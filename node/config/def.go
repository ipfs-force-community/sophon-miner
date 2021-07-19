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

func (sn FullNode) DialArgs(version string) (string, error) {
	ma, err := multiaddr.NewMultiaddr(sn.ListenAPI)
	if err == nil {
		_, addr, err := manet.DialArgs(ma)
		if err != nil {
			return "", err
		}

		return "ws://" + addr + "/rpc/" + version, nil
	}

	_, err = url.Parse(sn.ListenAPI)
	if err != nil {
		return "", err
	}
	return sn.ListenAPI + "/rpc/" + version, nil
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

type MySQLConfig struct {
	Conn            string        `json:"conn"`
	MaxOpenConn     int           `json:"maxOpenConn"`     // 100
	MaxIdleConn     int           `json:"maxIdleConn"`     // 10
	ConnMaxLifeTime time.Duration `json:"connMaxLifeTime"` // minuter: 60
	Debug           bool          `json:"debug"`
}

type MinerDbConfig struct {
	Type   string      `json:"type"`
	SFType string      `json:"sfType"`
	MySQL  MySQLConfig `json:"mysql"`
	Auth   *AuthConfig `json:"auth"`
}

func newDefaultMinerDbConfig() *MinerDbConfig {
	return &MinerDbConfig{
		Type:   "auth",
		SFType: "local",
		MySQL:  MySQLConfig{},
		Auth:   newDefaultAuthConfig(),
	}
}

// TraceConfig holds all configuration options related to enabling and exporting
// filecoin node traces.
type TraceConfig struct {
	// JaegerTracingEnabled will enable exporting traces to jaeger when true.
	JaegerTracingEnabled bool `json:"jaegerTracingEnabled"`
	// JaegerEndpoint is the URL traces are collected on.
	JaegerEndpoint string `json:"jaegerEndpoint"`
	// ProbabilitySampler will sample fraction of traces, 1.0 will sample all traces.
	ProbabilitySampler float64 `json:"probabilitySampler"`
	// ServerName
	ServerName string `json:"servername"`
}

func newDefaultTraceConfig() *TraceConfig {
	return &TraceConfig{
		JaegerTracingEnabled: true,
		JaegerEndpoint:       "localhost:6831",
		ProbabilitySampler:   1.0,
		ServerName:           "venus-miner",
	}
}

type MinerConfig struct {
	Common

	FullNode

	BlockRecord string

	Gateway *GatewayNode `json:"gateway"`

	Db *MinerDbConfig `json:"db"`

	Tracing *TraceConfig `json:"tracing"`
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
		Gateway:     newDefaultGatewayNode(),
		Db:          newDefaultMinerDbConfig(),
		Tracing:     newDefaultTraceConfig(),
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
