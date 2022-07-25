package config

import (
	"time"

	logging "github.com/ipfs/go-log/v2"

	"github.com/ipfs-force-community/metrics"
)

var log = logging.Logger("config")

type MySQLConfig struct {
	Conn            string
	MaxOpenConn     int           // 100
	MaxIdleConn     int           // 10
	ConnMaxLifeTime time.Duration // 60(s)
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
			ConnMaxLifeTime: 60,
			Debug:           false,
		},
	}
}

type MinerConfig struct {
	FullNode *APIInfo
	Gateway  *GatewayNode
	Auth     *APIInfo

	SlashFilter *SlashFilterConfig

	Tracing *metrics.TraceConfig
	Metrics *metrics.MetricsConfig
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
