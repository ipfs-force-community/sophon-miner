package config

import (
	"fmt"
	"net/url"
	"time"

	logging "github.com/ipfs/go-log/v2"

	"github.com/ipfs-force-community/metrics"
)

var log = logging.Logger("config")

// Duration is a wrapper type for Duration
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

type MySQLConfig struct {
	Conn            string
	MaxOpenConn     int      // 100
	MaxIdleConn     int      // 10
	ConnMaxLifeTime Duration // 60s
	Debug           bool
}

type SlashFilterConfig struct {
	Type  string
	MySQL MySQLConfig
}

func newSlashFilterConfig() *SlashFilterConfig {
	return &SlashFilterConfig{
		Type: "local",
		MySQL: MySQLConfig{
			Conn:            "",
			MaxOpenConn:     100,
			MaxIdleConn:     10,
			ConnMaxLifeTime: Duration(60 * time.Second),
			Debug:           false,
		},
	}
}

type API struct {
	ListenAddress       string
	RemoteListenAddress string
	Timeout             Duration

	PrivateKey string
}
type MinerConfig struct {
	API         API
	FullNode    *APIInfo
	Gateway     *GatewayNode
	Auth        *APIInfo
	SubmitNodes []*APIInfo

	PropagationDelaySecs uint64
	MinerOnceTimeout     Duration

	SlashFilter *SlashFilterConfig

	Tracing *metrics.TraceConfig
	Metrics *metrics.MetricsConfig
}

func DefaultMinerConfig() *MinerConfig {
	minerCfg := &MinerConfig{
		API: API{
			ListenAddress: "/ip4/127.0.0.1/tcp/12308",
		},
		FullNode:             defaultAPIInfo(),
		Gateway:              newDefaultGatewayNode(),
		Auth:                 defaultAPIInfo(),
		PropagationDelaySecs: 12,
		MinerOnceTimeout:     Duration(time.Second * 15),
		SlashFilter:          newSlashFilterConfig(),
		Tracing:              metrics.DefaultTraceConfig(),
		Metrics:              metrics.DefaultMetricsConfig(),
	}

	return minerCfg
}

func Check(cfg *MinerConfig) error {
	if len(cfg.API.ListenAddress) > 0 {
		return fmt.Errorf("must config listen address")
	}

	if len(cfg.FullNode.Addr) > 0 && len(cfg.FullNode.Token) > 0 {
		return fmt.Errorf("must config full node url and token")
	}

	_, err := url.Parse(cfg.Auth.Addr)
	if err != nil {
		return fmt.Errorf("auth url format not correct %s %w", cfg.Auth.Addr, err)
	}

	if len(cfg.Gateway.ListenAPI) == 0 {
		return fmt.Errorf("config at lease one gateway url")
	}

	if cfg.SlashFilter.Type == "mysql" {
		if len(cfg.SlashFilter.MySQL.Conn) == 0 {
			return fmt.Errorf("mysql dsn must set when slash filter is mysql")
		}
	} else if cfg.SlashFilter.Type == "local" {
	} else {
		return fmt.Errorf("not support slash filter %s", cfg.SlashFilter.Type)
	}
	return nil
}
