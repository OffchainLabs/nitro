// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package chaininfo

import (
	"reflect"
	"testing"
)

func TestDefaultChainConfigsCopyCorrectly(t *testing.T) {
	for _, chainName := range []string{"arb1", "nova", "goerli-rollup", "arb-dev-test", "anytrust-dev-test"} {
		if !reflect.DeepEqual(DefaultChainConfigs[chainName], fetchChainConfig(chainName)) {
			t.Fatalf("copy of %s default chain config mismatch", chainName)
		}
	}
}
