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
	Enable       bool            `koanf:"enable"`
	S3           s3syncer.Config `koanf:"s3"`
	PollInterval time.Duration   `koanf:"poll-interval"`
}

var DefaultConfig = Config{
	Enable:       false,
	PollInterval: 5 * time.Minute,
}

func ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultConfig.Enable, "enable restricted address synchronization service")
	s3syncer.ConfigAddOptions(prefix+".s3", f)
	f.Duration(prefix+".poll-interval", DefaultConfig.PollInterval, "interval between polling S3 for hash list updates")
}

func (c *Config) Validate() error {
	if !c.Enable {
		return nil
	}

	if err := c.S3.Validate(); err != nil {
		return err
	}

	if c.PollInterval <= 0 {
		return errors.New("restricted-addr.poll-interval must be positive")
	}

	return nil
}
