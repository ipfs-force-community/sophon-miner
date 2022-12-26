package v170

import (
	"time"

	"github.com/ipfs-force-community/metrics"

	"github.com/filecoin-project/venus-miner/node/config"
	"github.com/filecoin-project/venus-miner/node/config/migrate/v180"
)

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

type MinerConfig struct {
	FullNode *config.APIInfo
	Gateway  *config.GatewayNode
	Auth     *config.APIInfo

	SlashFilter *SlashFilterConfig

	Tracing *metrics.TraceConfig
	Metrics *metrics.MetricsConfig
}

func (cfgV170 *MinerConfig) ToMinerConfig(cfg *v180.MinerConfig) {
	cfg.FullNode = cfgV170.FullNode
	cfg.Gateway = cfgV170.Gateway
	cfg.Auth = cfgV170.Auth
	cfg.SlashFilter.Type = cfgV170.SlashFilter.Type
	cfg.SlashFilter.MySQL.Conn = cfgV170.SlashFilter.MySQL.Conn
	cfg.SlashFilter.MySQL.MaxOpenConn = cfgV170.SlashFilter.MySQL.MaxOpenConn
	cfg.SlashFilter.MySQL.MaxIdleConn = cfgV170.SlashFilter.MySQL.MaxIdleConn
	cfg.SlashFilter.MySQL.ConnMaxLifeTime = config.Duration(cfgV170.SlashFilter.MySQL.ConnMaxLifeTime)
	cfg.SlashFilter.MySQL.Debug = cfgV170.SlashFilter.MySQL.Debug
	cfg.Tracing = cfgV170.Tracing
	cfg.Metrics = cfgV170.Metrics
}
