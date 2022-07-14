package config

import (
	"time"

	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("config")

type MetricsExporterType string

const (
	METPrometheus MetricsExporterType = "prometheus"
	METGraphite   MetricsExporterType = "graphite"
)

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

type MetricsPrometheusExporterConfig struct {
	RegistryType string `json:"registryType"`
	Path         string `json:"path"`
}

func newMetricsPrometheusExporterConfig() *MetricsPrometheusExporterConfig {
	return &MetricsPrometheusExporterConfig{
		RegistryType: "define",
		Path:         "/debug/metrics",
	}
}

type MetricsGraphiteExporterConfig struct {
	Namespace string `json:"namespace"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
}

func newMetricsGraphiteExporterConfig() *MetricsGraphiteExporterConfig {
	return &MetricsGraphiteExporterConfig{
		Namespace: "miner",
		Host:      "127.0.0.1",
		Port:      4568,
	}
}

type MetricsExporterConfig struct {
	Type MetricsExporterType `json:"type"`

	Prometheus *MetricsPrometheusExporterConfig `json:"prometheus"`
	Graphite   *MetricsGraphiteExporterConfig   `json:"graphite"`
}

func newDefaultMetricsExporterConfig() *MetricsExporterConfig {
	return &MetricsExporterConfig{
		Type:       METPrometheus,
		Prometheus: newMetricsPrometheusExporterConfig(),
		Graphite:   newMetricsGraphiteExporterConfig(),
	}
}

type MetricsConfig struct {
	Enabled  bool                   `json:"enabled"`
	Exporter *MetricsExporterConfig `json:"exporter"`
}

func newDefaultMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		Enabled:  false,
		Exporter: newDefaultMetricsExporterConfig(),
	}
}

type MinerConfig struct {
	FullNode *APIInfo     `json:"fullnode"`
	Gateway  *GatewayNode `json:"gateway"`
	Auth     *APIInfo     `json:"auth"`

	SlashFilter *SlashFilterConfig

	Tracing *TraceConfig   `json:"tracing"`
	Metrics *MetricsConfig `json:"metrics"`
}

func DefaultMinerConfig() *MinerConfig {
	minerCfg := &MinerConfig{
		FullNode:    defaultAPIInfo(),
		Gateway:     newDefaultGatewayNode(),
		Auth:        defaultAPIInfo(),
		SlashFilter: newSlashFilterConfig(),
		Tracing:     newDefaultTraceConfig(),
		Metrics:     newDefaultMetricsConfig(),
	}

	return minerCfg
}
