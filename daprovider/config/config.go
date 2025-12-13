// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package config

import (
	"github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/daprovider/daclient"
)

// DAConfig contains configuration for all DA providers
// TODO move "DAS" configuration here and rename to Anytrust
type DAConfig struct {
	ExternalProvider daclient.ClientConfig `koanf:"external-provider" reload:"hot"`
}

var DefaultDAConfig = DAConfig{
	ExternalProvider: daclient.DefaultClientConfig,
}

func DAConfigAddOptions(prefix string, f *pflag.FlagSet) {
	daclient.ClientConfigAddOptions(prefix+".external-provider", f)
}

func (c *DAConfig) Validate() error {
	return c.ExternalProvider.RPC.Validate()
}
