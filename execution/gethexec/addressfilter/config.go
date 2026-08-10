// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/util/s3syncer"
)

type FileConfig struct {
	s3syncer.Config `koanf:",squash"`
	PollInterval    time.Duration `json:"poll-interval,omitempty" koanf:"poll-interval"`
}

var DefaultFileConfig = FileConfig{
	Config:       s3syncer.DefaultS3Config,
	PollInterval: 5 * time.Minute,
}

type Config struct {
	Files                     []FileConfig `koanf:"files"`
	FilesList                 string       `koanf:"files-list"`
	StaticList                string       `koanf:"static-list"`
	CacheSize                 int          `koanf:"cache-size"`
	AddressCheckerWorkerCount int          `koanf:"address-checker-worker-count"`
	AddressCheckerQueueSize   int          `koanf:"address-checker-queue-size"`
}

var DefaultConfig = Config{
	FilesList:                 "default",
	CacheSize:                 10000,
	AddressCheckerWorkerCount: 4,
	AddressCheckerQueueSize:   8192,
}

func ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".files-list", DefaultConfig.FilesList,
		"array of S3 hash-list file configs given as a json string, "+
			`e.g. [{"bucket":"b","object-key":"k","region":"us-east-1","download-dir":"/data/tmp","poll-interval":300000000000}]; `+
			"json keys match the "+prefix+".files config-file field names and time durations must be supplied as an integer number of nanoseconds; "+
			"omitted or zero chunk-size-mb, max-retries, concurrency and poll-interval fall back to their defaults, so max-retries cannot be set to 0")
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
func (c *FileConfig) numPreallocatedHashes() int {
	if !c.PreallocateMemory || c.MaxFileSizeMB <= 0 {
		return 0
	}
	// Compute the byte count in int64; it exceeds 32 bits for multi-GB files.
	return estimateHashCount(int64(c.MaxFileSizeMB) * bytesInMB)
}

// parseFileConfigsList decodes a files-list JSON array. Each element starts from
// DefaultFileConfig before unmarshaling so omitted fields keep their defaults
// instead of Go zero values. Unknown keys are rejected so a typo doesn't
// silently leave a field at its default, matching the strictness of the
// config-file path.
func parseFileConfigsList(list string) ([]FileConfig, error) {
	var rawEntries []json.RawMessage
	if err := json.Unmarshal([]byte(list), &rawEntries); err != nil {
		return nil, fmt.Errorf("failed to parse address-filter files-list string: %w", err)
	}
	fileConfigs := make([]FileConfig, 0, len(rawEntries))
	for i, raw := range rawEntries {
		fileConfig := DefaultFileConfig
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&fileConfig); err != nil {
			return nil, fmt.Errorf("failed to parse address-filter files-list entry %d: %w", i, err)
		}
		fileConfigs = append(fileConfigs, fileConfig)
	}
	return fileConfigs, nil
}

// applyFileDefaults fills in zero-valued fields that must be positive for the
// syncer to work. Config sources that decode into the Files slice (koanf
// config files) don't apply per-element defaults, so a zero here means the
// field was omitted.
func (c *FileConfig) applyFileDefaults() {
	if c.ChunkSizeMB == 0 {
		c.ChunkSizeMB = DefaultFileConfig.ChunkSizeMB
	}
	if c.MaxRetries == 0 {
		c.MaxRetries = DefaultFileConfig.MaxRetries
	}
	if c.Concurrency == 0 {
		c.Concurrency = DefaultFileConfig.Concurrency
	}
	if c.PollInterval == 0 {
		c.PollInterval = DefaultFileConfig.PollInterval
	}
}

func (c *Config) Validate() error {
	if len(c.Files) == 0 && c.FilesList != "default" {
		files, err := parseFileConfigsList(c.FilesList)
		if err != nil {
			return err
		}
		c.Files = files
	}

	if len(c.Files) == 0 && c.StaticList == "" {
		return errors.New("address-filter: at least one file must be configured via files or files-list, or a static list via static-list")
	}

	seen := make(map[string]struct{}, len(c.Files))
	for i := range c.Files {
		file := &c.Files[i]
		file.applyFileDefaults()
		if err := file.Config.Validate(); err != nil {
			return fmt.Errorf("address-filter.files[%d] (s3://%s/%s): %w", i, file.Bucket, file.ObjectKey, err)
		}
		if file.PollInterval <= 0 {
			return fmt.Errorf("address-filter.files[%d] (s3://%s/%s): poll-interval must be positive", i, file.Bucket, file.ObjectKey)
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
