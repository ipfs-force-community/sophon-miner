package metrics

import (
	"context"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

// Global Tags
var (
	MinerID, _ = tag.NewKey("miner_id")
)

// Distribution
var defaultMillisecondsDistribution = view.Distribution(100, 200, 400, 600, 800, 1000, 2000, 20000)
var defaultSecondsDistribution = view.Distribution(3, 4, 5, 6, 7, 8, 9, 10, 12, 14, 16, 18, 20, 25, 30, 180)

var (
	GetBaseInfoDuration   = stats.Float64("getbaseinfo_ms", "Duration of GetBaseInfo in miner", stats.UnitMilliseconds)
	ComputeTicketDuration = stats.Float64("computeticket_ms", "Duration of ComputeTicket in miner", stats.UnitMilliseconds)
	IsRoundWinnerDuration = stats.Float64("isroundwinner_ms", "Duration of IsRoundWinner in miner", stats.UnitMilliseconds)
	ComputeProofDuration  = stats.Float64("computeproof_s", "Duration of ComputeProof in miner", stats.UnitSeconds)

	NumberOfBlock         = stats.Int64("number_of_block", "Number of production blocks", stats.UnitDimensionless)
	NumberOfIsRoundWinner = stats.Int64("number_of_isroundwinner", "Number of is round winner", stats.UnitDimensionless)

	NumberOfMiningTimeout   = stats.Int64("number_of_mining_timeout", "Number of mining failures due to compute proof timeout", stats.UnitDimensionless)
	NumberOfMiningChainFork = stats.Int64("number_of_mining_chain_fork", "Number of mining failures due to chain fork", stats.UnitDimensionless)
	NumberOfMiningError     = stats.Int64("number_of_mining_error", "Number of mining failures due to error", stats.UnitDimensionless)
)

var (
	GetBaseInfoDurationView = &view.View{
		Measure:     GetBaseInfoDuration,
		Aggregation: defaultMillisecondsDistribution,
		TagKeys:     []tag.Key{MinerID},
	}
	ComputeTicketDurationView = &view.View{
		Measure:     ComputeTicketDuration,
		Aggregation: defaultMillisecondsDistribution,
		TagKeys:     []tag.Key{MinerID},
	}
	IsRoundWinnerDurationView = &view.View{
		Measure:     IsRoundWinnerDuration,
		Aggregation: defaultMillisecondsDistribution,
		TagKeys:     []tag.Key{MinerID},
	}
	ComputeProofDurationView = &view.View{
		Measure:     ComputeProofDuration,
		Aggregation: defaultSecondsDistribution,
		TagKeys:     []tag.Key{MinerID},
	}
	NumberOfBlockView = &view.View{
		Measure:     NumberOfBlock,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{MinerID},
	}
	IsRoundWinnerView = &view.View{
		Measure:     NumberOfIsRoundWinner,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{MinerID},
	}
	NumberOfMiningTimeoutView = &view.View{
		Measure:     NumberOfMiningTimeout,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{MinerID},
	}
	NumberOfMiningChainForkView = &view.View{
		Measure:     NumberOfMiningChainFork,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{MinerID},
	}
	NumberOfMiningErrorView = &view.View{
		Measure:     NumberOfMiningError,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{MinerID},
	}
)

var MinerNodeViews = []*view.View{
	GetBaseInfoDurationView,
	ComputeTicketDurationView,
	IsRoundWinnerDurationView,
	ComputeProofDurationView,
	NumberOfBlockView,
	IsRoundWinnerView,
	NumberOfMiningTimeoutView,
	NumberOfMiningChainForkView,
	NumberOfMiningErrorView,
}

func SinceInMilliseconds(startTime time.Time) float64 {
	return float64(time.Since(startTime).Nanoseconds()) / 1e6
}

func TimerMilliseconds(ctx context.Context, m *stats.Float64Measure, minerID string) func() {
	start := time.Now()
	return func() {
		ctx, _ = tag.New(
			ctx,
			tag.Upsert(MinerID, minerID),
		)
		stats.Record(ctx, m.M(SinceInMilliseconds(start)))
	}
}

func SinceInSeconds(startTime time.Time) float64 {
	return float64(time.Since(startTime).Microseconds()) / 1e6
}

func TimerSeconds(ctx context.Context, m *stats.Float64Measure, minerID string) func() {
	start := time.Now()
	return func() {
		ctx, _ = tag.New(
			ctx,
			tag.Upsert(MinerID, minerID),
		)
		stats.Record(ctx, m.M(SinceInSeconds(start)))
	}
}
