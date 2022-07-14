package metrics

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"time"

	"contrib.go.opencensus.io/exporter/graphite"
	"contrib.go.opencensus.io/exporter/prometheus"
	promclient "github.com/prometheus/client_golang/prometheus"
	"go.opencensus.io/stats/view"

	"github.com/filecoin-project/venus-miner/node/config"
)

type RegistryType string

const (
	RTDefault RegistryType = "default"
	RTDefine  RegistryType = "define"
)

func PrometheusExporter(rtType RegistryType) (http.Handler, error) {
	var registry *promclient.Registry
	var ok bool

	switch rtType {
	case RTDefault:
		// Prometheus globals are exposed as interfaces, but the prometheus
		// OpenCensus exporter expects a concrete *Registry. The concrete type of
		// the globals are actually *Registry, so we downcast them, staying
		// defensive in case things change under the hood.
		registry, ok = promclient.DefaultRegisterer.(*promclient.Registry)
		if !ok {
			return nil, fmt.Errorf("failed to export default prometheus registry; some metrics will be unavailable; unexpected type: %T", promclient.DefaultRegisterer)
		}
	case RTDefine:
		// The metrics of OpenCensus in the same process will be automatically
		// registered to the custom registry of Prometheus, so no additional
		// registration action is required
		registry = promclient.NewRegistry()
	default:
		return nil, fmt.Errorf("wrong registry type: %s", rtType)
	}

	exporter, err := prometheus.NewExporter(prometheus.Options{
		Registry:  registry,
		Namespace: "miner", // 不允许有-, 如"venus-miner". prometheus不接受
	})
	if err != nil {
		return nil, fmt.Errorf("could not create the prometheus stats exporter: %w", err)
	}
	view.RegisterExporter(exporter)

	return exporter, nil
}

func RegisterGraphiteExporter(cfg *config.MetricsGraphiteExporterConfig) error {
	exporter, err := graphite.NewExporter(graphite.Options{Namespace: "venus", Host: "127.0.0.1", Port: 4568})
	if err != nil {
		return fmt.Errorf("failed to create graphite exporter: %w", err)
	}

	view.RegisterExporter(exporter)

	view.SetReportingPeriod(5 * time.Second)

	return nil
}
