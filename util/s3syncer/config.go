// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package s3syncer

import (
	"errors"

	"github.com/spf13/pflag"
)

// Config holds the S3 configuration for syncing data.
type Config struct {
	Bucket      string `koanf:"bucket"`
	Region      string `koanf:"region"`
	ObjectKey   string `koanf:"object-key"`
	AccessKey   string `koanf:"access-key"`
	SecretKey   string `koanf:"secret-key"`
	ChunkSizeMB int    `koanf:"chunk-size-mb"`
	MaxRetries  int    `koanf:"max-retries"`
	Concurrency int    `koanf:"concurrency"`
}

// ConfigAddOptions adds S3 configuration flags to the given flag set.
func ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".bucket", "", "S3 bucket name")
	f.String(prefix+".region", "", "AWS region of the S3 bucket")
	f.String(prefix+".access-key", "", "AWS access key for S3 (optional, uses default credentials if "+
		"not provided which check for credentials in specific order like env variables, shared credentials, etc.)")
	f.String(prefix+".secret-key", "", "AWS secret key for S3 (optional, uses default credentials if "+
		"not provided which check for credentials in specific order like env variables, shared credentials, etc.)")
	f.String(prefix+".object-key", "", "S3 object key (path) to the file")
	f.Int(prefix+".chunk-size-mb", defaultChunkSizeMB, "S3 multipart download part size in MB")
	f.Int(prefix+".concurrency", defaultConcurrency, "S3 multipart download concurrency")
	f.Int(prefix+".max-retries", defaultMaxRetries, "maximum retries for S3 part body download")
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
	return nil
}


const (
	defaultChunkSizeMB = 32
	defaultMaxRetries  = 5
	defaultConcurrency = 10
)
