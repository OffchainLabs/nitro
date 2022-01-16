//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"math/big"
)

type ArbStatistics struct {
	Address addr
}

func (con ArbStatistics) GetStats(c ctx, evm mech) (huge, huge, huge, huge, huge, huge, error) {
	blockNum := evm.Context.BlockNumber
	classicNumAccounts := big.NewInt(0)  // TODO: hardcode the final value from Arbitrum Classic
	classicStorageSum := big.NewInt(0)   // TODO: hardcode the final value from Arbitrum Classic
	classicGasSum := big.NewInt(0)       // TODO: hardcode the final value from Arbitrum Classic
	classicNumTxes := big.NewInt(0)      // TODO: hardcode the final value from Arbitrum Classic
	classicNumContracts := big.NewInt(0) // TODO: hardcode the final value from Arbitrum Classic
	return blockNum, classicNumAccounts, classicStorageSum, classicGasSum, classicNumTxes, classicNumContracts, nil
}
