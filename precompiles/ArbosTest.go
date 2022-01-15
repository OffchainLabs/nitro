//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
)

// Provides a method of burning arbitrary amounts of gas, which exists for historical reasons.
type ArbosTest struct {
	Address addr
}

// Unproductively burns the amount of L2 ArbGas
func (con ArbosTest) BurnArbGas(c ctx, gasAmount huge) error {
	if !gasAmount.IsUint64() {
		return errors.New("Not a uint64")
	}
	//nolint:errcheck
	c.Burn(gasAmount.Uint64()) // burn the amount, even if it's more than the user has
	return nil
}
