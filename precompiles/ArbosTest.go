//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
)

type ArbosTest struct {
	Address addr
}

func (con ArbosTest) BurnArbGas(caller addr, evm mech, gasAmount huge) error {
	return nil
}

func (con ArbosTest) BurnArbGasGasCost(gasAmount huge) uint64 {
	if !gasAmount.IsUint64() {
		return ^uint64(0)
	}
	return gasAmount.Uint64()
}

func (con ArbosTest) GetAccountInfo(caller addr, evm mech, addr addr) error {
	return errors.New("unimplemented")
}

func (con ArbosTest) GetAccountInfoGasCost(addr addr) uint64 {
	return 0
}

func (con ArbosTest) GetMarshalledStorage(caller addr, evm mech, addr addr) error {
	return errors.New("unimplemented")
}

func (con ArbosTest) GetMarshalledStorageGasCost(addr addr) uint64 {
	return 0
}

func (con ArbosTest) InstallAccount(
	caller addr,
	evm mech,
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

func (con ArbosTest) SetNonce(caller addr, evm mech, addr addr, nonce huge) error {
	return errors.New("unimplemented")
}

func (con ArbosTest) SetNonceGasCost(addr addr, nonce huge) uint64 {
	return 0
}
