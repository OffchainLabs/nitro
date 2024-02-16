// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/solgen/go/node_interfacegen"
)

func TestL2BlockRangeForL1(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()
	user := builder.L1Info.GetDefaultTransactOpts("User", ctx)

	numTransactions := 200
	for i := 0; i < numTransactions; i++ {
		builder.L2.TransferBalanceTo(t, "Owner", util.RemapL1Address(user.From), big.NewInt(1e18), builder.L2Info)
	}

	nodeInterface, err := node_interfacegen.NewNodeInterface(types.NodeInterfaceAddress, builder.L2.Client)
	if err != nil {
		t.Fatalf("Error creating node interface: %v", err)
	}

	l1BlockNums := map[uint64]*[2]uint64{}
	latestL2, err := builder.L2.Client.BlockNumber(ctx)
	if err != nil {
		t.Fatalf("Error querying most recent l2 block: %v", err)
	}
	for l2BlockNum := uint64(0); l2BlockNum <= latestL2; l2BlockNum++ {
		l1BlockNum, err := nodeInterface.BlockL1Num(&bind.CallOpts{}, l2BlockNum)
		if err != nil {
			t.Fatalf("Error quering l1 block number for l2 block: %d, error: %v", l2BlockNum, err)
		}
		if _, ok := l1BlockNums[l1BlockNum]; !ok {
			l1BlockNums[l1BlockNum] = &[2]uint64{l2BlockNum, l2BlockNum}
		}
		l1BlockNums[l1BlockNum][1] = l2BlockNum
	}

	// Test success.
	for l1BlockNum := range l1BlockNums {
		rng, err := nodeInterface.L2BlockRangeForL1(&bind.CallOpts{}, l1BlockNum)
		if err != nil {
			t.Fatalf("Error getting l2 block range for l1 block: %d, error: %v", l1BlockNum, err)
		}
		expected := l1BlockNums[l1BlockNum]
		if rng.FirstBlock != expected[0] || rng.LastBlock != expected[1] {
			unexpectedL1BlockNum, err := nodeInterface.BlockL1Num(&bind.CallOpts{}, rng.LastBlock)
			if err != nil {
				t.Fatalf("Error quering l1 block number for l2 block: %d, error: %v", rng.LastBlock, err)
			}
			// Handle the edge case when new l2 blocks are produced between latestL2 was last calculated and now.
			if unexpectedL1BlockNum != l1BlockNum || rng.LastBlock < expected[1] || rng.FirstBlock != expected[0] {
				t.Errorf("L2BlockRangeForL1(%d) = (%d %d) want (%d %d)", l1BlockNum, rng.FirstBlock, rng.LastBlock, expected[0], expected[1])
			}
		}
	}
	// Test invalid case.
	if _, err := nodeInterface.L2BlockRangeForL1(&bind.CallOpts{}, 1e5); err == nil {
		t.Fatalf("GetL2BlockRangeForL1 didn't fail for an invalid input")
	}
}

func TestGetL1Confirmations(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	nodeInterface, err := node_interfacegen.NewNodeInterface(types.NodeInterfaceAddress, builder.L2.Client)
	Require(t, err)

	genesisBlock, err := builder.L2.Client.BlockByNumber(ctx, big.NewInt(0))
	Require(t, err)
	l1Confs, err := nodeInterface.GetL1Confirmations(&bind.CallOpts{}, genesisBlock.Hash())
	Require(t, err)

	numTransactions := 200

	if l1Confs >= uint64(numTransactions) {
		t.Fatalf("L1Confirmations for latest block %v is already %v (over %v)", genesisBlock.Number(), l1Confs, numTransactions)
	}

	for i := 0; i < numTransactions; i++ {
		builder.L1.TransferBalance(t, "User", "User", common.Big0, builder.L1Info)
	}

	l1Confs, err = nodeInterface.GetL1Confirmations(&bind.CallOpts{}, genesisBlock.Hash())
	Require(t, err)

	// Allow a gap of 10 for asynchronicity, just in case
	if l1Confs+10 < uint64(numTransactions) {
		t.Fatalf("L1Confirmations for latest block %v is only %v (did not hit expected %v)", genesisBlock.Number(), l1Confs, numTransactions)
	}
}
