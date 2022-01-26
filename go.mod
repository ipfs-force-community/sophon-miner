module github.com/filecoin-project/venus-miner

go 1.16

require (
	contrib.go.opencensus.io/exporter/jaeger v0.2.1
	contrib.go.opencensus.io/exporter/prometheus v0.4.0
	github.com/BurntSushi/toml v0.4.1
	github.com/GeertJohan/go.rice v1.0.2
	github.com/dgraph-io/badger/v2 v2.2007.3
	github.com/dustin/go-humanize v1.0.0
	github.com/elastic/gosigar v0.14.1
	github.com/filecoin-project/filecoin-ffi v0.30.4-0.20200910194244-f640612a1a1f
	github.com/filecoin-project/go-address v0.0.6
	github.com/filecoin-project/go-bitfield v0.2.4
	github.com/filecoin-project/go-crypto v0.0.1
	github.com/filecoin-project/go-jsonrpc v0.1.5
	github.com/filecoin-project/go-paramfetch v0.0.3-0.20220111000201-e42866db1a53
	github.com/filecoin-project/go-state-types v0.1.3
	github.com/filecoin-project/specs-actors v0.9.14
	github.com/filecoin-project/specs-actors/v2 v2.3.6
	github.com/filecoin-project/specs-actors/v7 v7.0.0-rc1
	github.com/filecoin-project/venus v1.2.0-rc2
	github.com/filecoin-project/venus-wallet v1.4.0-rc1
	github.com/gbrlsnchs/jwt/v3 v3.0.1
	github.com/go-resty/resty/v2 v2.4.0
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/ipfs-force-community/venus-common-utils v0.0.0-20211122032945-eb6cab79c62a
	github.com/ipfs/go-block-format v0.0.3
	github.com/ipfs/go-cid v0.1.0
	github.com/ipfs/go-datastore v0.5.1
	github.com/ipfs/go-ds-badger2 v0.1.2
	github.com/ipfs/go-ds-leveldb v0.5.0
	github.com/ipfs/go-ds-measure v0.2.0
	github.com/ipfs/go-fs-lock v0.0.6
	github.com/ipfs/go-ipfs-blockstore v1.1.2
	github.com/ipfs/go-ipfs-util v0.0.2
	github.com/ipfs/go-log/v2 v2.4.0
	github.com/ipfs/go-metrics-interface v0.0.1
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/libp2p/go-buffer-pool v0.0.2
	github.com/libp2p/go-libp2p-core v0.13.0
	github.com/libp2p/go-libp2p-pubsub v0.6.0
	github.com/libp2p/go-libp2p-record v0.1.3
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-base32 v0.0.4
	github.com/multiformats/go-multiaddr v0.4.1
	github.com/prometheus/client_golang v1.11.0
	github.com/raulk/clock v1.1.0
	github.com/raulk/go-watchdog v1.2.0
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.0
	github.com/urfave/cli/v2 v2.3.0
	github.com/whyrusleeping/cbor-gen v0.0.0-20210713220151-be142a5ae1a8
	go.opencensus.io v0.23.0
	go.uber.org/fx v1.15.0
	go.uber.org/zap v1.19.1
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	gorm.io/driver/mysql v1.1.1
	gorm.io/gorm v1.21.12
)

replace github.com/filecoin-project/venus-miner => ./

replace github.com/golangci/golangci-lint => github.com/golangci/golangci-lint v1.18.0

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi

replace github.com/ipfs/go-ipfs-cmds => github.com/ipfs-force-community/go-ipfs-cmds v0.6.1-0.20210521090123-4587df7fa0ab
