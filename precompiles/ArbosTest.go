// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package precompiles

import (
	"errors"
)

// ArbosTest provides a method of burning arbitrary amounts of gas, which exists for historical reasons.
type ArbosTest struct {
	Address addr // 0x69
}

// BurnArbGas unproductively burns the amount of L2 ArbGas
func (con ArbosTest) BurnArbGas(c ctx, gasAmount huge) error {
	if !gasAmount.IsUint64() {
		return errors.New("not a uint64")
	}
	//nolint:errcheck
	c.Burn(gasAmount.Uint64()) // burn the amount, even if it's more than the user has
	return nil
}
