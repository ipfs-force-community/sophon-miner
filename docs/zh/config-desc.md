# 配置文件解析

&ensp;&ensp; `venus-miner` 的配置文件默认位于 `~/.venusminer/config.toml`，执行命令 `venus-miner init` 时生成。文件中以 `#` 开头的行为注释。


## 旧版本

旧版本指的是版本号 `< v1.7.0` 的版本

```
# 链服务监听地址
ListenAPI = "/ip4/127.0.0.1/tcp/3453"
Token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiYWRtaW4iLCJwZXJtIjoiYWRtaW4iLCJleHQiOiIifQ.RRSdeQ-c1Ei-8roAj-L-wpOr-y6PssDorbGijMPxjoc"
# 生产的区块记录方式，已废弃，由 `slash filter` 取代
BlockRecord = "cache"

# `venus-miner` 服务监听地址，已废弃，由~/.venusminer/api` 取代
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
    Conn = "root:kuangfengjuexizhan@tcp(192.168.200.2:3308)/venus-miner-butterfly-200-19?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s"
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
  ServerName = "venus-miner"
```

## 新版本

版本号 `>= v1.7.0` 的版本.

```
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
  ServerName = "venus-miner"
```
