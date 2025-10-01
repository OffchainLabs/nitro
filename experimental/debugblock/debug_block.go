//go:build debugblock

package debugblock

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/spf13/pflag"
)

func ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".overwrite-chain-config", ConfigDefault.OverwriteChainConfig, "DANGEROUS! overwrites chain when opening existing database; chain debug mode will be enabled")
	f.String(prefix+".debug-address", ConfigDefault.DebugAddress, "DANGEROUS! address of debug account to be pre-funded")
	f.Uint64(prefix+".debug-blocknum", ConfigDefault.DebugBlockNum, "DANGEROUS! block number of injected debug block")
}

func (c *Config) Validate() error {
	if c.OverwriteChainConfig {
		log.Warn("DANGER! overwrite-chain-config set, chain config will be over-written")
	}
	if c.DebugAddress != "" && !common.IsHexAddress(c.DebugAddress) {
		return errors.New("invalid debug-address, hex address expected")
	}
	if c.DebugBlockNum != 0 {
		log.Warn("DANGER! debug-blocknum set", "blocknum", c.DebugBlockNum)
	}
	return nil
}

func (c *Config) Apply(chainConfig *params.ChainConfig) {
	if c.OverwriteChainConfig {
		chainConfig.ArbitrumChainParams.AllowDebugPrecompiles = true
		chainConfig.ArbitrumChainParams.DebugAddress = common.HexToAddress(c.DebugAddress)
		chainConfig.ArbitrumChainParams.DebugBlock = c.DebugBlockNum
	}
}

func DebugBlockStateUpdate(statedb *state.StateDB, expectedBalanceDelta *big.Int, chainConfig *params.ChainConfig) {
	// fund debug account
	balance := uint256.MustFromBig(new(big.Int).Lsh(big.NewInt(1), 254))
	statedb.SetBalance(chainConfig.ArbitrumChainParams.DebugAddress, balance, tracing.BalanceChangeUnspecified)
	expectedBalanceDelta.Add(expectedBalanceDelta, balance.ToBig())

	// save current chain config to arbos state in case it was changed to enable debug mode and debug block
	// replay binary reads chain config from arbos state, that will enable successful validation of future blocks
	// (debug block will still fail validation if chain config was changed off-chain)
	if serializedChainConfig, err := json.Marshal(chainConfig); err != nil {
		log.Error("debug block: failed to marshal chain config", "err", err)
	} else if arbStateWrite, err := arbosState.OpenSystemArbosState(statedb, nil, false); err != nil {
		log.Error("debug block: failed to open arbos state for writing", "err", err)
	} else if err = arbStateWrite.SetChainConfig(serializedChainConfig); err != nil {
		log.Error("debug block: failed to set chain config in arbos state", "err", err)
	}
	log.Warn("DANGER! Producing debug block and funding debug account", "debugAddress", chainConfig.ArbitrumChainParams.DebugAddress)
}
