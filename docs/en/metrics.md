## `sophon-miner` metrics

For the configuration of enabling indicators and exporters, please refer to[metrics config](./config-desc.md)

```
// Type: Histogram, Tag: minerID, Semantics: Time to get `BaseInfo` (unit: milliseconds)
GetBaseInfoDuration = stats.Float64("getbaseinfo_ms", "Duration of GetBaseInfo in miner", stats.UnitMilliseconds)
// Type: histogram, Tag: minerID, Semantics: time-consuming to calculate `ticket` (unit: milliseconds)
ComputeTicketDuration = stats.Float64("computeticket_ms", "Duration of ComputeTicket in miner", stats.UnitMilliseconds)
// Type: Histogram, Tag: minerID, Semantics: Time to calculate whether to win the block right (unit: milliseconds)
IsRoundWinnerDuration = stats.Float64("isroundwinner_ms", "Duration of IsRoundWinner in miner", stats.UnitMilliseconds)
// Type: Histogram, Tag: minerID, Semantics: Time to calculate winning proof (unit: seconds)
ComputeProofDuration = stats.Float64("computeproof_s", "Duration of ComputeProof in miner", stats.UnitSeconds)

// Type: Counter, Tag: minerID, Semantics: The number of successful blocks, which means that the local verification
// passed and the broadcast was successful. It was not recognized or excluded due to insufficient base or consensus error.
NumberOfBlock = stats.Int64("number_of_block", "Number of production blocks", stats.UnitDimensionless)
// Type: Counter, Tag: minerID, Semantics: The number of times to obtain the right to produce blocks,
// if the calculation is on the fork chain, the block production will not be recognized
NumberOfIsRoundWinner = stats.Int64("number_of_isroundwinner", "Number of is round winner", stats.UnitDimensionless)

// Type: Counter, Tag: minerID, Semantics: The number of times that the block failed due to the timeout of calculating the winning proof
NumberOfMiningTimeout = stats.Int64("number_of_mining_timeout", "Number of mining failures due to compute proof timeout", stats.UnitDimensionless)
// Type: Counter, Tag: minerID, Semantics: The number of times the block was abandoned due to the chain fork,
// the block generation on the forked chain is meaningless, so the subsequent logic will be abandoned 
// if the chain fork is verified before the block is generated
NumberOfMiningChainFork = stats.Int64("number_of_mining_chain_fork", "Number of mining failures due to chain fork", stats.UnitDimensionless)
// Type: Counter, Tag: minerID, Semantics: Number of block errors due to other errors
NumberOfMiningError = stats.Int64("number_of_mining_error", "Number of mining failures due to error", stats.UnitDimensionless)	
```
