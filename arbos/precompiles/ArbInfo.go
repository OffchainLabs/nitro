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

type ArbInfo struct{}

func (con ArbInfo) GetBalance(caller common.Address, st *state.StateDB, account common.Address) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbInfo) GetBalanceGasCost(account common.Address) *big.Int {
	return nil
}

func (con ArbInfo) GetCode(caller common.Address, st *state.StateDB, account common.Address) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbInfo) GetCodeGasCost(account common.Address) *big.Int {
	return nil
}
