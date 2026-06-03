// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package s3syncer

import (
	"errors"
	"fmt"

	"github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/util/s3client"
)

// Config holds the S3 configuration for syncing data.
type Config struct {
	s3client.Config `koanf:",squash"`
	Bucket          string `koanf:"bucket"`
	ObjectKey       string `koanf:"object-key"`
	ChunkSizeMB     int    `koanf:"chunk-size-mb"`
	MaxRetries      int    `koanf:"max-retries"`
	Concurrency     int    `koanf:"concurrency"`
	MaxFileSizeMB   int    `koanf:"max-file-size-mb"`
}

// ConfigAddOptions adds S3 configuration flags to the given flag set.
func ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	s3client.ConfigAddOptions(prefix, f)
	f.String(prefix+".bucket", DefaultS3Config.Bucket, "S3 bucket name")
	f.String(prefix+".object-key", "", "S3 object key (path) to the file")
	f.Int(prefix+".chunk-size-mb", DefaultS3Config.ChunkSizeMB, "S3 multipart download part size in MB")
	f.Int(prefix+".concurrency", DefaultS3Config.Concurrency, "S3 multipart download concurrency")
	f.Int(prefix+".max-retries", DefaultS3Config.MaxRetries, "maximum retries for S3 part body download")
	f.Int(prefix+".max-file-size-mb", DefaultS3Config.MaxFileSizeMB, "maximum allowed S3 object size in MB; if the object is larger, skip the download (0 disables the check)")
}

// Validate checks that required S3 configuration fields are set.
func (c *Config) Validate() error {
	if c.Bucket == "" {
		return errors.New("s3 bucket is required")
	}
	if c.Region == "" {
		return errors.New("s3 region is required")
	}
	if c.ObjectKey == "" {
		return errors.New("s3 object-key is required")
	}
	if c.MaxFileSizeMB < 0 {
		return fmt.Errorf("s3 max-file-size-mb must be >= 0, got %d", c.MaxFileSizeMB)
	}
	return nil
}

var DefaultS3Config = Config{
	ChunkSizeMB: 32,
	MaxRetries:  3,
	Concurrency: 10,
}
