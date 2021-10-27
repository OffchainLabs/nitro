//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
)

type ArbFunctionTable struct {
	Address addr
}

func (con ArbFunctionTable) Get(c ctx, evm mech, addr addr, index huge) (huge, bool, huge, error) {
	return nil, false, nil, errors.New("unimplemented")
}

func (con ArbFunctionTable) Size(c ctx, evm mech, addr addr) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbFunctionTable) Upload(c ctx, evm mech, buf []byte) error {
	return errors.New("unimplemented")
}
