// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

// buildEstimateGasFilterNode creates a single sequencer node with RPC filtering
// enabled. The txFilterer is wired into the backend at construction time.
func buildEstimateGasFilterNode(t *testing.T, ctx context.Context, enableETHCallFilter bool) (builder *NodeBuilder, cleanup func()) {
	t.Helper()
	builder = NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.isSequencer = true
	builder.execConfig.TransactionFiltering.EnableETHCallFilter = enableETHCallFilter
	cleanup = builder.Build(t)
	return builder, cleanup
}

// TestEstimateGasFilterDirectAddress verifies that eth_estimateGas rejects
// calls involving a filtered address when EnableETHCallFilter is true.
func TestEstimateGasFilterDirectAddress(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder, cleanup := buildEstimateGasFilterNode(t, ctx, true)
	defer cleanup()

	builder.L2Info.GenerateAccount("FilteredUser")
	builder.L2Info.GenerateAccount("NormalUser")
	builder.L2.TransferBalance(t, "Owner", "NormalUser", big.NewInt(1e18), builder.L2Info)
	builder.L2.TransferBalance(t, "Owner", "FilteredUser", big.NewInt(1e18), builder.L2Info)

	filteredAddr := builder.L2Info.GetAddress("FilteredUser")
	normalAddr := builder.L2Info.GetAddress("NormalUser")
	filter := newHashedChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(t, filter)

	// EstimateGas TO filtered address should fail
	_, err := builder.L2.Client.EstimateGas(ctx, ethereum.CallMsg{
		From:  normalAddr,
		To:    &filteredAddr,
		Value: big.NewInt(1e12),
	})
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error for EstimateGas TO filtered address, got: %v", err)
	}

	// EstimateGas FROM filtered address should fail
	_, err = builder.L2.Client.EstimateGas(ctx, ethereum.CallMsg{
		From:  filteredAddr,
		To:    &normalAddr,
		Value: big.NewInt(1e12),
	})
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error for EstimateGas FROM filtered address, got: %v", err)
	}

	// EstimateGas between clean addresses should succeed
	_, err = builder.L2.Client.EstimateGas(ctx, ethereum.CallMsg{
		From:  normalAddr,
		To:    &normalAddr,
		Value: big.NewInt(1e12),
	})
	Require(t, err)
}

// TestEstimateGasFilterDisabled verifies that eth_estimateGas does not reject
// calls to filtered addresses when EnableETHCallFilter is false.
func TestEstimateGasFilterDisabled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder, cleanup := buildEstimateGasFilterNode(t, ctx, false)
	defer cleanup()

	builder.L2Info.GenerateAccount("FilteredUser")
	builder.L2Info.GenerateAccount("NormalUser")
	builder.L2.TransferBalance(t, "Owner", "NormalUser", big.NewInt(1e18), builder.L2Info)
	builder.L2.TransferBalance(t, "Owner", "FilteredUser", big.NewInt(1e18), builder.L2Info)

	filteredAddr := builder.L2Info.GetAddress("FilteredUser")
	normalAddr := builder.L2Info.GetAddress("NormalUser")
	filter := newHashedChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(t, filter)

	// EstimateGas TO filtered address should succeed when RPC filter is disabled
	_, err := builder.L2.Client.EstimateGas(ctx, ethereum.CallMsg{
		From:  normalAddr,
		To:    &filteredAddr,
		Value: big.NewInt(1e12),
	})
	Require(t, err)
}
