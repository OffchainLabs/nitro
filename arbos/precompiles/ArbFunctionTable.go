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

type ArbFunctionTable struct{}

func (con ArbFunctionTable) Get(
	caller common.Address,
	st *state.StateDB,
	addr common.Address,
	index *big.Int,
) (*big.Int, bool, *big.Int, error) {
	return nil, false, nil, errors.New("unimplemented")
}

func (con ArbFunctionTable) GetGasCost(addr common.Address, index *big.Int) *big.Int {
	return nil
}

func (con ArbFunctionTable) Size(caller common.Address, st *state.StateDB, addr common.Address) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbFunctionTable) SizeGasCost(addr common.Address) *big.Int {
	return nil
}

func (con ArbFunctionTable) Upload(caller common.Address, st *state.StateDB, buf []byte) error {
	return errors.New("unimplemented")
}

func (con ArbFunctionTable) UploadGasCost(buf []byte) *big.Int {
	return nil
}
