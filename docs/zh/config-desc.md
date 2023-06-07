# 配置文件解析

&ensp;&ensp; `sophon-miner` 的配置文件默认位于 `~/.sophon-miner/config.toml`，执行命令 `sophon-miner init` 时生成。文件中以 `#` 开头的行为注释。


## 旧版本

旧版本指的是版本号 `< v1.7.0` 的版本

```toml
# 链服务监听地址
ListenAPI = "/ip4/127.0.0.1/tcp/3453"
Token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiYWRtaW4iLCJwZXJtIjoiYWRtaW4iLCJleHQiOiIifQ.RRSdeQ-c1Ei-8roAj-L-wpOr-y6PssDorbGijMPxjoc"
# 生产的区块记录方式，已废弃，由 `slash filter` 取代
BlockRecord = "cache"

# `sophon-miner` 服务监听地址
[API]
  ListenAddress = "/ip4/0.0.0.0/tcp/12308/http"
  RemoteListenAddress = ""
  Timeout = "30s"

# 事件网关服务监听地址
[Gateway]
  ListenAPI = ["/ip4/127.0.0.1/tcp/45132"]
  Token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiYWRtaW4iLCJwZXJtIjoiYWRtaW4iLCJleHQiOiIifQ.RRSdeQ-c1Ei-8roAj-L-wpOr-y6PssDorbGijMPxjoc"

# 数据库信息
[Db]
  # 矿工管理方式，已废弃，从 `venus-auth` 获取
  Type = "auth"
  # `slash filter` 模块区块存取方式
  SFType = "mysql"
  [Db.MySQL]
    Conn = "root:kuangfengjuexizhan@tcp(192.168.200.2:3308)/sophon-miner-butterfly-200-19?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s"
    MaxOpenConn = 100
    MaxIdleConn = 10
    ConnMaxLifeTime = 60
    Debug = false
  [Db.Auth]
    ListenAPI = "http://127.0.0.1:8989"
    Token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiYWRtaW4iLCJwZXJtIjoiYWRtaW4iLCJleHQiOiIifQ.RRSdeQ-c1Ei-8roAj-L-wpOr-y6PssDorbGijMPxjoc"

# Jaeger Tracing 服务信息，默认不启用
[Tracing]
  JaegerTracingEnabled = false
  JaegerEndpoint = "localhost:6831"
  ProbabilitySampler = 1.0
  ServerName = "sophon-miner"
```

## 新版本

版本号 `>= v1.7.0` 的版本.

```toml
# 等待base延迟的秒数
PropagationDelaySecs = 12
# 计算出块证明等待超时
MinerOnceTimeout = "15s"
# 选择消息API的超时(单位:秒),0-不启用
MpoolSelectDelaySecs = 0

# `sophon-miner` 服务监听地址
[API]
  ListenAddress = "/ip4/0.0.0.0/tcp/12308“
  
# 链服务监听地址
[FullNode]
  Addr = "/ip4/127.0.0.1/tcp/3453"
  Token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiY2hhaW4tc2VydmljZSIsInBlcm0iOiJhZG1pbiIsImV4dCI6IiJ9.DxlsJO-XrrdQLvJdA6wdWJxeYOhZt_kMYMHc7NdfQNw"

# 事件网关服务监听地址
[Gateway]
  ListenAPI = ["/ip4/127.0.0.1/tcp/45132"]
  Token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiY2hhaW4tc2VydmljZSIsInBlcm0iOiJhZG1pbiIsImV4dCI6IiJ9.DxlsJO-XrrdQLvJdA6wdWJxeYOhZt_kMYMHc7NdfQNw"

# 矿工管理服务监听地址
[Auth]
  Addr = "http://127.0.0.1:8989"
  Token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiY2hhaW4tc2VydmljZSIsInBlcm0iOiJhZG1pbiIsImV4dCI6IiJ9.DxlsJO-XrrdQLvJdA6wdWJxeYOhZt_kMYMHc7NdfQNw"

# `slash filter` 模块区块存取方式
[SlashFilter]
  Type = "local"
  [SlashFilter.MySQL]
    Conn = ""
    MaxOpenConn = 100
    MaxIdleConn = 10
    ConnMaxLifeTime = 60
    Debug = false

# Jaeger Tracing 服务信息，默认不启用
[Tracing]
  JaegerTracingEnabled = false
  JaegerEndpoint = "localhost:6831"
  ProbabilitySampler = 1.0
  ServerName = "sophon-miner"
```

### `Metrics` 配置项解析

`Metrics` 一份基本的配置样例如下：
```toml
[Metrics]
  # 是否开启metrics指标统计，默认为false
  Enabled = false
  
  [Metrics.Exporter]
    # 指标导出器类型，目前可选：prometheus或graphite，默认为prometheus
    Type = "prometheus"
    
    [Metrics.Exporter.Prometheus]
      # multiaddr
      EndPoint = "/ip4/0.0.0.0/tcp/12310"
      # 命名规范: "a_b_c", 不能带"-"
      Namespace = "miner" 
      # 指标注册表类型，可选：default（默认，会附带程序运行的环境指标）或 define（自定义）
      RegistryType = "define"
      # prometheus 服务路径
      Path = "/debug/metrics"
      # 上报周期，单位为 秒（s）
      ReportingPeriod = 10
      
    [Metrics.Exporter.Graphite]
      # 命名规范: "a_b_c", 不能带"-"
      Namespace = "miner" 
      # graphite exporter 收集器服务地址
      Host = "127.0.0.1"
      # graphite exporter 收集器服务监听端口
      Port = 12310
      # 上报周期，单位为 秒（s）
      ReportingPeriod = 10
```

如果选择 `Metrics.Exporter` 为 `Prometheus`, 可通过命令行快速查看指标：

```bash
 $ curl http://127.0.0.1:12310/debug/metrics
 # HELP miner_getbaseinfo_ms Duration of GetBaseInfo in miner
 # TYPE miner_getbaseinfo_ms histogram
 miner_getbaseinfo_ms_bucket{miner_id="t010938",le="100"} 50
 miner_getbaseinfo_ms_bucket{miner_id="t010938",le="200"} 51
 miner_getbaseinfo_ms_bucket{miner_id="t010938",le="400"} 51
 miner_getbaseinfo_ms_bucket{miner_id="t010938",le="600"} 51
 miner_getbaseinfo_ms_bucket{miner_id="t010938",le="800"} 51
 miner_getbaseinfo_ms_bucket{miner_id="t010938",le="1000"} 51
 miner_getbaseinfo_ms_bucket{miner_id="t010938",le="2000"} 51
 miner_getbaseinfo_ms_bucket{miner_id="t010938",le="20000"} 51
 miner_getbaseinfo_ms_bucket{miner_id="t010938",le="+Inf"} 51
 miner_getbaseinfo_ms_sum{miner_id="t010938"} 470.23516
 ... ...
```
> 如果遇到错误 `curl: (56) Recv failure: Connection reset by peer`, 请使用本机 `ip` 地址, 如下所示:
```bash
$  curl http://<ip>:12310/debug/metrics
```

### `SubmitNodes` 配置项解析

当选择消息的节点和广播区块的节点 `mpool` 中的消息不一致时, 有可能导致新出的区块无法验证,如区块中打包的消息在广播节点的 `mpool` 中不存在, 故引入`SubmitNodes`配置项, 保证区块正常上链.

默认的配置是空的,如下:
```toml
SubmitNodes = []
```

当使用了 `chain-co` 组建时,需要将对接的所有同步节点配置到 `SubmitNodes`, 参考配置如下:

```toml
[[SubmitNodes]]
  Addr = "/192.168.200.108/tcp/3453"
  Token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiY2hhaW4tc2VydmljZSIsInBlcm0iOiJhZG1pbiIsImV4dCI6IiJ9.DxlsJO-XrrdQLvJdA6wdWJxeYOhZt_kMYMHc7NdfQNw"

[[SubmitNodes]]
  Addr = "/ip4/192.168.200.107/tcp/3453"
  Token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiY2hhaW4tc2VydmljZSIsInBlcm0iOiJhZG1pbiIsImV4dCI6IiJ9.DxlsJO-XrrdQLvJdA6wdWJxeYOhZt_kMYMHc7NdfQNw"
```
