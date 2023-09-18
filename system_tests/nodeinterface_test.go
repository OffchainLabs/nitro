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

	getBlockL1Num := func(l2BlockNum uint64) uint64 {
		header, err := l2client.HeaderByNumber(ctx, big.NewInt(int64(l2BlockNum)))
		Require(t, err)
		l1BlockNum := types.DeserializeHeaderExtraInformation(header).L1BlockNumber
		return l1BlockNum
	}

	l1BlockNums := map[uint64][]uint64{}
	latestL2, err := l2client.BlockNumber(ctx)
	Require(t, err)
	for l2BlockNum := uint64(0); l2BlockNum <= latestL2; l2BlockNum++ {
		l1BlockNum := getBlockL1Num(l2BlockNum)
		if len(l1BlockNums[l1BlockNum]) <= 1 {
			l1BlockNums[l1BlockNum] = append(l1BlockNums[l1BlockNum], l2BlockNum)
		} else {
			l1BlockNums[l1BlockNum][1] = l2BlockNum
		}
	}

	// Test success
	for l1BlockNum := range l1BlockNums {
		rng, err := nodeInterface.L2BlockRangeForL1(&bind.CallOpts{}, l1BlockNum)
		Require(t, err)
		n := len(l1BlockNums[l1BlockNum])
		expected := []uint64{l1BlockNums[l1BlockNum][0], l1BlockNums[l1BlockNum][n-1]}
		if expected[0] != rng.FirstBlock || expected[1] != rng.LastBlock {
			unexpectedL1BlockNum := getBlockL1Num(rng.LastBlock)
			// Handle the edge case when new l2 blocks are produced between latestL2 was last calculated and now.
			if unexpectedL1BlockNum != l1BlockNum || rng.LastBlock < expected[1] {
				t.Errorf("L2BlockRangeForL1(%d) = (%d %d) want (%d %d)", l1BlockNum, rng.FirstBlock, rng.LastBlock, expected[0], expected[1])
			}
		}
	}
	// Test invalid case
	finalValidL1BlockNumber := getBlockL1Num(latestL2)
	_, err = nodeInterface.L2BlockRangeForL1(&bind.CallOpts{}, finalValidL1BlockNumber+1)
	if err == nil {
		t.Fatalf("GetL2BlockRangeForL1 didn't fail for an invalid input")
	}

}
