package tracing

import (
	"os"

	"contrib.go.opencensus.io/exporter/jaeger"
	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/trace"

	"github.com/ipfs-force-community/metrics"
)

var log = logging.Logger("tracing")

func SetupJaegerTracing(cfg *metrics.TraceConfig) *jaeger.Exporter {
	if !cfg.JaegerTracingEnabled {
		return nil
	}

	agentEndpointURI := cfg.JaegerEndpoint
	if _, ok := os.LookupEnv("VENUS_MINER_JAEGER"); ok {
		agentEndpointURI = os.Getenv("VENUS_MINER_JAEGER")
	}

	je, err := jaeger.NewExporter(jaeger.Options{
		AgentEndpoint: agentEndpointURI,
		Process: jaeger.Process{
			ServiceName: cfg.ServerName,
		},
	})
	if err != nil {
		log.Errorw("Failed to create the Jaeger exporter", "error", err)
		return nil
	}

	trace.RegisterExporter(je)
	trace.ApplyConfig(trace.Config{
		DefaultSampler: trace.ProbabilitySampler(cfg.ProbabilitySampler),
	})

	log.Infof("register tracing exporter:%s, service name:%s", cfg.JaegerEndpoint, cfg.ServerName)

	return je
}
