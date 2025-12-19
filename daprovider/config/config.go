// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package config

import (
	"github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/daprovider/anytrust"
	"github.com/offchainlabs/nitro/daprovider/daclient"
)

// DAConfig contains configuration for all DA providers
type DAConfig struct {
	ExternalProvider daclient.ClientConfig `koanf:"external-provider" reload:"hot"`
	AnyTrust         anytrust.Config       `koanf:"anytrust" reload:"hot"`
}

var DefaultDAConfig = DAConfig{
	ExternalProvider: daclient.DefaultClientConfig,
	AnyTrust:         anytrust.DefaultConfigForNode,
}

func DAConfigAddOptions(prefix string, f *pflag.FlagSet) {
	daclient.ClientConfigAddOptions(prefix+".external-provider", f)
	anytrust.ConfigAddNodeOptions(prefix+".anytrust", f)
}

func (c *DAConfig) Validate() error {
	return c.ExternalProvider.RPC.Validate()
}
