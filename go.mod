module github.com/filecoin-project/venus-miner

go 1.17

require (
	contrib.go.opencensus.io/exporter/jaeger v0.2.1
	contrib.go.opencensus.io/exporter/prometheus v0.4.0
	github.com/BurntSushi/toml v0.4.1
	github.com/GeertJohan/go.rice v1.0.2
	github.com/dgraph-io/badger/v2 v2.2007.3
	github.com/dustin/go-humanize v1.0.0
	github.com/elastic/gosigar v0.14.2
	github.com/filecoin-project/filecoin-ffi v0.30.4-0.20200910194244-f640612a1a1f
	github.com/filecoin-project/go-address v0.0.6
	github.com/filecoin-project/go-bitfield v0.2.4
	github.com/filecoin-project/go-crypto v0.0.1
	github.com/filecoin-project/go-jsonrpc v0.1.5
	github.com/filecoin-project/go-paramfetch v0.0.4
	github.com/filecoin-project/go-state-types v0.1.7
	github.com/filecoin-project/specs-actors v0.9.15-0.20220514164640-94e0d5e123bd
	github.com/filecoin-project/specs-actors/v2 v2.3.6
	github.com/filecoin-project/specs-actors/v7 v7.0.1-0.20220511223846-637436c27154
	github.com/filecoin-project/venus v1.3.0-rc2.0.20220531070929-8e7275c3a358
	github.com/gbrlsnchs/jwt/v3 v3.0.1
	github.com/go-resty/resty/v2 v2.4.0
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/ipfs/go-block-format v0.0.3
	github.com/ipfs/go-cid v0.1.0
	github.com/ipfs/go-datastore v0.5.1
	github.com/ipfs/go-ds-badger2 v0.1.2
	github.com/ipfs/go-ds-leveldb v0.5.0
	github.com/ipfs/go-ds-measure v0.2.0
	github.com/ipfs/go-fs-lock v0.0.7
	github.com/ipfs/go-ipfs-blockstore v1.1.2
	github.com/ipfs/go-ipfs-util v0.0.2
	github.com/ipfs/go-log/v2 v2.5.1
	github.com/ipfs/go-metrics-interface v0.0.1
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/libp2p/go-buffer-pool v0.0.2
	github.com/libp2p/go-libp2p-core v0.15.1
	github.com/libp2p/go-libp2p-pubsub v0.6.1
	github.com/libp2p/go-libp2p-record v0.1.3
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-base32 v0.0.4
	github.com/multiformats/go-multiaddr v0.5.0
	github.com/prometheus/client_golang v1.12.1
	github.com/raulk/clock v1.1.0
	github.com/raulk/go-watchdog v1.2.0
	github.com/stretchr/testify v1.7.1
	github.com/syndtr/goleveldb v1.0.0
	github.com/urfave/cli/v2 v2.3.0
	github.com/whyrusleeping/cbor-gen v0.0.0-20220323183124-98fa8256a799
	go.opencensus.io v0.23.0
	go.uber.org/fx v1.15.0
	go.uber.org/zap v1.21.0
	golang.org/x/xerrors v0.0.0-20220411194840-2f41105eb62f
	gorm.io/driver/mysql v1.1.1
	gorm.io/gorm v1.21.12
)

require (
	github.com/DataDog/zstd v1.4.1 // indirect
	github.com/GeertJohan/go.incremental v1.0.0 // indirect
	github.com/ahmetb/go-linq/v3 v3.2.0 // indirect
	github.com/akavel/rsrc v0.8.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bluele/gcache v0.0.0-20190518031135-bc40bd653833 // indirect
	github.com/btcsuite/btcd v0.22.0-beta // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/containerd/cgroups v1.0.3 // indirect
	github.com/coreos/go-systemd/v22 v22.3.2 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/daaku/go.zipexe v1.0.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgraph-io/ristretto v0.0.4-0.20210122082011-bb5d392ed82d // indirect
	github.com/dgryski/go-farm v0.0.0-20190423205320-6a90982ecee2 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/filecoin-project/go-amt-ipld/v2 v2.1.1-0.20201006184820-924ee87a1349 // indirect
	github.com/filecoin-project/go-amt-ipld/v3 v3.1.0 // indirect
	github.com/filecoin-project/go-amt-ipld/v4 v4.0.0 // indirect
	github.com/filecoin-project/go-fil-commcid v0.1.0 // indirect
	github.com/filecoin-project/go-hamt-ipld v0.1.5 // indirect
	github.com/filecoin-project/go-hamt-ipld/v2 v2.0.0 // indirect
	github.com/filecoin-project/go-hamt-ipld/v3 v3.1.0 // indirect
	github.com/filecoin-project/specs-actors/v3 v3.1.2 // indirect
	github.com/filecoin-project/specs-actors/v4 v4.0.2 // indirect
	github.com/filecoin-project/specs-actors/v5 v5.0.6-0.20220514165557-0b29a778685b // indirect
	github.com/filecoin-project/specs-actors/v6 v6.0.2-0.20220511204807-569c6d12432b // indirect
	github.com/filecoin-project/specs-actors/v8 v8.0.0-20220422153930-0afe155bfffa // indirect
	github.com/filecoin-project/specs-storage v0.4.0 // indirect
	github.com/fxamacker/cbor/v2 v2.4.0 // indirect
	github.com/go-kit/log v0.2.0 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/go-sql-driver/mysql v1.6.0 // indirect
	github.com/godbus/dbus/v5 v5.0.6 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.3 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/ipfs/bbloom v0.0.4 // indirect
	github.com/ipfs/go-blockservice v0.2.1 // indirect
	github.com/ipfs/go-ipfs-ds-help v1.1.0 // indirect
	github.com/ipfs/go-ipfs-exchange-interface v0.1.0 // indirect
	github.com/ipfs/go-ipld-cbor v0.0.6 // indirect
	github.com/ipfs/go-ipld-format v0.2.0 // indirect
	github.com/ipfs/go-ipld-legacy v0.1.1 // indirect
	github.com/ipfs/go-log v1.0.5 // indirect
	github.com/ipfs/go-merkledag v0.5.1 // indirect
	github.com/ipfs/go-verifcid v0.0.1 // indirect
	github.com/ipld/go-car v0.3.3 // indirect
	github.com/ipld/go-codec-dagpb v1.3.2 // indirect
	github.com/ipld/go-ipld-prime v0.16.0 // indirect
	github.com/ipsn/go-secp256k1 v0.0.0-20180726113642-9d62b9f0bc52 // indirect
	github.com/jbenet/goprocess v0.1.4 // indirect
	github.com/jessevdk/go-flags v1.4.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.2 // indirect
	github.com/klauspost/cpuid/v2 v2.0.9 // indirect
	github.com/libp2p/go-flow-metrics v0.0.3 // indirect
	github.com/libp2p/go-libp2p-discovery v0.6.0 // indirect
	github.com/libp2p/go-libp2p-peerstore v0.6.0 // indirect
	github.com/libp2p/go-msgio v0.1.0 // indirect
	github.com/libp2p/go-openssl v0.0.7 // indirect
	github.com/magefile/mage v1.11.0 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-runewidth v0.0.10 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/minio/sha256-simd v1.0.0 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/multiformats/go-base36 v0.1.0 // indirect
	github.com/multiformats/go-multiaddr-fmt v0.1.0 // indirect
	github.com/multiformats/go-multibase v0.0.3 // indirect
	github.com/multiformats/go-multicodec v0.4.1 // indirect
	github.com/multiformats/go-multihash v0.1.0 // indirect
	github.com/multiformats/go-varint v0.0.6 // indirect
	github.com/nkovacs/streamquote v1.0.0 // indirect
	github.com/opencontainers/runtime-spec v1.0.3-0.20210326190908-1c3f411f0417 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/polydawn/refmt v0.0.0-20201211092308-30ac6d18308e // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/prometheus/statsd_exporter v0.21.0 // indirect
	github.com/rivo/uniseg v0.1.0 // indirect
	github.com/russross/blackfriday/v2 v2.0.1 // indirect
	github.com/shurcooL/sanitized_anchor_name v1.0.0 // indirect
	github.com/spacemonkeygo/spacelog v0.0.0-20180420211403-2296661a0572 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/uber/jaeger-client-go v2.25.0+incompatible // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.1.0 // indirect
	github.com/warpfork/go-testmark v0.9.0 // indirect
	github.com/whyrusleeping/timecache v0.0.0-20160911033111-cfcb2f1abfee // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/dig v1.13.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	go4.org v0.0.0-20200411211856-f5505b9728dd // indirect
	golang.org/x/crypto v0.0.0-20220411220226-7b82a4e95df4 // indirect
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616 // indirect
	golang.org/x/net v0.0.0-20220418201149-a630d4f3e7a2 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/sys v0.0.0-20220412211240-33da011f77ad // indirect
	golang.org/x/tools v0.1.10 // indirect
	google.golang.org/api v0.43.0 // indirect
	google.golang.org/grpc v1.45.0 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
	gopkg.in/cheggaaa/pb.v1 v1.0.28 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	lukechampine.com/blake3 v1.1.7 // indirect
)

replace github.com/filecoin-project/venus-miner => ./

replace github.com/golangci/golangci-lint => github.com/golangci/golangci-lint v1.18.0

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi
