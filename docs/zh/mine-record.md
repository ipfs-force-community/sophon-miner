## 出块记录说明

### 出块记录是什么？

出块记录是 `venus-miner` 在计算出块时，保留在数据库中的，关于计算出块过程中出现的一些关键信息以及时间信息。出块记录一方面，可以用于链服务管理者排查出块过程中存在的问题，另一方面可以为链服务的用户，提供更加详细的出块信息，方便用户进行出块相关方面的分析。 

#### 出块记录的内容
一个典型的出块记录会包含一下字段：
```json
{
    // 高度以及 miner 当下的基础信息
    "epoch": "814088",
    "miner": "t01038",
    "worker": "t3rjjw3z2rtxsplsbga4u...z4nylkbl4bbo4thh7ja",
    "miner_power": "1182013654564864",
    "network_power": "5496119577509888",

    // 是否被允许参加出块选举
    "eligible_for_mining": "true",

    // 赢票
    "win_count": "1",

    // 出块过程中产生的关键信息，例如：miner 没有获得出块权等
    "info":"",

    // 出块过程中的报错信息
    "error":"",

    // 出块过程中的起始时间戳，以及各阶段耗时
    "start": "2023-08-11 18:16:51.013248275 +0800 CST m=+67.262220230",
    "compute_election_proof": "33.231212ms",
    "compute_post_proof": "4.155131366s",
    "compute_ticket": "33.637688ms",
    "get_base_info": "8.442663ms",
    "seed": "5.689287ms",
    "select_message": "3.640518079s",
    "create_block": "35.231231ms",
    "end": "2023-08-11 18:16:58.925227964 +0800 CST m=+75.174199917",
}
```

值得注意的是，上面各个字段都不是一定会出现的，例如：`error` 字段，如果出块过程中没有报错，那么这个字段就不会出现在出块记录中。如果没有获取到出块权的话，也不会有 `win_count` 字段。

### 使用

默认情况下， `sophon-miner` 会记录七天的出块记录。可以使用 cli 查询，和请求接口查询等方式获取出块记录。

#### 通过 cli 查询出块记录

```sh
sophon-miner record query
NAME:
   sophon-miner record query - query record

USAGE:
   sophon-miner record query [command options] <miner address> <epoch>

OPTIONS:
   --limit value  query nums record of limit (default: 1)

# 举例： 查询 t01038 在 814000 ~ 814199 的出块记录
sophon-miner record query --limit 200 t01038 814000
```

#### 通过接口查询出块记录

Path： `/rpc/v0`
Method： `POST`
Header： `X-VENUS-API-NAMESPACE: miner.MinerAPI`,`Content-Type: application/json`,`Authorization: Bearer eyJhbGci...OiJIUzI`
Body：
```json
{
    "method": "Filecoin.QueryRecord",
    "params": [
        {
            "Miner": "t01038",
            "Epoch": 814000,
            "Limit": 200
        }
    ],
    "id": 0
}
```
Response:
```json
[
	{
        "epoch": "814000",
        "miner": "t01038",
        "worker": "t3rjjw3z2rtxsplsbga4u...z4nylkbl4bbo4thh7ja",
        "miner_power": "1182013654564864",
        "network_power": "5496119577509888",
        "eligible_for_mining": "true",
        "win_count": "1",
        "info":"",
        "error":"",
        "start": "2023-08-11 18:16:51.013248275 +0800 CST m=+67.262220230",
        "compute_election_proof": "33.231212ms",
        "compute_post_proof": "4.155131366s",
        "compute_ticket": "33.637688ms",
        "get_base_info": "8.442663ms",
        "seed": "5.689287ms",
        "select_message": "3.640518079s",
        "create_block": "35.231231ms",
        "end": "2023-08-11 18:16:58.925227964 +0800 CST m=+75.174199917",
    },
    ,,,,,
    {
        "epoch": "814199",
        "miner": "t01038",
        "worker": "t3rjjw3z2rtxsplsbga4u...z4nylkbl4bbo4thh7ja",
        "miner_power": "1182013654564864",
        "network_power": "5496119577509888",
        "eligible_for_mining": "true",
        "win_count": "1",
        "info":"",
        "error":"",
        "start": "2023-08-11 18:16:51.013248275 +0800 CST m=+67.262220230",
        "compute_election_proof": "33.231212ms",
        "compute_post_proof": "4.155131366s",
        "compute_ticket": "33.637688ms",
        "get_base_info": "8.442663ms",
        "seed": "5.689287ms",
        "select_message": "3.640518079s",
        "create_block": "35.231231ms",
        "end": "2023-08-11 18:16:58.925227964 +0800 CST m=+75.174199917",
    },
]
```

##### 使用示例

```sh
# 使用 curl 命令查询
curl http://gateway:45132/rpc/v0 -v -X POST -H "X-VENUS-API-NAMESPACE: miner.MinerAPI" -H "Content-Type: application/json"  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs...HTQgbqUna4"  -d '{"method": "Filecoin.QueryRecord", "params": [ { "Miner": "t01002" , "Epoch":27200 , "Limit":200 }
], "id": 0}'
```
