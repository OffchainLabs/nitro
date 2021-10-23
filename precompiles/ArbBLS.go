//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
)

type ArbBLS struct{}

func (con ArbBLS) GetPublicKey(caller addr, st *stateDB, addr addr) (huge, huge, huge, huge, error) {
	return nil, nil, nil, nil, errors.New("unimplemented")
}

func (con ArbBLS) GetPublicKeyGasCost(addr addr) uint64 {
	return 0
}

func (con ArbBLS) Register(caller addr, st *stateDB, x0, x1, y0, y1 huge) error {
	return errors.New("unimplemented")
}

func (con ArbBLS) RegisterGasCost(x0, x1, y0, y1 huge) uint64 {
	return 0
}
