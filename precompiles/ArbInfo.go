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

func (con ArbInfo) GetBalance(c ctx, evm mech, account addr) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbInfo) GetCode(c ctx, evm mech, account addr) ([]byte, error) {
	return nil, errors.New("unimplemented")
}
