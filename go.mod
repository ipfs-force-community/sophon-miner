module github.com/filecoin-project/venus-miner

go 1.15

require (
	contrib.go.opencensus.io/exporter/jaeger v0.1.0
	contrib.go.opencensus.io/exporter/prometheus v0.1.0
	github.com/BurntSushi/toml v0.3.1
	github.com/GeertJohan/go.rice v1.0.2
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/dgraph-io/badger/v2 v2.2007.2
	github.com/dgraph-io/ristretto v0.0.4-0.20210122082011-bb5d392ed82d // indirect
	github.com/docker/go-units v0.4.0
	github.com/dustin/go-humanize v1.0.0
	github.com/elastic/gosigar v0.12.0
	github.com/fatih/color v1.9.0 // indirect
	github.com/filecoin-project/filecoin-ffi v0.30.4-0.20200910194244-f640612a1a1f
	github.com/filecoin-project/go-address v0.0.5
	github.com/filecoin-project/go-bitfield v0.2.4
	github.com/filecoin-project/go-commp-utils v0.1.0 // indirect
	github.com/filecoin-project/go-crypto v0.0.0-20191218222705-effae4ea9f03
	github.com/filecoin-project/go-data-transfer v1.4.3
	github.com/filecoin-project/go-fil-markets v1.2.5 // indirect
	github.com/filecoin-project/go-jsonrpc v0.1.4-0.20210217175800-45ea43ac2bec
	github.com/filecoin-project/go-paramfetch v0.0.2-0.20210330140417-936748d3f5ec
	github.com/filecoin-project/go-state-types v0.1.1-0.20210506134452-99b279731c48
	github.com/filecoin-project/go-statestore v0.1.1 // indirect
	github.com/filecoin-project/specs-actors v0.9.13
	github.com/filecoin-project/specs-actors/v2 v2.3.5-0.20210114162132-5b58b773f4fb
	github.com/filecoin-project/specs-actors/v3 v3.1.0
	github.com/filecoin-project/specs-actors/v4 v4.0.0
	github.com/filecoin-project/specs-actors/v5 v5.0.0-20210609212542-73e0409ac77c
	github.com/filecoin-project/venus-wallet v1.1.0
	github.com/gbrlsnchs/jwt/v3 v3.0.0
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/go-resty/resty/v2 v2.4.0
	github.com/golang/protobuf v1.5.1 // indirect
	github.com/golang/snappy v0.0.2-0.20190904063534-ff6b7dc882cf // indirect
	github.com/google/uuid v1.2.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/ipfs/go-block-format v0.0.3
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-datastore v0.4.5
	github.com/ipfs/go-ds-badger2 v0.1.1-0.20200708190120-187fc06f714e
	github.com/ipfs/go-ds-leveldb v0.4.2
	github.com/ipfs/go-ds-measure v0.1.0
	github.com/ipfs/go-fs-lock v0.0.6
	github.com/ipfs/go-ipfs-blockstore v1.0.3
	github.com/ipfs/go-ipfs-util v0.0.2
	github.com/ipfs/go-ipld-cbor v0.0.5
	github.com/ipfs/go-log v1.0.4
	github.com/ipfs/go-log/v2 v2.1.3
	github.com/ipfs/go-metrics-interface v0.0.1
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/libp2p/go-buffer-pool v0.0.2
	github.com/libp2p/go-libp2p-core v0.7.0
	github.com/libp2p/go-libp2p-pubsub v0.4.2-0.20210212194758-6c1addf493eb
	github.com/libp2p/go-libp2p-record v0.1.3
	github.com/magefile/mage v1.11.0 // indirect
	github.com/mattn/go-colorable v0.1.6 // indirect
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-base32 v0.0.3
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/multiformats/go-multihash v0.0.14
	github.com/prometheus/common v0.25.0
	github.com/raulk/clock v1.1.0
	github.com/raulk/go-watchdog v1.0.1
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.0
	github.com/urfave/cli/v2 v2.3.0
	github.com/whyrusleeping/bencher v0.0.0-20190829221104-bb6607aa8bba
	github.com/whyrusleeping/cbor-gen v0.0.0-20210219115102-f37d292932f2
	github.com/xorcare/golden v0.6.1-0.20191112154924-b87f686d7542
	go.opencensus.io v0.22.5
	go.uber.org/fx v1.9.0
	go.uber.org/zap v1.16.0
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2 // indirect
	golang.org/x/sys v0.0.0-20210324051608-47abb6519492 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gorm.io/driver/mysql v1.0.5
	gorm.io/gorm v1.21.3
)

replace github.com/filecoin-project/venus-miner => ./

replace github.com/golangci/golangci-lint => github.com/golangci/golangci-lint v1.18.0

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi
