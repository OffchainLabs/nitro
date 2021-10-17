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

type ArbBLS struct{}

func (con ArbBLS) GetPublicKey(
	st *state.StateDB,
	addr common.Address,
) (*big.Int, *big.Int, *big.Int, *big.Int, error) {
	return nil, nil, nil, nil, errors.New("unimplemented")
}

func (con ArbBLS) Register(st *state.StateDB, x0, x1, y0, y1 *big.Int) error {
	return errors.New("unimplemented")
}
