package metrics

import (
	"context"
	"fmt"

	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/stats/view"

	"github.com/ipfs-force-community/metrics"
)

var log = logging.Logger("miner")

func SetupMetrics(ctx context.Context, metricsConfig *metrics.MetricsConfig) error {
	if err := view.Register(
		MinerNodeViews...,
	); err != nil {
		return fmt.Errorf("cannot register the view: %w", err)
	}

	return metrics.SetupMetrics(ctx, metricsConfig)
}
