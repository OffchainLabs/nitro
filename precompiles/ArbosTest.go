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

func (con ArbosTest) BurnArbGas(c ctx, evm mech, gasAmount huge) error {
	if !gasAmount.IsUint64() {
		return errors.New("Not a uint64")
	}
	//nolint:errcheck
	c.Burn(gasAmount.Uint64()) // burn the amount, even if it's more than the user has
	return nil
}
