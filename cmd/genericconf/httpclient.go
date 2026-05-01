// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package genericconf

import (
	"errors"
	"fmt"
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

// Validate checks that c is well-formed when the feature it backs is enabled.
// Callers gate on URL == "" to mean "feature disabled" and should skip
// Validate in that case.
func (c *HTTPClientConfig) Validate() error {
	if c.URL == "" {
		return errors.New("url is required")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive, got %s", c.Timeout)
	}
	return nil
}

func HTTPClientConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".url", HTTPClientConfigDefault.URL, "HTTP endpoint URL")
	f.Duration(prefix+".timeout", HTTPClientConfigDefault.Timeout, "HTTP client timeout")
}
