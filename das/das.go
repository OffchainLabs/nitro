// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbstate/daprovider"
)

type DataAvailabilityServiceWriter interface {
	// Store requests that the message be stored until timeout (UTC time in unix epoch seconds).
	Store(ctx context.Context, message []byte, timeout uint64) (*daprovider.DataAvailabilityCertificate, error)
	fmt.Stringer
}

type DataAvailabilityServiceReader interface {
	daprovider.DASReader
	fmt.Stringer
}

type DataAvailabilityServiceHealthChecker interface {
	HealthCheck(ctx context.Context) error
}

type DataAvailabilityConfig struct {
	Enable bool `koanf:"enable"`

	RequestTimeout time.Duration `koanf:"request-timeout"`

	LocalCache CacheConfig `koanf:"local-cache"`
	RedisCache RedisConfig `koanf:"redis-cache"`

	LocalDBStorage     LocalDBStorageConfig            `koanf:"local-db-storage"`
	LocalFileStorage   LocalFileStorageConfig          `koanf:"local-file-storage"`
	S3Storage          S3StorageServiceConfig          `koanf:"s3-storage"`
	GoogleCloudStorage GoogleCloudStorageServiceConfig `koanf:"google-cloud-storage"`

	MigrateLocalDBToFileStorage bool `koanf:"migrate-local-db-to-file-storage"`

	Key KeyConfig `koanf:"key"`

	RPCAggregator  AggregatorConfig              `koanf:"rpc-aggregator"`
	RestAggregator RestfulClientAggregatorConfig `koanf:"rest-aggregator"`

	ParentChainNodeURL              string `koanf:"parent-chain-node-url"`
	ParentChainConnectionAttempts   int    `koanf:"parent-chain-connection-attempts"`
	SequencerInboxAddress           string `koanf:"sequencer-inbox-address"`
	ExtraSignatureCheckingPublicKey string `koanf:"extra-signature-checking-public-key"`

	PanicOnError             bool `koanf:"panic-on-error"`
	DisableSignatureChecking bool `koanf:"disable-signature-checking"`
}

var DefaultDataAvailabilityConfig = DataAvailabilityConfig{
	RequestTimeout:                5 * time.Second,
	Enable:                        false,
	RestAggregator:                DefaultRestfulClientAggregatorConfig,
	RPCAggregator:                 DefaultAggregatorConfig,
	ParentChainConnectionAttempts: 15,
	PanicOnError:                  false,
}

func OptionalAddressFromString(s string) (*common.Address, error) {
	if s == "none" {
		return nil, nil
	}
	if s == "" {
		return nil, errors.New("must provide address for signer or specify 'none'")
	}
	if !common.IsHexAddress(s) {
		return nil, fmt.Errorf("invalid address for signer: %v", s)
	}
	addr := common.HexToAddress(s)
	return &addr, nil
}

func DataAvailabilityConfigAddNodeOptions(prefix string, f *flag.FlagSet) {
	dataAvailabilityConfigAddOptions(prefix, f, roleNode)
}

func DataAvailabilityConfigAddDaserverOptions(prefix string, f *flag.FlagSet) {
	dataAvailabilityConfigAddOptions(prefix, f, roleDaserver)
}

type role int

const (
	roleNode role = iota
	roleDaserver
)

func dataAvailabilityConfigAddOptions(prefix string, f *flag.FlagSet, r role) {
	f.Bool(prefix+".enable", DefaultDataAvailabilityConfig.Enable, "enable Anytrust Data Availability mode")
	f.Bool(prefix+".panic-on-error", DefaultDataAvailabilityConfig.PanicOnError, "whether the Data Availability Service should fail immediately on errors (not recommended)")

	if r == roleDaserver {
		f.Bool(prefix+".disable-signature-checking", DefaultDataAvailabilityConfig.DisableSignatureChecking, "disables signature checking on Data Availability Store requests (DANGEROUS, FOR TESTING ONLY)")

		// Cache options
		CacheConfigAddOptions(prefix+".local-cache", f)
		RedisConfigAddOptions(prefix+".redis-cache", f)

		// Storage options
		LocalDBStorageConfigAddOptions(prefix+".local-db-storage", f)
		LocalFileStorageConfigAddOptions(prefix+".local-file-storage", f)
		S3ConfigAddOptions(prefix+".s3-storage", f)
		GoogleCloudConfigAddOptions(prefix+".google-cloud-storage", f)
		f.Bool(prefix+".migrate-local-db-to-file-storage", DefaultDataAvailabilityConfig.MigrateLocalDBToFileStorage, "daserver will migrate all data on startup from local-db-storage to local-file-storage, then mark local-db-storage as unusable")

		// Key config for storage
		KeyConfigAddOptions(prefix+".key", f)

		f.String(prefix+".extra-signature-checking-public-key", DefaultDataAvailabilityConfig.ExtraSignatureCheckingPublicKey, "public key to use to validate Data Availability Store requests in addition to the Sequencer's public key determined using sequencer-inbox-address, can be a file or the hex-encoded public key beginning with 0x; useful for testing")
	}
	if r == roleNode {
		// These are only for batch poster
		AggregatorConfigAddOptions(prefix+".rpc-aggregator", f)
		f.Duration(prefix+".request-timeout", DefaultDataAvailabilityConfig.RequestTimeout, "Data Availability Service timeout duration for Store requests")
	}

	// Both the Nitro node and daserver can use these options.
	RestfulClientAggregatorConfigAddOptions(prefix+".rest-aggregator", f)

	f.String(prefix+".parent-chain-node-url", DefaultDataAvailabilityConfig.ParentChainNodeURL, "URL for parent chain node, only used in standalone daserver; when running as part of a node that node's L1 configuration is used")
	f.Int(prefix+".parent-chain-connection-attempts", DefaultDataAvailabilityConfig.ParentChainConnectionAttempts, "parent chain RPC connection attempts (spaced out at least 1 second per attempt, 0 to retry infinitely), only used in standalone daserver; when running as part of a node that node's parent chain configuration is used")
	f.String(prefix+".sequencer-inbox-address", DefaultDataAvailabilityConfig.SequencerInboxAddress, "parent chain address of SequencerInbox contract")
}

func GetL1Client(ctx context.Context, maxConnectionAttempts int, l1URL string) (*ethclient.Client, error) {
	if maxConnectionAttempts <= 0 {
		maxConnectionAttempts = math.MaxInt
	}
	var l1Client *ethclient.Client
	var err error
	for i := 1; i <= maxConnectionAttempts; i++ {
		l1Client, err = ethclient.DialContext(ctx, l1URL)
		if err == nil {
			return l1Client, nil
		}
		log.Warn("error connecting to L1 from DAS", "l1URL", l1URL, "err", err)

		timer := time.NewTimer(time.Second * 1)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, errors.New("aborting startup")
		case <-timer.C:
		}
	}
	return nil, err
}
