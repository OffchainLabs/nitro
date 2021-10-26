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

func (con ArbInfo) GetBalance(b burn, caller addr, evm mech, account addr) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbInfo) GetCode(b burn, caller addr, evm mech, account addr) ([]byte, error) {
	return nil, errors.New("unimplemented")
}
