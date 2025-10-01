//go:build !debugblock

package debugblock

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/spf13/pflag"
)

func (c *Config) Validate() error {
	if c.OverwriteChainConfig || c.DebugAddress != "" || c.DebugBlockNum != 0 {
		errors.New("debug block injection is not supported in this build")
	}
	return nil
}

func (c *Config) Apply(_ *params.ChainConfig) {
	// do nothing
}

func ConfigAddOptions(_ string, _ *pflag.FlagSet) {
	// don't add any of debug block options
}

func DebugBlockStateUpdate(_ *state.StateDB, _ *big.Int, _ *params.ChainConfig) {
	log.Warn("debugBlockStateUpdate is not supported in this build")
}
