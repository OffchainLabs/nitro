// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/solgen/go/node_interfacegen"
)

func getL1BlockNum(t *testing.T, ctx context.Context, client *ethclient.Client, l2BlockNum uint64) uint64 {
	header, err := client.HeaderByNumber(ctx, big.NewInt(int64(l2BlockNum)))
	Require(t, err)
	l1BlockNum := types.DeserializeHeaderExtraInformation(header).L1BlockNumber
	return l1BlockNum
}

func TestGetL2BlockRangeForL1(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l2info, node, l2client, l1info, _, _, l1stack := createTestNodeOnL1(t, ctx, true)
	defer requireClose(t, l1stack)
	defer node.StopAndWait()
	user := l1info.GetDefaultTransactOpts("User", ctx)

	numTransactions := 30
	for i := 0; i < numTransactions; i++ {
		TransferBalanceTo(t, "Owner", util.RemapL1Address(user.From), big.NewInt(1e18), l2info, l2client, ctx)
	}

	nodeInterface, err := node_interfacegen.NewNodeInterface(types.NodeInterfaceAddress, l2client)
	Require(t, err)

	l1BlockNums := map[uint64][]uint64{}
	latestL2, err := l2client.BlockNumber(ctx)
	Require(t, err)
	for l2BlockNum := uint64(0); l2BlockNum <= latestL2; l2BlockNum++ {
		l1BlockNum := getL1BlockNum(t, ctx, l2client, l2BlockNum)
		l1BlockNums[l1BlockNum] = append(l1BlockNums[l1BlockNum], l2BlockNum)
	}

	// Test success
	for l1BlockNum := range l1BlockNums {
		rng, err := nodeInterface.GetL2BlockRangeForL1(&bind.CallOpts{}, l1BlockNum)
		Require(t, err)
		n := len(l1BlockNums[l1BlockNum])
		expected := []uint64{l1BlockNums[l1BlockNum][0], l1BlockNums[l1BlockNum][n-1]}
		if expected[0] != rng[0] || expected[1] != rng[1] {
			unexpectedL1BlockNum := getL1BlockNum(t, ctx, l2client, rng[1])
			// handle the edge case when new l2 blocks are produced between latestL2 was last calculated and now
			if unexpectedL1BlockNum != l1BlockNum {
				t.Fatalf("GetL2BlockRangeForL1 failed to get a valid range for L1 block number: %v. Given range: %v. Expected range: %v", l1BlockNum, rng, expected)
			}
		}
	}
	// Test invalid case
	finalValidL1BlockNumber := getL1BlockNum(t, ctx, l2client, latestL2)
	_, err = nodeInterface.GetL2BlockRangeForL1(&bind.CallOpts{}, finalValidL1BlockNumber+1)
	if err == nil {
		t.Fatalf("GetL2BlockRangeForL1 didn't fail for an invalid input")
	}

}
