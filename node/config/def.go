package config

import (
	"time"

	logging "github.com/ipfs/go-log/v2"
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
		JaegerTracingEnabled: false,
		JaegerEndpoint:       "localhost:6831",
		ProbabilitySampler:   1.0,
		ServerName:           "venus-miner",
	}
}

type MinerConfig struct {
	FullNode *APIInfo     `json:"fullnode"`
	Gateway  *GatewayNode `json:"gateway"`
	Auth     *APIInfo     `json:"auth"`

	SlashFilter *SlashFilterConfig

	Tracing *TraceConfig `json:"tracing"`
}

func DefaultMinerConfig() *MinerConfig {
	minerCfg := &MinerConfig{
		FullNode:    defaultAPIInfo(),
		Gateway:     newDefaultGatewayNode(),
		Auth:        defaultAPIInfo(),
		SlashFilter: newSlashFilterConfig(),
		Tracing:     newDefaultTraceConfig(),
	}

	return minerCfg
}
