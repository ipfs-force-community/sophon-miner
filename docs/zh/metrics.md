## `venus-miner` 指标

有关开启指标，导出器的配置请参考[metrics config](./config-desc.md)

```
// 类型：直方图，Tag：minerID，语义：获取 `BaseInfo` 耗时(单位:毫秒)
GetBaseInfoDuration   = stats.Float64("getbaseinfo_ms", "Duration of GetBaseInfo in miner", stats.UnitMilliseconds)
// 类型：直方图，Tag：minerID，语义：计算 `ticket` 耗时(单位:毫秒)
ComputeTicketDuration = stats.Float64("computeticket_ms", "Duration of ComputeTicket in miner", stats.UnitMilliseconds)
// 类型：直方图，Tag：minerID，语义：计算是否赢得出块权耗时(单位:毫秒)
IsRoundWinnerDuration = stats.Float64("isroundwinner_ms", "Duration of IsRoundWinner in miner", stats.UnitMilliseconds)
// 类型：直方图，Tag：minerID，语义：计算获胜证明耗时(单位:秒)
ComputeProofDuration  = stats.Float64("computeproof_s", "Duration of ComputeProof in miner", stats.UnitSeconds)

// 类型：计数器，Tag：minerID，语义：成功出块次数,指本地验证通过并成功广播,因base不足或共识错误没被承认没排除
NumberOfBlock         = stats.Int64("number_of_block", "Number of production blocks", stats.UnitDimensionless)
// 类型：计数器，Tag：minerID，语义：获得出块权的次数,如果计算时处于分叉链上,则出块不会被承认
NumberOfIsRoundWinner = stats.Int64("number_of_isroundwinner", "Number of is round winner", stats.UnitDimensionless)

// 类型：计数器，Tag：minerID，语义：因计算获胜证明超时导致出块失败的次数
NumberOfMiningTimeout   = stats.Int64("number_of_mining_timeout", "Number of mining failures due to compute proof timeout", stats.UnitDimensionless)
// 类型：计数器，Tag：minerID，语义：因链分叉放弃出块的次数,在分叉链上的出块没有意义,故在出块前验证到链分叉则放弃后续逻辑
NumberOfMiningChainFork = stats.Int64("number_of_mining_chain_fork", "Number of mining failures due to chain fork", stats.UnitDimensionless)
// 类型：计数器，Tag：minerID，语义：因其他错误导致出块错误的次数
NumberOfMiningError     = stats.Int64("number_of_mining_error", "Number of mining failures due to error", stats.UnitDimensionless)	
```


### rpc

```
# 调用无效RPC方法的次数
RPCInvalidMethod = stats.Int64("rpc/invalid_method", "Total number of invalid RPC methods called", stats.UnitDimensionless)
# RPC请求失败的次数
RPCRequestError  = stats.Int64("rpc/request_error", "Total number of request errors handled", stats.UnitDimensionless)
# RPC响应失败的次数
RPCResponseError = stats.Int64("rpc/response_error", "Total number of responses errors handled", stats.UnitDimensionless)
```
