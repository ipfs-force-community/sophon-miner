## `venus-miner` 快速启动

## 实例初始化

初始化 `Repo`

`< v1.7.0` 版本:
```shell script
$ ./venus-miner init
# For nettype, choose from mainnet, debug, 2k, calibnet
--nettype=calibnet \
--auth-api=<http://VENUS_AUTH_IP:PORT> \
--token=<SHARED_ADMIN_AUTH_TOKEN> \
--gateway-api=/ip4/<VENUS_GATEWAY_IP>/tcp/PORT \
--api=/ip4/<VENUS_DAEMON_IP>/tcp/PORT \
--slash-filter local
```

`>= v1.7.0` 版本:
```shell script
$ ./venus-miner init
--api=/ip4/<VENUS_DAEMON_IP>/tcp/PORT \
--gateway-api=/ip4/<VENUS_GATEWAY_IP>/tcp/PORT \
--auth-api <http://VENUS_AUTH_IP:PORT> \
--token <SHARED_ADMIN_AUTH_TOKEN> \
--slash-filter local
```

启动 `venus-miner`

```shell script
$ nohup ./venus-miner run > miner.log 2>&1 &
```

## 矿工管理

`venus-miner` 启动时会从 `venus-auth` 中拉取矿工列表，然后预执行一遍出块流程，如果失败可用 `state` 命令查看原因：

```shell script
$ ./venus-miner address state
[
	{
		"Addr": "<MINER_ID>",
		"IsMining": true,
		"Err": null
	}
]
```

查看当前的矿工列表：

```shell script
$ ./venus-miner address list
```


`start` 或 `stop` 命令可以开始或暂停某个矿工的的出块流程，如暂停 `f01008` 的出块流程：

```shell script
$ ./venus-miner address stop f01008
```
> 注意：暂停后即使矿工有算力也不会执行出块，故在故障修复后需执行 `start` 开启出块流程。


有矿工加入或退出矿池,或矿工信息有变化时,需要重新从`venus_auth`拉取矿工列表：

```shell script
$ ./venus-miner address update
```

## 出块权统计

统计矿工在指定链高度区间内获得的出块权：

```shell script
./venus-miner winner count --epoch-start=<START_EPOCH> --epoch-end=<END_EPOCH> <MINER_ID>
```

> 这个是统计历史出块权的，不是预测未来的，所以： 1. `epoch-end` > `epoch-start`; 2. `epoch-end` 须小于当前链高度。
