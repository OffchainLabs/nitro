// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package util

import "errors"

var ErrZeroGasEstimate = errors.New("gas estimate is zero")

// CheckedGasEstimate rejects a gas estimate of zero, which no transaction we send
// can run on, so sending one is guaranteed to fail. It is meant to wrap an
// estimation call directly:
//
//	gas, err := util.CheckedGasEstimate(client.EstimateGas(ctx, msg))
func CheckedGasEstimate(gas uint64, err error) (uint64, error) {
	if err != nil {
		return 0, err
	}
	if gas == 0 {
		return 0, ErrZeroGasEstimate
	}
	return gas, nil
}
