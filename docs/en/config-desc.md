# Configuration parsing

&ensp;&ensp; The file is located in `~/.venusminer/config.toml` by default, which is generated when the command `venus-miner init` is executed. Behavior comments starting with `#` in the file.


## Old version

version number `< v1.7.0`

```toml
# venus fullnode API
ListenAPI = "/ip4/127.0.0.1/tcp/3453"
Token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiYWRtaW4iLCJwZXJtIjoiYWRtaW4iLCJleHQiOiIifQ.RRSdeQ-c1Ei-8roAj-L-wpOr-y6PssDorbGijMPxjoc"
# deprecated, replaced by `slash filter`
BlockRecord = "cache"

# deprecated, replaced by ~/.venusminer/api`
[API]
  ListenAddress = "/ip4/0.0.0.0/tcp/12308/http"
  RemoteListenAddress = ""
  Timeout = "30s"

# venus-gateway API
[Gateway]
  ListenAPI = ["/ip4/127.0.0.1/tcp/45132"]
  Token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiYWRtaW4iLCJwZXJtIjoiYWRtaW4iLCJleHQiOiIifQ.RRSdeQ-c1Ei-8roAj-L-wpOr-y6PssDorbGijMPxjoc"

[Db]
  # deprecated, replaced by ~venus-auth`
  Type = "auth"
  # `slash filter`
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

# Jaeger Tracing
[Tracing]
  JaegerTracingEnabled = false
  JaegerEndpoint = "localhost:6831"
  ProbabilitySampler = 1.0
  ServerName = "venus-miner"
```

## New version

version number `>= v1.7.0`

```toml
# venus fullnode API
[FullNode]
  Addr = "/ip4/127.0.0.1/tcp/3453"
  Token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiY2hhaW4tc2VydmljZSIsInBlcm0iOiJhZG1pbiIsImV4dCI6IiJ9.DxlsJO-XrrdQLvJdA6wdWJxeYOhZt_kMYMHc7NdfQNw"

# venus-gateway API
[Gateway]
  ListenAPI = ["/ip4/127.0.0.1/tcp/45132"]
  Token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiY2hhaW4tc2VydmljZSIsInBlcm0iOiJhZG1pbiIsImV4dCI6IiJ9.DxlsJO-XrrdQLvJdA6wdWJxeYOhZt_kMYMHc7NdfQNw"

# venus-auth API
[Auth]
  Addr = "http://127.0.0.1:8989"
  Token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiY2hhaW4tc2VydmljZSIsInBlcm0iOiJhZG1pbiIsImV4dCI6IiJ9.DxlsJO-XrrdQLvJdA6wdWJxeYOhZt_kMYMHc7NdfQNw"

# `slash filter`
[SlashFilter]
  Type = "local" # optional: local,mysql
  [SlashFilter.MySQL]
    Conn = ""
    MaxOpenConn = 100
    MaxIdleConn = 10
    ConnMaxLifeTime = 60
    Debug = false

# Jaeger Tracing
[Tracing]
  JaegerTracingEnabled = false
  JaegerEndpoint = "localhost:6831"
  ProbabilitySampler = 1.0
  ServerName = "venus-miner"
```

### Metrics Configuration parsing

A basic configuration example of `Metrics` is as follows：
```toml
[Metrics]
  # Whether to enable metrics statistics, the default is false
  Enabled = false
  
  [Metrics.Exporter]
    # Metric exporter type, currently optional: prometheus or graphite, default is prometheus
    Type = "prometheus"
    
    [Metrics.Exporter.Prometheus]
      #multiaddr
      EndPoint = "/ip4/0.0.0.0/tcp/12310"
      # Naming convention: "a_b_c", no "-"
      Namespace = "miner"
      # Indicator registry type, optional: default (with the environment indicators that the program runs) or define (custom)
      RegistryType = "define"
      # prometheus service path
      Path = "/debug/metrics"
      # Reporting period, in seconds (s)
      ReportingPeriod = 10
      
    [Metrics.Exporter.Graphite]
      # Naming convention: "a_b_c", no "-"
      Namespace = "miner"
      # graphite exporter collector service address
      Host = "127.0.0.1"
      # graphite exporter collector service listening port
      Port = 12310
      # Reporting period, in seconds (s)
      ReportingPeriod = 10
```

If you choose `Metrics.Exporter` as `Prometheus`, you can quickly view metrics through the command line：

```
 $ curl http://127.0.0.1:<port><path>
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
> If you encounter the error `curl: (56) Recv failure: Connection reset by peer`, use the native `ip` address as follows:
```
$  curl http://<ip>:<port><path>
```
