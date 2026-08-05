// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"errors"
	"time"

	"github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/util/s3syncer"
)

type Config struct {
	S3                        s3syncer.Config `koanf:"s3"`
	PollInterval              time.Duration   `koanf:"poll-interval"`
	CacheSize                 int             `koanf:"cache-size"`
	AddressCheckerWorkerCount int             `koanf:"address-checker-worker-count"`
	AddressCheckerQueueSize   int             `koanf:"address-checker-queue-size"`
}

var DefaultConfig = Config{
	S3:                        s3syncer.DefaultS3Config,
	PollInterval:              5 * time.Minute,
	CacheSize:                 10000,
	AddressCheckerWorkerCount: 4,
	AddressCheckerQueueSize:   8192,
}

func ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	s3syncer.ConfigAddOptions(prefix+".s3", f)
	f.Duration(prefix+".poll-interval", DefaultConfig.PollInterval, "interval between polling S3 for hash list updates")
	f.Int(prefix+".cache-size", DefaultConfig.CacheSize, "LRU cache size for address lookup results")
	f.Int(prefix+".address-checker-worker-count", DefaultConfig.AddressCheckerWorkerCount, "number of workers for address checker")
	f.Int(prefix+".address-checker-queue-size", DefaultConfig.AddressCheckerQueueSize, "work queue size for address checker")
}

const bytesInMB = 1024 * 1024

// minBytesPerHashEntry is a hard lower bound on the JSON size of one 32-byte hash entry: 64 hex chars plus the two
// surrounding quotes. Dividing the max file size by it yields a safe upper bound on the number of hashes.
const minBytesPerHashEntry = 66

// estimateHashCount returns a safe upper bound on the number of hashes in a
// hash-list JSON document of the given byte size.
func estimateHashCount(sizeBytes int64) int {
	if sizeBytes < 0 {
		return 0
	}
	return int(sizeBytes / minBytesPerHashEntry)
}

// numPreallocatedHashes returns how many hashes to preallocate structures for, derived from max-file-size-mb, or 0 when
// preallocation is disabled (the toggle is off or max-file-size-mb is unset).
func (c *Config) numPreallocatedHashes() int {
	if !c.S3.PreallocateMemory || c.S3.MaxFileSizeMB <= 0 {
		return 0
	}
	// Compute the byte count in int64; it exceeds 32 bits for multi-GB files.
	return estimateHashCount(int64(c.S3.MaxFileSizeMB) * bytesInMB)
}

func (c *Config) Validate() error {
	if err := c.S3.Validate(); err != nil {
		return err
	}

	if c.PollInterval <= 0 {
		return errors.New("address-filter.poll-interval must be positive")
	}

	if c.CacheSize <= 0 {
		return errors.New("address-filter.cache-size must be positive")
	}

	if c.AddressCheckerWorkerCount <= 0 {
		return errors.New("address-filter.address-checker-worker-count must be positive")
	}

	if c.AddressCheckerQueueSize <= 0 {
		return errors.New("address-filter.address-checker-queue-size must be positive")
	}

	return nil
}
