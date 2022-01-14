//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

type ArbBLS struct {
	Address addr
}

func (con ArbBLS) GetPublicKey(c ctx, evm mech, address addr) (huge, huge, huge, huge, error) {
	return c.state.BLSTable().GetPublicKey(address)
}

func (con ArbBLS) Register(c ctx, evm mech, x0, x1, y0, y1 huge) error {
	return c.state.BLSTable().Register(c.caller, x0, x1, y0, y1)
}
