//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
)

type ArbosTest struct{}

func (con ArbosTest) BurnArbGas(caller addr, st *stateDB, gasAmount huge) error {
	return nil
}

func (con ArbosTest) BurnArbGasGasCost(gasAmount huge) uint64 {
	if !gasAmount.IsUint64() {
		return ^uint64(0)
	}
	return gasAmount.Uint64()
}

func (con ArbosTest) GetAccountInfo(caller addr, st *stateDB, addr addr) error {
	return errors.New("unimplemented")
}

func (con ArbosTest) GetAccountInfoGasCost(addr addr) uint64 {
	return 0
}

func (con ArbosTest) GetMarshalledStorage(caller addr, st *stateDB, addr addr) error {
	return errors.New("unimplemented")
}

func (con ArbosTest) GetMarshalledStorageGasCost(addr addr) uint64 {
	return 0
}

func (con ArbosTest) InstallAccount(
	caller addr,
	st *stateDB,
	addr addr,
	isEOA bool,
	balance huge,
	nonce huge,
	code []byte,
	initStorage []byte,
) error {
	return errors.New("unimplemented")
}

func (con ArbosTest) InstallAccountGasCost(
	addr addr,
	isEOA bool,
	balance huge,
	nonce huge,
	code []byte,
	initStorage []byte,
) uint64 {
	return 0
}

func (con ArbosTest) SetNonce(caller addr, st *stateDB, addr addr, nonce huge) error {
	return errors.New("unimplemented")
}

func (con ArbosTest) SetNonceGasCost(addr addr, nonce huge) uint64 {
	return 0
}
