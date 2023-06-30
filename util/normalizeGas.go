// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package util

import (
	"github.com/offchainlabs/nitro/arbos/l2pricing"
)

// NormalizeL2GasForL1GasInitial is for testing, adjusts an L2 gas amount that represents L1 gas spending, to compensate for
// the difference between the assumed L2 base fee and the actual initial L2 base fee.
func NormalizeL2GasForL1GasInitial(l2gas uint64, assumedL2Basefee uint64) uint64 {
	return l2gas * assumedL2Basefee / l2pricing.InitialBaseFeeWei
}
