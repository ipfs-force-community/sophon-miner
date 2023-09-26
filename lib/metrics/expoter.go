package metrics

import (
	"context"
	"fmt"

	"go.opencensus.io/stats/view"

	"github.com/ipfs-force-community/metrics"
)

func SetupMetrics(ctx context.Context, metricsConfig *metrics.MetricsConfig) error {
	if err := view.Register(
		MinerNodeViews...,
	); err != nil {
		return fmt.Errorf("cannot register the view: %w", err)
	}

	return metrics.SetupMetrics(ctx, metricsConfig)
}
