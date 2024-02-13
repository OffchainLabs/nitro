// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package precompiles

import (
	"math/big"
)

// ArbStatistics provides statistics about the rollup right before the Nitro upgrade.
// In Classic, this was how a user would get info such as the total number of accounts,
// but there's now better ways to do that with geth.
type ArbStatistics struct {
	Address addr // 0x6e
}

// GetStats returns the current block number and some statistics about the rollup's pre-Nitro state
func (con ArbStatistics) GetStats(c ctx, evm mech) (huge, huge, huge, huge, huge, huge, error) {
	blockNum := evm.Context.BlockNumber
	classicNumAccounts := big.NewInt(0)  // TODO: hardcode the final value from Arbitrum Classic
	classicStorageSum := big.NewInt(0)   // TODO: hardcode the final value from Arbitrum Classic
	classicGasSum := big.NewInt(0)       // TODO: hardcode the final value from Arbitrum Classic
	classicNumTxes := big.NewInt(0)      // TODO: hardcode the final value from Arbitrum Classic
	classicNumContracts := big.NewInt(0) // TODO: hardcode the final value from Arbitrum Classic
	return blockNum, classicNumAccounts, classicStorageSum, classicGasSum, classicNumTxes, classicNumContracts, nil
}
