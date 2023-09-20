// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/solgen/go/node_interfacegen"
)

func TestL2BlockRangeForL1(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l2info, node, l2client, l1info, _, _, l1stack := createTestNodeOnL1(t, ctx, true)
	defer requireClose(t, l1stack)
	defer node.StopAndWait()
	user := l1info.GetDefaultTransactOpts("User", ctx)

	numTransactions := 200
	for i := 0; i < numTransactions; i++ {
		TransferBalanceTo(t, "Owner", util.RemapL1Address(user.From), big.NewInt(1e18), l2info, l2client, ctx)
	}

	nodeInterface, err := node_interfacegen.NewNodeInterface(types.NodeInterfaceAddress, l2client)
	Require(t, err)

	l1BlockNums := map[uint64][2]uint64{}
	latestL2, err := l2client.BlockNumber(ctx)
	Require(t, err)
	for l2BlockNum := uint64(0); l2BlockNum <= latestL2; l2BlockNum++ {
		l1BlockNum, err := nodeInterface.BlockL1Num(&bind.CallOpts{}, l2BlockNum)
		Require(t, err)
		if _, ok := l1BlockNums[l1BlockNum]; !ok {
			l1BlockNums[l1BlockNum] = [2]uint64{l2BlockNum, l2BlockNum}
		} else {
			l1BlockNums[l1BlockNum] = [2]uint64{l1BlockNums[l1BlockNum][0], l2BlockNum}
		}
	}

	// Test success
	for l1BlockNum := range l1BlockNums {
		rng, err := nodeInterface.L2BlockRangeForL1(&bind.CallOpts{}, l1BlockNum)
		Require(t, err)
		expected := l1BlockNums[l1BlockNum]
		if rng.FirstBlock != expected[0] || rng.LastBlock != expected[1] {
			unexpectedL1BlockNum, err := nodeInterface.BlockL1Num(&bind.CallOpts{}, rng.LastBlock)
			Require(t, err)
			// Handle the edge case when new l2 blocks are produced between latestL2 was last calculated and now.
			if unexpectedL1BlockNum != l1BlockNum || rng.LastBlock < expected[1] || rng.FirstBlock != expected[0] {
				t.Errorf("L2BlockRangeForL1(%d) = (%d %d) want (%d %d)", l1BlockNum, rng.FirstBlock, rng.LastBlock, expected[0], expected[1])
			}
		}
	}
	// Test invalid case
	finalValidL1BlockNumber, err := nodeInterface.BlockL1Num(&bind.CallOpts{}, latestL2)
	Require(t, err)
	if _, err := nodeInterface.L2BlockRangeForL1(&bind.CallOpts{}, finalValidL1BlockNumber+1); err == nil {
		t.Fatalf("GetL2BlockRangeForL1 didn't fail for an invalid input")
	}

}
