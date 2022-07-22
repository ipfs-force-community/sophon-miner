package config

import (
	"time"

	logging "github.com/ipfs/go-log/v2"

	"github.com/ipfs-force-community/metrics"
)

var log = logging.Logger("config")

type MySQLConfig struct {
	Conn            string        `json:"conn"`
	MaxOpenConn     int           `json:"maxOpenConn"`     // 100
	MaxIdleConn     int           `json:"maxIdleConn"`     // 10
	ConnMaxLifeTime time.Duration `json:"connMaxLifeTime"` // 60(s)
	Debug           bool          `json:"debug"`
}

type SlashFilterConfig struct {
	Type  string      `json:"type"`
	MySQL MySQLConfig `json:"mysql"`
}

func newSlashFilterConfig() *SlashFilterConfig {
	return &SlashFilterConfig{
		Type: "local",
		MySQL: MySQLConfig{
			Conn:            "",
			MaxOpenConn:     100,
			MaxIdleConn:     10,
			ConnMaxLifeTime: 60,
			Debug:           false,
		},
	}
}

type MinerConfig struct {
	FullNode *APIInfo     `json:"fullnode"`
	Gateway  *GatewayNode `json:"gateway"`
	Auth     *APIInfo     `json:"auth"`

	SlashFilter *SlashFilterConfig

	Tracing *metrics.TraceConfig   `json:"tracing"`
	Metrics *metrics.MetricsConfig `toml:"metrics"`
}

func DefaultMinerConfig() *MinerConfig {
	minerCfg := &MinerConfig{
		FullNode:    defaultAPIInfo(),
		Gateway:     newDefaultGatewayNode(),
		Auth:        defaultAPIInfo(),
		SlashFilter: newSlashFilterConfig(),
		Tracing:     metrics.DefaultTraceConfig(),
		Metrics:     metrics.DefaultMetricsConfig(),
	}

	return minerCfg
}
