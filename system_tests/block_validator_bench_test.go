// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// race detection makes things slow and miss timeouts
//go:build block_validator_bench

package arbtest

import (
	"testing"
)

func TestBlockValidatorBenchmark(t *testing.T) {
	opts := Options{
		dasModeString: "onchain",
		workloadLoops: 1,
		workload:      depleteGas,
		arbitrator:    true,
	}
	testBlockValidatorSimple(t, opts)
}
