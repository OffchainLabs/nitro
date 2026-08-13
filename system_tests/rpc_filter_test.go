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
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
)

// buildRPCFilterNode creates a single sequencer node with RPC filtering
// enabled or disabled. The txFilterer is wired into the backend at construction time.
func buildRPCFilterNode(t *testing.T, ctx context.Context, enableETHCallFilter bool, withL1 bool) (builder *NodeBuilder, s3Filter *fakeS3AddressFilter, cleanup func()) {
	t.Helper()
	builder = NewNodeBuilder(ctx).DefaultConfig(t, withL1)
	builder.isSequencer = true
	builder.execConfig.TransactionFiltering.EnableETHCallFilter = enableETHCallFilter
	s3Filter = setupFakeS3AddressFilter(t, builder)
	cleanup = builder.Build(t)
	return builder, s3Filter, cleanup
}

// TestEstimateGasFilterDirectAddress verifies that eth_estimateGas rejects
// calls involving a filtered address when EnableETHCallFilter is true.
func TestEstimateGasFilterDirectAddress(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder, s3Filter, cleanup := buildRPCFilterNode(t, ctx, true, false)
	defer cleanup()

	builder.L2Info.GenerateAccount("FilteredUser")
	builder.L2Info.GenerateAccount("NormalUser")
	builder.L2.TransferBalance(t, "Owner", "NormalUser", big.NewInt(1e18), builder.L2Info)
	builder.L2.TransferBalance(t, "Owner", "FilteredUser", big.NewInt(1e18), builder.L2Info)

	filteredAddr := builder.L2Info.GetAddress("FilteredUser")
	normalAddr := builder.L2Info.GetAddress("NormalUser")
	s3Filter.setFilteredAddresses(t, ctx, builder.L2.ExecNode, []common.Address{filteredAddr})

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

	builder, s3Filter, cleanup := buildRPCFilterNode(t, ctx, false, false)
	defer cleanup()

	builder.L2Info.GenerateAccount("FilteredUser")
	builder.L2Info.GenerateAccount("NormalUser")
	builder.L2.TransferBalance(t, "Owner", "NormalUser", big.NewInt(1e18), builder.L2Info)
	builder.L2.TransferBalance(t, "Owner", "FilteredUser", big.NewInt(1e18), builder.L2Info)

	filteredAddr := builder.L2Info.GetAddress("FilteredUser")
	normalAddr := builder.L2Info.GetAddress("NormalUser")
	s3Filter.setFilteredAddresses(t, ctx, builder.L2.ExecNode, []common.Address{filteredAddr})

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

	builder, s3Filter, cleanup := buildRPCFilterNode(t, ctx, true, false)
	defer cleanup()

	builder.L2Info.GenerateAccount("FilteredUser")
	builder.L2Info.GenerateAccount("NormalUser")
	builder.L2.TransferBalance(t, "Owner", "NormalUser", big.NewInt(1e18), builder.L2Info)
	builder.L2.TransferBalance(t, "Owner", "FilteredUser", big.NewInt(1e18), builder.L2Info)

	filteredAddr := builder.L2Info.GetAddress("FilteredUser")
	normalAddr := builder.L2Info.GetAddress("NormalUser")
	s3Filter.setFilteredAddresses(t, ctx, builder.L2.ExecNode, []common.Address{filteredAddr})

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

	// eth_call with nil To (contract creation) FROM filtered address should fail
	_, err = builder.L2.Client.CallContract(ctx, ethereum.CallMsg{
		From:  filteredAddr,
		Value: big.NewInt(1e12),
	}, nil)
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error for eth_call with nil To FROM filtered address, got: %v", err)
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

	builder, s3Filter, cleanup := buildRPCFilterNode(t, ctx, false, false)
	defer cleanup()

	builder.L2Info.GenerateAccount("FilteredUser")
	builder.L2Info.GenerateAccount("NormalUser")
	builder.L2.TransferBalance(t, "Owner", "NormalUser", big.NewInt(1e18), builder.L2Info)
	builder.L2.TransferBalance(t, "Owner", "FilteredUser", big.NewInt(1e18), builder.L2Info)

	filteredAddr := builder.L2Info.GetAddress("FilteredUser")
	normalAddr := builder.L2Info.GetAddress("NormalUser")
	s3Filter.setFilteredAddresses(t, ctx, builder.L2.ExecNode, []common.Address{filteredAddr})

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

	builder, s3Filter, cleanup := buildRPCFilterNode(t, ctx, true, true)
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
	delayedBridge, err := arbnode.NewDelayedBridge(builder.L1.Client, builder.L1Info.GetAddress("Bridge"), 0)
	Require(t, err)
	lookupL2Tx := getLookupL2Tx(t, ctx, delayedBridge)

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
	ticketId := lookupL2Tx(l1Receipt).Hash()
	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 30)
	_, err = WaitForTx(ctx, builder.L2.Client, ticketId, 30*time.Second)
	Require(t, err)

	// Craft eth_call data: ArbRetryableTx.redeem(ticketId)
	arbRetryableABI, err := precompilesgen.ArbRetryableTxMetaData.GetAbi()
	Require(t, err)
	redeemData, err := arbRetryableABI.Pack("redeem", ticketId)
	Require(t, err)
	arbRetryableAddr := types.ArbRetryableTxAddress

	callMsg := ethereum.CallMsg{
		From: userAddr,
		To:   &arbRetryableAddr,
		Data: redeemData,
	}

	// Build a second node with transaction filtering fully disabled to obtain
	// a true unfiltered baseline: filtering is enabled at node construction
	// time, so it cannot be turned off on the primary node.
	execConfigB := builder.ExecConfigDefaultTest(t, false)
	unfilteredNode, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{execConfig: execConfigB})
	defer cleanupB()

	// Pin all eth_calls to the ticket's block to avoid flakiness from state
	// changes. The primary node's head can be ahead of the last posted batch,
	// so the ticket block is used because it is guaranteed to be available on
	// the unfiltered node once it syncs the ticket.
	ticketReceipt, err := builder.L2.Client.TransactionReceipt(ctx, ticketId)
	Require(t, err)
	block := ticketReceipt.BlockNumber

	// Wait for the unfiltered node to sync the ticket
	_, err = WaitForTx(ctx, unfilteredNode.Client, ticketId, 30*time.Second)
	Require(t, err)

	// eth_call on the unfiltered node — filtering fully disabled
	resultUnfiltered, err := unfilteredNode.Client.CallContract(ctx, callMsg, block)
	Require(t, err)

	// eth_call with filtering enabled and an empty filter list
	resultEmptyFilterList, err := builder.L2.Client.CallContract(ctx, callMsg, block)
	Require(t, err)

	// Enabling filtering must not alter the return value
	if !bytes.Equal(resultUnfiltered, resultEmptyFilterList) {
		t.Fatalf("eth_call results differ with filtering enabled:\n  unfiltered:        %x\n  empty filter list: %x",
			resultUnfiltered, resultEmptyFilterList)
	}

	// Set address checker with an unrelated address (not involved in the call)
	s3Filter.setFilteredAddresses(t, ctx, builder.L2.ExecNode, []common.Address{unrelatedAddr})

	// eth_call with filtering enabled and an unrelated address in the filter list
	resultUnrelatedFiltered, err := builder.L2.Client.CallContract(ctx, callMsg, block)
	Require(t, err)

	// Filtering an uninvolved address must not alter the return value
	if !bytes.Equal(resultEmptyFilterList, resultUnrelatedFiltered) {
		t.Fatalf("eth_call results differ with populated filter list:\n  empty filter list: %x\n  unrelated filtered: %x",
			resultEmptyFilterList, resultUnrelatedFiltered)
	}

	// Set address checker to filter userAddr
	s3Filter.setFilteredAddresses(t, ctx, builder.L2.ExecNode, []common.Address{userAddr})

	_, err = builder.L2.Client.CallContract(ctx, callMsg, block)
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error for eth_call with filtered retryable address, got: %v", err)
	}
}
