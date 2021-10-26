//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
)

type ArbBLS struct {
	Address addr
}

func (con ArbBLS) GetPublicKey(b burn, caller addr, evm mech, addr addr) (huge, huge, huge, huge, error) {
	return nil, nil, nil, nil, errors.New("unimplemented")
}

func (con ArbBLS) Register(b burn, caller addr, evm mech, x0, x1, y0, y1 huge) error {
	return errors.New("unimplemented")
}
