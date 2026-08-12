// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/util/s3syncer"
)

type FileConfig struct {
	s3syncer.Config `koanf:",squash"`
	PollInterval    time.Duration `koanf:"poll-interval"`
	// MinBytesPerHashEntry is the assumed minimum JSON size of one hash-list
	// entry, used to size memory preallocation from max-file-size-mb. The
	// default is the smallest entry any scheme allows (a quoted plaintext
	// address); lists known to hold only sha256 entries can raise it to 66 (a
	// quoted 64-hex hash) to avoid preallocating for entries the list can
	// never contain.
	MinBytesPerHashEntry int `koanf:"min-bytes-per-hash-entry"`
}

var DefaultFileConfig = FileConfig{
	Config:               s3syncer.DefaultS3Config,
	PollInterval:         5 * time.Minute,
	MinBytesPerHashEntry: minBytesPerHashEntry,
}

type Config struct {
	// Files has no dedicated command-line flag (pflag cannot express an array
	// of structs); configure it in a config file or via --conf.string.
	Files                     []FileConfig `koanf:"files"`
	StaticList                string       `koanf:"static-list"`
	CacheSize                 int          `koanf:"cache-size"`
	AddressCheckerWorkerCount int          `koanf:"address-checker-worker-count"`
	AddressCheckerQueueSize   int          `koanf:"address-checker-queue-size"`
}

var DefaultConfig = Config{
	CacheSize:                 10000,
	AddressCheckerWorkerCount: 4,
	AddressCheckerQueueSize:   8192,
}

func ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".static-list", DefaultConfig.StaticList,
		"hash-list JSON document given inline as a json string, with the same schema as the S3 hash-list files, "+
			`e.g. {"id":"<uuid>","salt":"<uuid>","hashing_scheme":"sha256-stringinput|sha256-rawbytesinput|plaintext","hashes":["0x...."]}; `+
			"applied in addition to any S3 files (an address is filtered if it appears in any list) and fixed for the lifetime of the node; "+
			"when set, configuring S3 files becomes optional")
	f.Int(prefix+".cache-size", DefaultConfig.CacheSize, "LRU cache size for address lookup results")
	f.Int(prefix+".address-checker-worker-count", DefaultConfig.AddressCheckerWorkerCount, "number of workers for address checker")
	f.Int(prefix+".address-checker-queue-size", DefaultConfig.AddressCheckerQueueSize, "work queue size for address checker")
}

const bytesInMB = 1024 * 1024

// minBytesPerHashEntry is a hard lower bound on the JSON size of one entry across all schemes: a plaintext address
// without "0x" is 40 hex chars plus the two surrounding quotes. Dividing the max file size by it yields a safe upper
// bound on the number of hashes; for sha256 lists (66+ bytes per entry) it overestimates, which only costs extra
// preallocated bucket memory, never correctness.
const minBytesPerHashEntry = 42

func estimateHashCount(sizeBytes int64, minBytesPerEntry int) int {
	if sizeBytes < 0 {
		return 0
	}
	return int(sizeBytes / int64(minBytesPerEntry))
}

func (c *FileConfig) numPreallocatedHashes() int {
	if c.DisablePreallocateMemory || c.MaxFileSizeMB <= 0 {
		return 0
	}
	// Compute the byte count in int64; it exceeds 32 bits for multi-GB files.
	return estimateHashCount(int64(c.MaxFileSizeMB)*bytesInMB, c.MinBytesPerHashEntry)
}

// withDefaults returns a copy with zero-valued fields backfilled from
// DefaultFileConfig. Koanf doesn't apply per-element defaults when decoding
// into the Files slice (no flags are registered for its keys), so a zero is
// treated as an omitted field.
func (c *FileConfig) withDefaults() FileConfig {
	file := *c
	if file.ChunkSizeMB == 0 {
		file.ChunkSizeMB = DefaultFileConfig.ChunkSizeMB
	}
	if file.MaxRetries == 0 {
		file.MaxRetries = DefaultFileConfig.MaxRetries
	}
	if file.Concurrency == 0 {
		file.Concurrency = DefaultFileConfig.Concurrency
	}
	if file.PollInterval == 0 {
		file.PollInterval = DefaultFileConfig.PollInterval
	}
	if file.MinBytesPerHashEntry == 0 {
		file.MinBytesPerHashEntry = DefaultFileConfig.MinBytesPerHashEntry
	}
	return file
}

func (c *Config) Validate() error {
	if len(c.Files) == 0 && c.StaticList == "" {
		return errors.New("address-filter: at least one file must be configured via files, or a static list via static-list")
	}

	seen := make(map[string]struct{}, len(c.Files))
	for i := range c.Files {
		file := c.Files[i].withDefaults()
		if err := file.Config.Validate(); err != nil {
			return fmt.Errorf("address-filter.files[%d] (s3://%s/%s): %w", i, file.Bucket, file.ObjectKey, err)
		}
		if file.PollInterval <= 0 {
			return fmt.Errorf("address-filter.files[%d] (s3://%s/%s): poll-interval must be positive", i, file.Bucket, file.ObjectKey)
		}
		if file.MinBytesPerHashEntry <= 0 {
			return fmt.Errorf("address-filter.files[%d] (s3://%s/%s): min-bytes-per-hash-entry must be positive", i, file.Bucket, file.ObjectKey)
		}
		key := file.Bucket + "\x00" + file.ObjectKey
		if _, dup := seen[key]; dup {
			return fmt.Errorf("address-filter.files[%d]: duplicate entry for s3://%s/%s", i, file.Bucket, file.ObjectKey)
		}
		seen[key] = struct{}{}
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
