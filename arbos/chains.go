//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/params"
)

func GetChainConfig(chainId *big.Int) (*params.ChainConfig, error) {
	for _, potentialChainConfig := range params.ArbitrumSupportedChainConfigs {
		if potentialChainConfig.ChainID.Cmp(chainId) == 0 {
			return potentialChainConfig, nil
		}
	}
	return nil, fmt.Errorf("unsupported L2 chain ID %v", chainId)
}
