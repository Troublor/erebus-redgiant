module github.com/Troublor/erebus-redgiant

go 1.19

require github.com/Troublor/erebus/troubeth v0.0.1

replace (
	github.com/Troublor/erebus/troubeth => ./troubeth
	github.com/ledgerwatch/secp256k1 => ./secp256k1
)

require (
	github.com/Workiva/go-datastructures v1.0.53
	github.com/allegro/bigcache v1.2.1
	github.com/ethereum/go-ethereum v1.10.23
	github.com/goccy/go-graphviz v0.0.9
	github.com/golang/mock v1.6.0
	github.com/holiman/uint256 v1.2.1
	github.com/ledgerwatch/erigon v1.9.7-0.20220905043355-6bfd2c3472a3
	github.com/ledgerwatch/erigon-lib v0.0.0-20220912053753-1b5dd96e17df
	github.com/ledgerwatch/log/v3 v3.4.1
	github.com/onsi/ginkgo/v2 v2.1.6
	github.com/onsi/gomega v1.20.2
	github.com/panjf2000/ants/v2 v2.5.0
	github.com/rs/zerolog v1.26.1
	github.com/samber/lo v1.35.0
	github.com/spf13/cobra v1.5.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.10.1
	github.com/status-im/keycard-go v0.0.0-20190316090335-8537d3370df4
	go.mongodb.org/mongo-driver v1.9.0
	golang.org/x/crypto v0.0.0-20220829220503-c86fa9a7ed90
	golang.org/x/sync v0.0.0-20220722155255-886fb9371eb4
	gonum.org/v1/gonum v0.11.0
)

require (
	github.com/RoaringBitmap/roaring v1.2.1 // indirect
	github.com/StackExchange/wmi v1.2.1 // indirect
	github.com/VictoriaMetrics/fastcache v1.10.0 // indirect
	github.com/VictoriaMetrics/metrics v1.22.2 // indirect
	github.com/benesch/cgosymbolizer v0.0.0-20190515212042-bec6fe6e597b // indirect
	github.com/bits-and-blooms/bitset v1.2.2 // indirect
	github.com/btcsuite/btcd v0.22.0-beta // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.2.1 // indirect
	github.com/c2h5oh/datasize v0.0.0-20220606134207-859f65c6625b // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/crate-crypto/go-ipa v0.0.0-20220523130400-f11357ae11c7 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/deckarep/golang-set v1.8.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.1.0 // indirect
	github.com/edsrzf/mmap-go v1.1.0 // indirect
	github.com/emicklei/dot v1.0.0 // indirect
	github.com/fjl/gencodec v0.0.0-20220412091415-8bb9e558978c // indirect
	github.com/flanglet/kanzi-go v1.9.1-0.20211212184056-72dda96261ee // indirect
	github.com/fogleman/gg v1.3.0 // indirect
	github.com/fsnotify/fsnotify v1.5.4 // indirect
	github.com/garslo/gogen v0.0.0-20170306192744-1d203ffc1f61 // indirect
	github.com/gballet/go-verkle v0.0.0-20220722103930-acd34254ebff // indirect
	github.com/go-kit/kit v0.10.0 // indirect
	github.com/go-logfmt/logfmt v0.5.0 // indirect
	github.com/go-ole/go-ole v1.2.5 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/goccy/go-json v0.9.7 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.4.1 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/btree v1.1.2 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0 // indirect
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/holiman/bloomfilter/v2 v2.0.3 // indirect
	github.com/huin/goupnp v1.0.3 // indirect
	github.com/ianlancetaylor/cgosymbolizer v0.0.0-20220405231054-a1ae3e4bba26 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jackpal/go-nat-pmp v1.0.2 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kevinburke/go-bindata v3.21.0+incompatible // indirect
	github.com/klauspost/compress v1.13.6 // indirect
	github.com/ledgerwatch/secp256k1 v1.0.0 // indirect
	github.com/magiconair/properties v1.8.6 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/mitchellh/mapstructure v1.4.3 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pion/stun v0.3.5 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/tsdb v0.7.1 // indirect
	github.com/quasilyte/go-ruleguard/dsl v0.3.21 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/shirou/gopsutil v3.21.4-0.20210419000835-c7a38de76ee5+incompatible // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/spf13/afero v1.6.0 // indirect
	github.com/spf13/cast v1.4.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/subosito/gotenv v1.2.0 // indirect
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7 // indirect
	github.com/tendermint/go-amino v0.14.1 // indirect
	github.com/tendermint/tendermint v0.31.11 // indirect
	github.com/tklauser/go-sysconf v0.3.9 // indirect
	github.com/tklauser/numcpus v0.3.0 // indirect
	github.com/torquem-ch/mdbx-go v0.25.3 // indirect
	github.com/ugorji/go/codec v1.1.13 // indirect
	github.com/ugorji/go/codec/codecgen v1.1.13 // indirect
	github.com/valyala/fastjson v1.6.3 // indirect
	github.com/valyala/fastrand v1.1.0 // indirect
	github.com/valyala/histogram v1.2.0 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.0.2 // indirect
	github.com/xdg-go/stringprep v1.0.2 // indirect
	github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d // indirect
	go.uber.org/atomic v1.10.0 // indirect
	golang.org/x/exp v0.0.0-20220722155223-a9213eeb770e // indirect
	golang.org/x/image v0.0.0-20220302094943-723b81ca9867 // indirect
	golang.org/x/mod v0.6.0-dev.0.20220419223038-86c51ed26bb4 // indirect
	golang.org/x/net v0.0.0-20220909164309-bea034e7d591 // indirect
	golang.org/x/sys v0.0.0-20220909162455-aba9fc2a8ff2 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/tools v0.1.12 // indirect
	google.golang.org/genproto v0.0.0-20211208223120-3a66f561d7aa // indirect
	google.golang.org/grpc v1.48.0 // indirect
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.2.0 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/ini.v1 v1.66.2 // indirect
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
