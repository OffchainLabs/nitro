//go:build debugblock

package arbos

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/offchainlabs/nitro/arbos/arbosState"
)

func debugBlockStateUpdate(statedb *state.StateDB, expectedBalanceDelta *big.Int, chainConfig *params.ChainConfig) {
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
