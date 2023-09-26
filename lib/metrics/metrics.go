package metrics

import (
	"context"
	"time"

	rpcMetrics "github.com/filecoin-project/go-jsonrpc/metrics"
	"github.com/ipfs-force-community/metrics"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

// Global Tags
var (
	MinerID, _ = tag.NewKey("miner_id")
)

var (
	NumberOfBlock         = stats.Int64("number_of_block", "Number of production blocks", stats.UnitDimensionless)
	NumberOfIsRoundWinner = stats.Int64("number_of_isroundwinner", "Number of is round winner", stats.UnitDimensionless)

	NumberOfMiningTimeout   = stats.Int64("number_of_mining_timeout", "Number of mining failures due to compute proof timeout", stats.UnitDimensionless)
	NumberOfMiningChainFork = stats.Int64("number_of_mining_chain_fork", "Number of mining failures due to chain fork", stats.UnitDimensionless)
	NumberOfMiningError     = stats.Int64("number_of_mining_error", "Number of mining failures due to error", stats.UnitDimensionless)
)

var (
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

var (
	ApiState        = metrics.NewInt64("api/state", "api service state. 0: down, 1: up", "")
	MinerNumInState = metrics.NewInt64WithCategory("miner/num", "miner num in vary state", "")

	GetBaseInfoDuration      = metrics.NewTimerMs("mine/getbaseinfo", "Duration of GetBaseInfo in miner", MinerID)
	ComputeTicketDuration    = metrics.NewTimerMs("mine/computeticket", "Duration of ComputeTicket in miner", MinerID)
	CheckRoundWinnerDuration = metrics.NewTimerMs("mine/checkroundwinner", "Duration of Check Round Winner in miner", MinerID)
	ComputeProofDuration     = metrics.NewTimerMs("mine/computeproof", "Duration of ComputeProof in miner", MinerID)
)

var MinerNodeViews = append([]*view.View{
	NumberOfBlockView,
	IsRoundWinnerView,
	NumberOfMiningTimeoutView,
	NumberOfMiningChainForkView,
	NumberOfMiningErrorView,
}, rpcMetrics.DefaultViews...)

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
