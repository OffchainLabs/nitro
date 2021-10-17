//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"math/big"
)

type ArbStatistics struct{}

func (con ArbStatistics) GetStats(
	caller common.Address,
	st *state.StateDB,
) (*big.Int, *big.Int, *big.Int, *big.Int, *big.Int, *big.Int, error) {
	return nil, nil, nil, nil, nil, nil, errors.New("unimplemented")
}
