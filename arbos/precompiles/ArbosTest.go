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

type ArbosTest struct{}

func (con ArbosTest) BurnArbGas(st *state.StateDB, gasAmount *big.Int) error {
	return errors.New("unimplemented")
}

func (con ArbosTest) GetAccountInfo(st *state.StateDB, addr common.Address) error {
	return errors.New("unimplemented")
}

func (con ArbosTest) GetMarshalledStorage(st *state.StateDB, addr common.Address) error {
	return errors.New("unimplemented")
}

func (con ArbosTest) InstallAccount(
	st *state.StateDB,
	addr common.Address,
	isEOA bool,
	balance *big.Int,
	nonce *big.Int,
	code []byte,
	initStorage []byte,
) error {
	return errors.New("unimplemented")
}

func (con ArbosTest) SetNonce(st *state.StateDB, addr common.Address, nonce *big.Int) error {
	return errors.New("unimplemented")
}
