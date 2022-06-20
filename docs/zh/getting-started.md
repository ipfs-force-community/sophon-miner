## venus-miner

初始化`venus-miner`。
```bash
$ ./venus-miner init
# For nettype, choose from mainnet, debug, 2k, calibnet
--nettype <NET_TYPE> \
--auth-api <http://VENUS_AUTH_IP_ADDRESS:PORT> \
--token <SHARED_ADMIN_AUTH_TOKEN> \
--gateway-api /ip4/<VENUS_GATEWAY_IP_ADDRESS>/tcp/45132 \
--api /ip4/<VENUS_DAEMON_IP_ADDRESS>/tcp/3453 \
--slash-filter local
```

启动`venus-miner`。

```bash
$ nohup ./venus-miner run > miner.log 2>&1 &
```

## miner管理

`venus-miner` 启动时会从 `venus-auth` 中拉取最新的`miner_id`列表，然后预执行一遍出块流程，如果失败会在state中体现出来，可以通过以下方式查询`miner_id`的状态。

```bash
$ ./venus-miner address state
[
	{
		"Addr": "<MINER_ID>",
		"IsMining": true,
		"Err": null
	}
]
```
如果某个`miner_id`的 Err 不是 `null`,则需要根据错误信息分析原因，并通知用户解决。


我们可以开始或暂停某个`miner_id`的出块逻辑，如启动`miner_id=f01008`的出块逻辑。

```bash
$ ./venus-miner address start f01008
```

列出所有已连接到链服务的`miner id`。

```bash
$ ./venus-miner address list
```

在`venus-miner`运行期间有`miner_id`加入或退出矿池,或有`miner_id`保存在venus-auth的信息有改变,需要重新从`venus_auth`拉取数据.

```bash
$ ./venus-miner address update
```

统计某个`miner_id`在特定链高度区间内获得的出块权

```bash
./venus-miner winner count --epoch-start=<START_EPOCH> --epoch-end=<END_EPOCH> <MINER_ID>
```

> 1. epoch-end>epoch-start; 2. epoch-end必须小于当前链高度，即这个命令是用来查询历史出块情况的，并非是预测未来出块权的。举例如下:

```bash
 ./venus-miner winner count --epoch-start=60300 --epoch-end=60345 f01008
[
        {
                "miner": "f01008",
                "totalWinCount": 7,
                "msg": "",
                "winEpochList": [
                        {
                                "epoch": 60340,
                                "winCount": 2
                        },
                        {
                                "epoch": 60329,
                                "winCount": 1
                        },
                        {
                                "epoch": 60326,
                                "winCount": 2
                        },
                        {
                                "epoch": 60339,
                                "winCount": 1
                        },
                        {
                                "epoch": 60315,
                                "winCount": 1
                        }
                ]
        }
]
```