package v180

import (
	"github.com/ipfs-force-community/metrics"

	"github.com/filecoin-project/venus-miner/node/config"
)

type MySQLConfig struct {
	Conn            string
	MaxOpenConn     int             // 100
	MaxIdleConn     int             // 10
	ConnMaxLifeTime config.Duration // "1m0s"
	Debug           bool
}

type SlashFilterConfig struct {
	Type  string
	MySQL MySQLConfig
}

type MinerConfig struct {
	FullNode *config.APIInfo
	Gateway  *config.GatewayNode
	Auth     *config.APIInfo

	SlashFilter *SlashFilterConfig

	Tracing *metrics.TraceConfig
	Metrics *metrics.MetricsConfig
}
