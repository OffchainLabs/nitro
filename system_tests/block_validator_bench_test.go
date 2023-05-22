// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

// race detection makes things slow and miss timeouts
//go:build block_validator_bench
// +build block_validator_bench

package arbtest

import (
	"testing"
)

func TestBlockValidatorBenchmark(t *testing.T) {
	testBlockValidatorSimple(t, "onchain", 1, depleteGas, true)
}
