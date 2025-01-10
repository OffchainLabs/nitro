module github.com/offchainlabs/nitro

go 1.23

replace github.com/VictoriaMetrics/fastcache => ./fastcache

replace github.com/ethereum/go-ethereum => ./go-ethereum

replace github.com/offchainlabs/bold => ./bold

require (
	cloud.google.com/go/storage v1.43.0
	github.com/EspressoSystems/espresso-sequencer-go v0.0.29
	github.com/Knetic/govaluate v3.0.1-0.20171022003610-9aa49832a739+incompatible
	github.com/Shopify/toxiproxy v2.1.4+incompatible
	github.com/alicebob/miniredis/v2 v2.32.1
	github.com/andybalholm/brotli v1.0.4
	github.com/aws/aws-sdk-go-v2 v1.31.0
	github.com/aws/aws-sdk-go-v2/config v1.27.40
	github.com/aws/aws-sdk-go-v2/credentials v1.17.38
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.17.27
	github.com/aws/aws-sdk-go-v2/service/s3 v1.64.1
	github.com/cavaliergopher/grab/v3 v3.0.1
	github.com/cockroachdb/pebble v1.1.0
	github.com/codeclysm/extract/v3 v3.0.2
	github.com/dgraph-io/badger/v4 v4.2.0
	github.com/enescakir/emoji v1.0.0
	github.com/ethereum/go-ethereum v1.13.15
	github.com/fatih/structtag v1.2.0
	github.com/gdamore/tcell/v2 v2.7.1
	github.com/gobwas/httphead v0.1.0
	github.com/gobwas/ws v1.2.1
	github.com/gobwas/ws-examples v0.0.0-20190625122829-a9e8908d9484
	github.com/google/btree v1.1.2
	github.com/google/go-cmp v0.6.0
	github.com/google/uuid v1.6.0
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/holiman/uint256 v1.2.4
	github.com/knadh/koanf v1.4.0
	github.com/mailru/easygo v0.0.0-20190618140210-3c14a0dc985f
	github.com/mitchellh/mapstructure v1.4.1
	github.com/offchainlabs/bold v0.0.0-00010101000000-000000000000
	github.com/pkg/errors v0.9.1
	github.com/r3labs/diff/v3 v3.0.1
	github.com/redis/go-redis/v9 v9.6.1
	github.com/rivo/tview v0.0.0-20240307173318-e804876934a1
	github.com/spf13/pflag v1.0.5
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
	github.com/wealdtech/go-merkletree v1.0.0
	golang.org/x/crypto v0.24.0
	golang.org/x/sys v0.21.0
	golang.org/x/term v0.21.0
	golang.org/x/tools v0.21.1-0.20240508182429-e35e4ccd0d2d
	google.golang.org/api v0.187.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
)

require (
	cloud.google.com/go v0.115.0 // indirect
	cloud.google.com/go/auth v0.6.1 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.2 // indirect
	cloud.google.com/go/compute/metadata v0.3.0 // indirect
	cloud.google.com/go/iam v1.1.8 // indirect
	github.com/ccoveille/go-safecast v1.1.0
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/googleapis/gax-go/v2 v2.12.5 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/testify v1.9.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.49.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.49.0 // indirect
	go.opentelemetry.io/otel v1.24.0 // indirect
	go.opentelemetry.io/otel/metric v1.24.0 // indirect
	go.opentelemetry.io/otel/trace v1.24.0 // indirect
	golang.org/x/exp v0.0.0-20231110203233-9a3e6036ecaa // indirect
	google.golang.org/genproto v0.0.0-20240624140628-dc46fd24d27d // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240617180043-68d350f18fd4 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240624140628-dc46fd24d27d // indirect
	google.golang.org/grpc v1.64.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require (
	github.com/DataDog/zstd v1.5.2 // indirect
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/VictoriaMetrics/fastcache v1.12.1 // indirect
	github.com/alicebob/gopher-json v0.0.0-20200520072559-a9ecdc9d1d3a // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.5 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.14 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.18 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.18 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.18 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.11.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.3.20 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.11.20 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.17.18 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.23.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.27.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.31.4 // indirect
	github.com/aws/smithy-go v1.22.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.10.0 // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.3.2 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cockroachdb/errors v1.11.1 // indirect
	github.com/cockroachdb/logtags v0.0.0-20230118201751-21c54148d20b // indirect
	github.com/cockroachdb/redact v1.1.5 // indirect
	github.com/cockroachdb/tokenbucket v0.0.0-20230807174530-cc333fc44b06 // indirect
	github.com/consensys/bavard v0.1.13 // indirect
	github.com/consensys/gnark-crypto v0.12.1 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/crate-crypto/go-ipa v0.0.0-20231025140028-3c0104f4b233 // indirect
	github.com/crate-crypto/go-kzg-4844 v1.0.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/deckarep/golang-set/v2 v2.1.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.2.0 // indirect
	github.com/dgraph-io/ristretto v0.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/dlclark/regexp2 v1.7.0 // indirect
	github.com/dop251/goja v0.0.0-20230806174421-c933cf95e127 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/ethereum/c-kzg-4844 v1.0.0 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/gammazero/deque v0.2.1 // indirect
	github.com/gballet/go-libpcsclite v0.0.0-20191108122812-4678299bea08 // indirect
	github.com/gballet/go-verkle v0.1.1-0.20231031103413-a67434b50f46 // indirect
	github.com/gdamore/encoding v1.0.0 // indirect
	github.com/getsentry/sentry-go v0.18.0 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gofrs/flock v0.8.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.0 // indirect
	github.com/golang/glog v1.2.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v0.0.5-0.20220116011046-fa5810519dcb // indirect
	github.com/google/flatbuffers v1.12.1 // indirect
	github.com/google/go-github/v62 v62.0.0
	github.com/google/pprof v0.0.0-20231023181126-ff6d637d2a7b // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/graph-gophers/graphql-go v1.3.0 // indirect
	github.com/h2non/filetype v1.0.6 // indirect
	github.com/hashicorp/go-bexpr v0.1.10 // indirect
	github.com/holiman/billy v0.0.0-20240216141850-2abb0c79d3c4 // indirect
	github.com/holiman/bloomfilter/v2 v2.0.3 // indirect
	github.com/huin/goupnp v1.3.0 // indirect
	github.com/jackpal/go-nat-pmp v1.0.2 // indirect
	github.com/jmoiron/sqlx v1.3.5 // indirect
	github.com/juju/errors v0.0.0-20181118221551-089d3ea4e4d5 // indirect
	github.com/juju/loggo v0.0.0-20180524022052-584905176618 // indirect
	github.com/klauspost/compress v1.17.2 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/mattn/go-sqlite3 v1.14.6 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/pointerstructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/mmcloughlin/addchain v0.4.0 // indirect
	github.com/onsi/gomega v1.27.4 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/prometheus/client_golang v1.14.0 // indirect
	github.com/prometheus/client_model v0.4.0 // indirect
	github.com/prometheus/common v0.39.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/rhnvrm/simples3 v0.6.1 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rogpeppe/go-internal v1.9.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/shirou/gopsutil v3.21.11+incompatible // indirect
	github.com/status-im/keycard-go v0.2.0 // indirect
	github.com/supranational/blst v0.3.11 // indirect
	github.com/urfave/cli/v2 v2.25.7 // indirect
	github.com/vmihailenco/msgpack/v5 v5.3.5 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/mod v0.17.0 // indirect
	golang.org/x/net v0.26.0 // indirect
	golang.org/x/oauth2 v0.22.0
	golang.org/x/sync v0.8.0
	golang.org/x/text v0.16.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	rsc.io/tmplfunc v0.0.3 // indirect
)

require (
	github.com/benbjohnson/clock v1.3.5 // indirect
	github.com/btcsuite/btcd v0.22.1 // indirect
	github.com/btcsuite/btcd/chaincfg/chainhash v1.0.1 // indirect
	github.com/btcsuite/btcutil v1.0.3-0.20201208143702-a53e38424cce // indirect
	github.com/miguelmota/go-ethereum-hdwallet v0.1.2 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/rs/cors v1.7.0 // indirect
	github.com/sigurn/crc8 v0.0.0-20220107193325-2243fe600f9f // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	github.com/tyler-smith/go-bip39 v1.1.0 // indirect
)
