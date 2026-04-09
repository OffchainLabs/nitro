// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"bytes"
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
)

// buildRPCFilterNode creates a single sequencer node with RPC filtering
// enabled or disabled. The txFilterer is wired into the backend at construction time.
func buildRPCFilterNode(t *testing.T, ctx context.Context, enableETHCallFilter bool) (builder *NodeBuilder, cleanup func()) {
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

	builder, cleanup := buildRPCFilterNode(t, ctx, true)
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

	builder, cleanup := buildRPCFilterNode(t, ctx, false)
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

// TestEthCallFilterDirectAddress verifies that eth_call rejects
// calls involving a filtered address when EnableETHCallFilter is true.
func TestEthCallFilterDirectAddress(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder, cleanup := buildRPCFilterNode(t, ctx, true)
	defer cleanup()

	builder.L2Info.GenerateAccount("FilteredUser")
	builder.L2Info.GenerateAccount("NormalUser")
	builder.L2.TransferBalance(t, "Owner", "NormalUser", big.NewInt(1e18), builder.L2Info)
	builder.L2.TransferBalance(t, "Owner", "FilteredUser", big.NewInt(1e18), builder.L2Info)

	filteredAddr := builder.L2Info.GetAddress("FilteredUser")
	normalAddr := builder.L2Info.GetAddress("NormalUser")
	filter := newHashedChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(t, filter)

	// eth_call TO filtered address should fail
	_, err := builder.L2.Client.CallContract(ctx, ethereum.CallMsg{
		From:  normalAddr,
		To:    &filteredAddr,
		Value: big.NewInt(1e12),
	}, nil)
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error for eth_call TO filtered address, got: %v", err)
	}

	// eth_call FROM filtered address should fail
	_, err = builder.L2.Client.CallContract(ctx, ethereum.CallMsg{
		From:  filteredAddr,
		To:    &normalAddr,
		Value: big.NewInt(1e12),
	}, nil)
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error for eth_call FROM filtered address, got: %v", err)
	}

	// eth_call between clean addresses should succeed
	_, err = builder.L2.Client.CallContract(ctx, ethereum.CallMsg{
		From:  normalAddr,
		To:    &normalAddr,
		Value: big.NewInt(1e12),
	}, nil)
	Require(t, err)
}

// TestEthCallFilterDisabled verifies that eth_call does not reject
// calls to filtered addresses when EnableETHCallFilter is false.
func TestEthCallFilterDisabled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder, cleanup := buildRPCFilterNode(t, ctx, false)
	defer cleanup()

	builder.L2Info.GenerateAccount("FilteredUser")
	builder.L2Info.GenerateAccount("NormalUser")
	builder.L2.TransferBalance(t, "Owner", "NormalUser", big.NewInt(1e18), builder.L2Info)
	builder.L2.TransferBalance(t, "Owner", "FilteredUser", big.NewInt(1e18), builder.L2Info)

	filteredAddr := builder.L2Info.GetAddress("FilteredUser")
	normalAddr := builder.L2Info.GetAddress("NormalUser")
	filter := newHashedChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(t, filter)

	// eth_call TO filtered address should succeed when RPC filter is disabled
	_, err := builder.L2.Client.CallContract(ctx, ethereum.CallMsg{
		From:  normalAddr,
		To:    &filteredAddr,
		Value: big.NewInt(1e12),
	}, nil)
	Require(t, err)
}

// TestEthCallFilterPreservesResultWithScheduledTxes verifies that address
// filtering does not alter the eth_call result when scheduled transactions
// (retryable redeems) are involved.
func TestEthCallFilterPreservesResultWithScheduledTxes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Build node with L1 and filtering enabled
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.isSequencer = true
	builder.execConfig.TransactionFiltering.EnableETHCallFilter = true
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("User")
	builder.L2Info.GenerateAccount("Unrelated")
	builder.L2.TransferBalance(t, "Owner", "User", big.NewInt(1e18), builder.L2Info)

	userAddr := builder.L2Info.GetAddress("User")
	unrelatedAddr := builder.L2Info.GetAddress("Unrelated")

	// Submit retryable via L1 with gasLimit=0 (no auto-redeem) so the
	// ticket survives for manual redeem via eth_call.
	delayedInbox, err := bridgegen.NewInbox(builder.L1Info.GetAddress("Inbox"), builder.L1.Client)
	Require(t, err)

	deposit := arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))
	l1opts := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)
	l1opts.Value = deposit
	l1tx, err := delayedInbox.CreateRetryableTicket(
		&l1opts,
		userAddr,
		big.NewInt(1e6),
		big.NewInt(1e16),
		userAddr,
		userAddr,
		big.NewInt(0), // gasLimit=0 → no auto-redeem
		big.NewInt(l2pricing.InitialBaseFeeWei*2),
		nil,
	)
	Require(t, err)
	l1Receipt, err := builder.L1.EnsureTxSucceeded(l1tx)
	Require(t, err)

	// Extract ticket ID and wait for it on L2
	ticketId := lookupSubmissionTxHash(t, ctx, builder, l1Receipt)
	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 30)
	_, err = WaitForTx(ctx, builder.L2.Client, ticketId, 30*time.Second)
	Require(t, err)

	// Craft eth_call data: ArbRetryableTx.redeem(ticketId)
	arbRetryableABI, err := precompilesgen.ArbRetryableTxMetaData.GetAbi()
	Require(t, err)
	redeemData, err := arbRetryableABI.Pack("redeem", ticketId)
	Require(t, err)
	arbRetryableAddr := common.HexToAddress("0x6e")

	callMsg := ethereum.CallMsg{
		From: userAddr,
		To:   &arbRetryableAddr,
		Data: redeemData,
	}

	// eth_call WITHOUT address checker — filtering path runs but checker is nil
	resultWithoutChecker, err := builder.L2.Client.CallContract(ctx, callMsg, nil)
	Require(t, err)

	// Set address checker with an unrelated address (not involved in the call)
	filter := newHashedChecker([]common.Address{unrelatedAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(t, filter)

	// eth_call WITH address checker — full filtering path with active checker
	resultWithChecker, err := builder.L2.Client.CallContract(ctx, callMsg, nil)
	Require(t, err)

	// Results must be identical — filtering must not alter the return value
	if !bytes.Equal(resultWithoutChecker, resultWithChecker) {
		t.Fatalf("eth_call results differ with filtering active:\n  without checker: %x\n  with checker:    %x",
			resultWithoutChecker, resultWithChecker)
	}
}
