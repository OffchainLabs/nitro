// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package pruner

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/util/rpcclient"
)

type Config struct {
	Enable                   bool                   `koanf:"enable"`
	StartDelayedMessageIndex uint64                 `koanf:"start-delayed-message-index"`
	PollInterval             time.Duration          `koanf:"poll-interval"`
	ParentChainScanRange     uint64                 `koanf:"parent-chain-scan-range"`
	BridgeAddress            string                 `koanf:"bridge-address"`
	ParentChain              rpcclient.ClientConfig `koanf:"parent-chain"`
}

var defaultParentChainConfig = func() rpcclient.ClientConfig {
	c := rpcclient.DefaultClientConfig
	c.URL = ""
	return c
}()

var DefaultConfig = Config{
	Enable:                   false,
	StartDelayedMessageIndex: 0,
	PollInterval:             10 * time.Second,
	ParentChainScanRange:     1000,
	BridgeAddress:            "",
	ParentChain:              defaultParentChainConfig,
}

func ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultConfig.Enable, "enable pruning of filtered transactions by scanning delayed messages on the parent chain")
	f.Uint64(prefix+".start-delayed-message-index", DefaultConfig.StartDelayedMessageIndex, "delayed message index from which the pruner will start processing")
	f.Duration(prefix+".poll-interval", DefaultConfig.PollInterval, "how long the pruner waits between iterations when it has no work left")
	f.Uint64(prefix+".parent-chain-scan-range", DefaultConfig.ParentChainScanRange, "maximum number of parent chain blocks scanned per pruner iteration")
	f.String(prefix+".bridge-address", DefaultConfig.BridgeAddress, "parent chain bridge contract address used to look up delayed messages")
	rpcclient.RPCClientAddOptions(prefix+".parent-chain", f, &DefaultConfig.ParentChain)
}

func (c *Config) Validate() error {
	if !c.Enable {
		return nil
	}
	if c.ParentChain.URL == "" {
		return errors.New("pruning.parent-chain.url must be set when pruning is enabled")
	}
	if c.BridgeAddress == "" {
		return errors.New("pruning.bridge-address must be set when pruning is enabled")
	}
	if !common.IsHexAddress(c.BridgeAddress) {
		return fmt.Errorf("pruning.bridge-address %q is not a valid hex address", c.BridgeAddress)
	}
	if c.PollInterval <= 0 {
		return errors.New("pruning.poll-interval must be positive")
	}
	if c.ParentChainScanRange == 0 {
		return errors.New("pruning.parent-chain-scan-range must be positive")
	}
	return c.ParentChain.Validate()
}
