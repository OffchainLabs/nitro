//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
)

type ArbInfo struct {
	Address addr
}

func (con ArbInfo) GetBalance(caller addr, evm mech, account addr) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbInfo) GetBalanceGasCost(account addr) uint64 {
	return 0
}

func (con ArbInfo) GetCode(caller addr, evm mech, account addr) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbInfo) GetCodeGasCost(account addr) uint64 {
	return 0
}
