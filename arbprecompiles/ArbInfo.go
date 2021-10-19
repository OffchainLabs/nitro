//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbprecompiles

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
)

type ArbInfo struct{}

func (con ArbInfo) GetBalance(caller common.Address, st *state.StateDB, account common.Address) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbInfo) GetBalanceGasCost(account common.Address) uint64 {
	return 0
}

func (con ArbInfo) GetCode(caller common.Address, st *state.StateDB, account common.Address) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbInfo) GetCodeGasCost(account common.Address) uint64 {
	return 0
}
