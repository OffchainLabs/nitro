// Copyright 2026-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"errors"
	"time"

	"github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/util/s3syncer"
)

type Config struct {
	Enable       bool            `koanf:"enable"`
	S3           s3syncer.Config `koanf:"s3"`
	PollInterval time.Duration   `koanf:"poll-interval"`
	CacheSize    int             `koanf:"cache-size"`
}

var DefaultConfig = Config{
	Enable:       false,
	PollInterval: 5 * time.Minute,
	CacheSize:    10000,
	S3:           s3syncer.DefaultS3Config,
}

func ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultConfig.Enable, "enable restricted address synchronization service")
	s3syncer.ConfigAddOptions(prefix+".s3", f)
	f.Duration(prefix+".poll-interval", DefaultConfig.PollInterval, "interval between polling S3 for hash list updates")
	f.Int(prefix+".cache-size", DefaultConfig.CacheSize, "LRU cache size for address lookup results")
}

func (c *Config) Validate() error {
	if !c.Enable {
		return nil
	}

	if err := c.S3.Validate(); err != nil {
		return err
	}

	if c.PollInterval <= 0 {
		return errors.New("address-filter.poll-interval must be positive")
	}

	if c.CacheSize <= 0 {
		return errors.New("address-filter.cache-size must be positive")
	}

	return nil
}
