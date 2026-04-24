// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package genericconf

import (
	"errors"
	"time"

	"github.com/spf13/pflag"
)

type HTTPClientConfig struct {
	URL     string        `koanf:"url"`
	Timeout time.Duration `koanf:"timeout"`
}

var HTTPClientConfigDefault = HTTPClientConfig{
	Timeout: 5 * time.Second,
}

func (c *HTTPClientConfig) Validate() error {
	if c.URL == "" {
		return errors.New("url is required")
	}
	return nil
}

func HTTPClientConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".url", HTTPClientConfigDefault.URL, "HTTP endpoint URL")
	f.Duration(prefix+".timeout", HTTPClientConfigDefault.Timeout, "HTTP client timeout")
}
