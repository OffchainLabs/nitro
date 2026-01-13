// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package restrictedaddr

import (
	"errors"
	"time"

	"github.com/spf13/pflag"
)

type Config struct {
	Enable       bool          `koanf:"enable"`
	S3Bucket     string        `koanf:"s3-bucket"`
	S3Region     string        `koanf:"s3-region"`
	S3AccessKey  string        `koanf:"s3-access-key"`
	S3SecretKey  string        `koanf:"s3-secret-key"`
	S3ObjectKey  string        `koanf:"s3-object-key"`
	PollInterval time.Duration `koanf:"poll-interval"`
}

var DefaultConfig = Config{
	Enable:       false,
	PollInterval: 5 * time.Minute,
}

func ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultConfig.Enable, "enable restricted address synchronization service")
	f.String(prefix+".s3-bucket", "", "S3 bucket containing the restricted address hash list")
	f.String(prefix+".s3-region", "", "AWS region of the S3 bucket")
	f.String(prefix+".s3-access-key", "", "AWS access key for S3 (optional, uses default credentials if "+
		"not provided which check for credentials in specific order like env variables, shared credentials, etc.)")
	f.String(prefix+".s3-secret-key", "", "AWS secret key for S3 (optional, uses default credentials if "+
		"not provided which check for credentials in specific order like env variables, shared credentials, etc.)")
	f.String(prefix+".s3-object-key", "", "S3 object key (path) to the hash list JSON file")
	f.Duration(prefix+".poll-interval", DefaultConfig.PollInterval, "interval between polling S3 for hash list updates")
}

func (c *Config) Validate() error {
	if !c.Enable {
		return nil
	}

	if c.S3Bucket == "" {
		return errors.New("restricted-addr.s3-bucket is required when enabled")
	}
	if c.S3Region == "" {
		return errors.New("restricted-addr.s3-region is required when enabled")
	}
	if c.S3ObjectKey == "" {
		return errors.New("restricted-addr.s3-object-key is required when enabled")
	}
	if c.PollInterval <= 0 {
		return errors.New("restricted-addr.poll-interval must be positive")
	}

	return nil
}
