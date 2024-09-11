// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package chaininfo

import (
	"reflect"
	"testing"
)

func TestDefaultChainConfigsCopyCorrectly(t *testing.T) {
	if !reflect.DeepEqual(DefaultChainConfigs["arb1"], ArbitrumOneChainConfig()) {
		t.Fatal("copy of arb1 default chain config mismatch")
	}
	if !reflect.DeepEqual(DefaultChainConfigs["nova"], ArbitrumNovaChainConfig()) {
		t.Fatal("copy of nova default chain config mismatch")
	}
	if !reflect.DeepEqual(DefaultChainConfigs["goerli-rollup"], ArbitrumRollupGoerliTestnetChainConfig()) {
		t.Fatal("copy of goerli-rollup default chain config mismatch")
	}
	if !reflect.DeepEqual(DefaultChainConfigs["arb-dev-test"], ArbitrumDevTestChainConfig()) {
		t.Fatal("copy of arb-dev-test default chain config mismatch")
	}
	if !reflect.DeepEqual(DefaultChainConfigs["anytrust-dev-test"], ArbitrumDevTestDASChainConfig()) {
		t.Fatal("copy of anytrust-dev-test default chain config mismatch")
	}
}
