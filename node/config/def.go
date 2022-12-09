package config

import (
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

type MinerConfig struct {
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
