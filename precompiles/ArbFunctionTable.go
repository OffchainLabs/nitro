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

func (con ArbFunctionTable) Get(b burn, caller addr, evm mech, addr addr, index huge) (huge, bool, huge, error) {
	return nil, false, nil, errors.New("unimplemented")
}

func (con ArbFunctionTable) Size(b burn, caller addr, evm mech, addr addr) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbFunctionTable) Upload(b burn, caller addr, evm mech, buf []byte) error {
	return errors.New("unimplemented")
}
