// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package config

import (
	"fmt"

	"github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/daprovider/daclient"
)

// DAConfig contains configuration for all DA providers
// TODO move "DAS" configuration here and rename to Anytrust
type DAConfig struct {
	ExternalProvider  daclient.ClientConfig               `koanf:"external-provider" reload:"hot"`
	ExternalProviders daclient.ExternalProviderConfigList `koanf:"external-providers" reload:"hot"`
}

func DAConfigAddOptions(prefix string, f *pflag.FlagSet) {
	daclient.ClientConfigAddOptions(prefix+".external-provider", f)
	daclient.ExternalProviderConfigAddPluralOptions(prefix+".external-provider", f)
}

func (c *DAConfig) Validate() error {
	if len(c.ExternalProviders) > 0 && c.ExternalProvider.RPC.URL != "" {
		return fmt.Errorf("cannot specify both external-provider and external-providers; use only external-providers for multiple providers")
	}
	if len(c.ExternalProviders) == 0 {
		c.ExternalProviders = daclient.ExternalProviderConfigList{c.ExternalProvider}
	}

	for i := range c.ExternalProviders {
		if err := c.ExternalProviders[i].RPC.Validate(); err != nil {
			return fmt.Errorf("failed to validate external-providers[%d].rpc: %w", i, err)
		}
	}
	return nil
}
