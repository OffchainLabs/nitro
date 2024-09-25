// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package precompiles

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestArbStatistics(t *testing.T) {
	evm := newMockEVMForTesting()
	stats := ArbStatistics{}
	context := testContext(common.Address{}, evm)

	blockNum, _, _, _, _, _, err := stats.GetStats(context, evm)
	Require(t, err)
	if blockNum.Cmp(evm.Context.BlockNumber) != 0 {
		t.Error("Unexpected block number")
	}
}
