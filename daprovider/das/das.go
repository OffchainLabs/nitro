// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package das

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

type DataAvailabilityServiceHealthChecker interface {
	HealthCheck(ctx context.Context) error
}

// This specifically refers to AnyTrust DA Config and will be moved/renamed in future.
type DataAvailabilityConfig struct {
	Enable bool `koanf:"enable"`

	RequestTimeout time.Duration `koanf:"request-timeout"`

	LocalCache CacheConfig `koanf:"local-cache"`
	RedisCache RedisConfig `koanf:"redis-cache"`

	LocalFileStorage   LocalFileStorageConfig          `koanf:"local-file-storage"`
	S3Storage          S3StorageServiceConfig          `koanf:"s3-storage"`
	GoogleCloudStorage GoogleCloudStorageServiceConfig `koanf:"google-cloud-storage"`

	Key KeyConfig `koanf:"key"`

	RPCAggregator  AggregatorConfig              `koanf:"rpc-aggregator"`
	RestAggregator RestfulClientAggregatorConfig `koanf:"rest-aggregator"`

	ExtraSignatureCheckingPublicKey string `koanf:"extra-signature-checking-public-key"`

	PanicOnError             bool `koanf:"panic-on-error"`
	DisableSignatureChecking bool `koanf:"disable-signature-checking"`
}

var DefaultDataAvailabilityConfig = DataAvailabilityConfig{
	RequestTimeout: 5 * time.Second,
	Enable:         false,
	RestAggregator: DefaultRestfulClientAggregatorConfig,
	RPCAggregator:  DefaultAggregatorConfig,
	PanicOnError:   false,
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

func DataAvailabilityConfigAddNodeOptions(prefix string, f *pflag.FlagSet) {
	dataAvailabilityConfigAddOptions(prefix, f, roleNode)
}

func DataAvailabilityConfigAddDaserverOptions(prefix string, f *pflag.FlagSet) {
	dataAvailabilityConfigAddOptions(prefix, f, roleDaserver)
}

type role int

const (
	roleNode role = iota
	roleDaserver
)

func dataAvailabilityConfigAddOptions(prefix string, f *pflag.FlagSet, r role) {
	f.Bool(prefix+".enable", DefaultDataAvailabilityConfig.Enable, "enable Anytrust Data Availability mode")
	f.Bool(prefix+".panic-on-error", DefaultDataAvailabilityConfig.PanicOnError, "whether the Data Availability Service should fail immediately on errors (not recommended)")

	if r == roleDaserver {
		f.Bool(prefix+".disable-signature-checking", DefaultDataAvailabilityConfig.DisableSignatureChecking, "disables signature checking on Data Availability Store requests (DANGEROUS, FOR TESTING ONLY)")

		// Cache options
		CacheConfigAddOptions(prefix+".local-cache", f)
		RedisConfigAddOptions(prefix+".redis-cache", f)

		// Storage options
		LocalFileStorageConfigAddOptions(prefix+".local-file-storage", f)
		S3ConfigAddOptions(prefix+".s3-storage", f)
		GoogleCloudConfigAddOptions(prefix+".google-cloud-storage", f)

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
