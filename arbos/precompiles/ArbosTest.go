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

func (con ArbosTest) BurnArbGas(caller common.Address, st *state.StateDB, gasAmount *big.Int) error {
	return nil
}

func (con ArbosTest) BurnArbGasGasCost(gasAmount *big.Int) *big.Int {
	return gasAmount
}

func (con ArbosTest) GetAccountInfo(caller common.Address, st *state.StateDB, addr common.Address) error {
	return errors.New("unimplemented")
}

func (con ArbosTest) GetAccountInfoGasCost(addr common.Address) *big.Int {
	return nil
}

func (con ArbosTest) GetMarshalledStorage(caller common.Address, st *state.StateDB, addr common.Address) error {
	return errors.New("unimplemented")
}

func (con ArbosTest) GetMarshalledStorageGasCost(addr common.Address) *big.Int {
	return nil
}

func (con ArbosTest) InstallAccount(
	caller common.Address,
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

func (con ArbosTest) InstallAccountGasCost(
	addr common.Address,
	isEOA bool,
	balance *big.Int,
	nonce *big.Int,
	code []byte,
	initStorage []byte,
) *big.Int {
	return nil
}

func (con ArbosTest) SetNonce(caller common.Address, st *state.StateDB, addr common.Address, nonce *big.Int) error {
	return errors.New("unimplemented")
}

func (con ArbosTest) SetNonceGasCost(addr common.Address, nonce *big.Int) *big.Int {
	return nil
}
