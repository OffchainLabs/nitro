// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"bytes"
	"context"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/arbitrum/filter"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/execution/gethexec/addressfilter"
	"github.com/offchainlabs/nitro/execution/gethexec/eventfilter"
	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/util/s3client"
	"github.com/offchainlabs/nitro/util/s3syncer"
)

func isFilteredError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "internal error")
}

func newHashedChecker(addrs []common.Address) *addressfilter.HashedAddressChecker {
	return newHashedCheckerWithScheme(addrs, addressfilter.HashingSchemeStringInput)
}

func newHashedCheckerWithScheme(addrs []common.Address, scheme addressfilter.HashingScheme) *addressfilter.HashedAddressChecker {
	const cacheSize = 100
	store := addressfilter.NewHashStore(cacheSize)
	if len(addrs) > 0 {
		salt, _ := uuid.Parse("3ccf0cbf-b23f-47ba-9c2f-4e7bd672b4c7")
		hashes := make([]common.Hash, len(addrs))
		if scheme == addressfilter.HashingSchemeRawBytesInput {
			for i, addr := range addrs {
				hashes[i] = addressfilter.HashRawBytesInput(salt, addr)
			}
		} else {
			hashPrefix := addressfilter.GetHashStringInputPrefix(salt)
			for i, addr := range addrs {
				hashes[i] = addressfilter.HashStringInputWithPrefix(hashPrefix, addr)
			}
		}
		store.Store(uuid.New(), salt, scheme, hashes, "test")
	}
	checker := addressfilter.NewHashedAddressChecker(store, 4, 8192)
	checker.Start(context.Background())
	return checker
}

func TestAddressFilterDirectTransfer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.isSequencer = true
	filteringReportStack, endpoint := SetupFilteringReport(t)
	builder.execConfig.TransactionFiltering.FilteringReportRPCClient.URL = filteringReportStack.HTTPEndpoint()
	cleanup := builder.Build(t)
	defer cleanup()

	// Create accounts
	builder.L2Info.GenerateAccount("FilteredUser")
	builder.L2Info.GenerateAccount("NormalUser")

	// Fund accounts
	builder.L2.TransferBalance(t, "Owner", "NormalUser", big.NewInt(1e18), builder.L2Info)
	builder.L2.TransferBalance(t, "Owner", "FilteredUser", big.NewInt(1e18), builder.L2Info)

	// Set up address filter to block FilteredUser
	filteredAddr := builder.L2Info.GetAddress("FilteredUser")
	addrFilter := newHashedChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(t, addrFilter)

	// Test 1: Transaction TO a filtered address should fail and produce a report
	tx := builder.L2Info.PrepareTx("NormalUser", "FilteredUser", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	err := builder.L2.Client.SendTransaction(ctx, tx)
	if err == nil {
		t.Fatal("expected transaction to filtered address to be rejected")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}
	report := endpoint.NextReport(t)
	CheckCommonReportFields(t, ctx, builder, report, tx)
	if report.IsDelayed {
		t.Fatal("report should not be marked as delayed")
	}
	foundToReason := false
	for _, fa := range report.FilteredAddresses {
		if fa.Address == filteredAddr {
			if fa.FilterReason.Reason != filter.ReasonTo {
				t.Fatalf("expected filter reason %q for TO address, got %q", filter.ReasonTo, fa.FilterReason.Reason)
			}
			if fa.FilterReason.EventRuleMatch != nil {
				t.Fatal("expected nil EventRuleMatch for direct address filter")
			}
			foundToReason = true
			break
		}
	}
	if !foundToReason {
		t.Fatalf("report should contain filtered address %s with ReasonTo", filteredAddr.Hex())
	}

	// Reset nonce since tx was rejected
	builder.L2Info.GetInfoWithPrivKey("NormalUser").Nonce.Store(0)

	// Test 2: Transaction FROM a filtered address should fail and produce a report
	tx = builder.L2Info.PrepareTx("FilteredUser", "NormalUser", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	err = builder.L2.Client.SendTransaction(ctx, tx)
	if err == nil {
		t.Fatal("expected transaction from filtered address to be rejected")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}

	report2 := endpoint.NextReport(t)
	CheckCommonReportFields(t, ctx, builder, report2, tx)
	if report2.IsDelayed {
		t.Fatal("report2 should not be marked as delayed")
	}
	foundFromReason := false
	for _, fa := range report2.FilteredAddresses {
		if fa.Address == filteredAddr {
			if fa.FilterReason.Reason != filter.ReasonFrom {
				t.Fatalf("expected filter reason %q for FROM address, got %q", filter.ReasonFrom, fa.FilterReason.Reason)
			}
			if fa.FilterReason.EventRuleMatch != nil {
				t.Fatal("expected nil EventRuleMatch for direct address filter")
			}
			foundFromReason = true
			break
		}
	}
	if !foundFromReason {
		t.Fatalf("report2 should contain filtered address %s with ReasonFrom", filteredAddr.Hex())
	}
	// Reset nonce since tx was rejected
	builder.L2Info.GetInfoWithPrivKey("FilteredUser").Nonce.Store(0)

	// Test 3: Transaction between non-filtered addresses should succeed with no report
	builder.L2Info.GenerateAccount("AnotherUser")
	builder.L2.TransferBalance(t, "Owner", "AnotherUser", big.NewInt(1e18), builder.L2Info)
	tx = builder.L2Info.PrepareTx("NormalUser", "AnotherUser", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	err = builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	endpoint.AssertNoReport(t, 500*time.Millisecond)
}

// TestAddressFilterMultipleTxsInBlock sequences a clean / bad-A / clean / bad-B
// / clean pattern into a single block and asserts that every clean tx is
// included and each filtered tx is reported with only its own offending
// address.
func TestAddressFilterMultipleTxsInBlock(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.isSequencer = true
	filteringReportStack, endpoint := SetupFilteringReport(t)
	builder.execConfig.TransactionFiltering.FilteringReportRPCClient.URL = filteringReportStack.HTTPEndpoint()
	cleanup := builder.Build(t)
	defer cleanup()

	// Two distinct filtered addresses, so the test can detect cross-tx
	// attribution leaks where a later tx's report mentions an earlier tx's address.
	builder.L2Info.GenerateAccount("FilteredUserA")
	builder.L2Info.GenerateAccount("FilteredUserB")
	filteredA := builder.L2Info.GetAddress("FilteredUserA")
	filteredB := builder.L2Info.GetAddress("FilteredUserB")

	// One sender per tx: a filtered tx is rejected before execution and never
	// bumps the state nonce, so reusing the same sender for a later tx would
	// fail the nonce check rather than the filter.
	senders := []string{"Sender1", "Sender2", "Sender3", "Sender4", "Sender5"}
	for _, s := range senders {
		builder.L2Info.GenerateAccount(s)
		builder.L2.TransferBalance(t, "Owner", s, big.NewInt(1e18), builder.L2Info)
	}
	builder.L2Info.GenerateAccount("CleanReceiver1")
	builder.L2Info.GenerateAccount("CleanReceiver3")
	builder.L2Info.GenerateAccount("CleanReceiver5")

	addrFilter := newHashedChecker([]common.Address{filteredA, filteredB})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(t, addrFilter)

	clean1 := builder.L2Info.PrepareTx("Sender1", "CleanReceiver1", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	badA := builder.L2Info.PrepareTx("Sender2", "FilteredUserA", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	clean3 := builder.L2Info.PrepareTx("Sender3", "CleanReceiver3", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	badB := builder.L2Info.PrepareTx("Sender4", "FilteredUserB", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	clean5 := builder.L2Info.PrepareTx("Sender5", "CleanReceiver5", builder.L2Info.TransferGas, big.NewInt(1e12), nil)

	sequencer := builder.L2.ExecNode.Sequencer
	// SequenceTransactionsForTest bypasses the tx queue and sequences every
	// tx into a single block.
	sequencer.Pause()
	defer sequencer.Activate()
	block, txErrors := sequencer.SequenceTransactionsForTest(
		t, types.Transactions{clean1, badA, clean3, badB, clean5},
	)
	require.NotNil(t, block, "block should have been produced")
	require.Len(t, txErrors, 5)

	require.NoError(t, txErrors[0], "clean tx 1 should have been sequenced")
	require.Error(t, txErrors[1], "tx to FilteredUserA should have been rejected")
	require.Truef(t, isFilteredError(txErrors[1]), "tx 2 rejection should be a filter error, got: %v", txErrors[1])
	require.NoErrorf(t, txErrors[2], "clean tx 3 must not inherit filter state from tx 2, got: %v", txErrors[2])
	require.Error(t, txErrors[3], "tx to FilteredUserB should have been rejected")
	require.Truef(t, isFilteredError(txErrors[3]), "tx 4 rejection should be a filter error, got: %v", txErrors[3])
	require.NoErrorf(t, txErrors[4], "clean tx 5 must not inherit filter state from tx 4, got: %v", txErrors[4])

	cleanReceipt1, err := builder.L2.EnsureTxSucceeded(clean1)
	require.NoError(t, err)
	cleanReceipt3, err := builder.L2.EnsureTxSucceeded(clean3)
	require.NoError(t, err)
	cleanReceipt5, err := builder.L2.EnsureTxSucceeded(clean5)
	require.NoError(t, err)
	require.Equal(t, cleanReceipt1.BlockNumber.Uint64(), cleanReceipt3.BlockNumber.Uint64(),
		"clean txs must share the same block")
	require.Equal(t, cleanReceipt1.BlockNumber.Uint64(), cleanReceipt5.BlockNumber.Uint64(),
		"clean txs must share the same block")

	// Exactly one report per filtered tx; reports may arrive in any order.
	reports := map[common.Hash]*addressfilter.FilteredTxReport{}
	for i := 0; i < 2; i++ {
		r := endpoint.NextReport(t)
		reports[r.TxHash] = r
	}
	endpoint.AssertNoReport(t, 500*time.Millisecond)

	require.Contains(t, reports, badA.Hash(), "no report received for tx to FilteredUserA")
	require.Contains(t, reports, badB.Hash(), "no report received for tx to FilteredUserB")

	assertReportFilteredOn(t, reports[badA.Hash()], filteredA, filteredB)
	assertReportFilteredOn(t, reports[badB.Hash()], filteredB, filteredA)

	CheckCommonReportFields(t, ctx, builder, reports[badA.Hash()], badA)
	CheckCommonReportFields(t, ctx, builder, reports[badB.Hash()], badB)
}

// assertReportFilteredOn asserts that report mentions `expected` (with
// ReasonTo) and does not mention `notExpected`. This catches cross-tx attribution
// leaks where a previous tx's filtered address is reported against a later tx.
func assertReportFilteredOn(t *testing.T, report *addressfilter.FilteredTxReport, expected, notExpected common.Address) {
	t.Helper()
	foundExpected := false
	for _, fa := range report.FilteredAddresses {
		require.NotEqualf(t, notExpected, fa.Address,
			"report for tx %s leaked filtered address %s from another tx",
			report.TxHash.Hex(), notExpected.Hex())
		if fa.Address == expected {
			require.Equal(t, filter.ReasonTo, fa.FilterReason.Reason,
				"expected filter reason %q, got %q", filter.ReasonTo, fa.FilterReason.Reason)
			foundExpected = true
		}
	}
	require.Truef(t, foundExpected, "report should mention filtered address %s", expected.Hex())
}

func TestAddressFilterEventRuleReport(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up Transfer event filter rule
	transferEvent := "Transfer(address,address,uint256)"
	selector, _, err := eventfilter.CanonicalSelectorFromEvent(transferEvent)
	Require(t, err)
	rules := []eventfilter.EventRule{{
		Event:          transferEvent,
		Selector:       selector,
		TopicAddresses: []int{1, 2},
	}}

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false).WithEventFilterRules(rules)
	builder.isSequencer = true
	filteringReportStack, endpoint := SetupFilteringReport(t)
	builder.execConfig.TransactionFiltering.FilteringReportRPCClient.URL = filteringReportStack.HTTPEndpoint()
	cleanup := builder.Build(t)
	defer cleanup()

	// Deploy test contract
	contractAddr, contract := deployAddressFilterTestContract(t, ctx, builder)

	// Create filtered address and set up address filter
	builder.L2Info.GenerateAccount("FilteredBeneficiary")
	filteredAddr := builder.L2Info.GetAddress("FilteredBeneficiary")
	addrFilter := newHashedChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(t, addrFilter)

	// Emit Transfer event with filtered address as recipient (topic[2])
	// This triggers postTxFilter via the event filter path
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	tx, err := contract.EmitTransfer(&auth, auth.From, filteredAddr)
	if err == nil {
		t.Fatal("expected EmitTransfer to filtered beneficiary to be rejected")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}

	report := endpoint.NextReport(t)
	CheckCommonReportFields(t, ctx, builder, report, tx)
	if report.IsDelayed {
		t.Fatal("report should not be marked as delayed")
	}
	// Verify that the report contains the filtered address with an EventRuleMatch
	foundEventRule := false
	for _, fa := range report.FilteredAddresses {
		if fa.Address == filteredAddr && fa.FilterReason.Reason == filter.ReasonEventRule {
			if fa.FilterReason.EventRuleMatch == nil {
				t.Fatal("expected non-nil EventRuleMatch for event rule filter")
			}
			if fa.FilterReason.EventRuleMatch.MatchedEvent != transferEvent {
				t.Fatalf("expected MatchedEvent %q, got %q", transferEvent, fa.FilterReason.EventRuleMatch.MatchedEvent)
			}
			if fa.FilterReason.EventRuleMatch.MatchedTopicIndex != 2 {
				t.Fatalf("expected MatchedTopicIndex 2, got %d", fa.FilterReason.EventRuleMatch.MatchedTopicIndex)
			}
			rawLog := fa.FilterReason.EventRuleMatch.RawLog
			if rawLog == nil {
				t.Fatal("expected non-nil RawLog in EventRuleMatch")
			}
			if rawLog.Address != contractAddr {
				t.Fatalf("expected RawLog.Address %s, got %s", contractAddr.Hex(), rawLog.Address.Hex())
			}
			if len(rawLog.Topics) != 3 {
				t.Fatalf("expected 3 topics, got %d", len(rawLog.Topics))
			}
			expectedSelector := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
			if rawLog.Topics[0] != expectedSelector {
				t.Fatalf("expected topic[0] %s, got %s", expectedSelector.Hex(), rawLog.Topics[0].Hex())
			}
			if rawLog.Topics[1] != common.BytesToHash(auth.From.Bytes()) {
				t.Fatalf("expected topic[1] to contain Owner address %s, got %s", auth.From.Hex(), rawLog.Topics[1].Hex())
			}
			if rawLog.Topics[2] != common.BytesToHash(filteredAddr.Bytes()) {
				t.Fatalf("expected topic[2] to contain filtered address %s, got %s", filteredAddr.Hex(), rawLog.Topics[2].Hex())
			}
			expectedData := common.BigToHash(big.NewInt(1)).Bytes()
			if !bytes.Equal(rawLog.Data, expectedData) {
				t.Fatalf("expected RawLog.Data %x, got %x", expectedData, rawLog.Data)
			}
			foundEventRule = true
			break
		}
	}
	if !foundEventRule {
		t.Fatalf("report should contain filtered address %s with ReasonEventRule and EventRuleMatch", filteredAddr.Hex())
	}
}

func deployAddressFilterTestContract(t *testing.T, ctx context.Context, builder *NodeBuilder) (common.Address, *localgen.AddressFilterTest) {
	t.Helper()
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	addr, tx, contract, err := localgen.DeployAddressFilterTest(&auth, builder.L2.Client)
	Require(t, err, "could not deploy AddressFilterTest contract")
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	return addr, contract
}

func TestAddressFilterCall(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.isSequencer = true
	filteringReportStack, endpoint := SetupFilteringReport(t)
	builder.execConfig.TransactionFiltering.FilteringReportRPCClient.URL = filteringReportStack.HTTPEndpoint()
	cleanup := builder.Build(t)
	defer cleanup()

	// Deploy caller contract (not filtered)
	_, caller := deployAddressFilterTestContract(t, ctx, builder)

	// Deploy target contract (will be filtered)
	targetAddr, _ := deployAddressFilterTestContract(t, ctx, builder)

	// Set up filter to block the target contract
	checker := newHashedChecker([]common.Address{targetAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(t, checker)

	// Test: CALL to filtered address should fail
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	tx, err := caller.CallTarget(&auth, targetAddr)
	if err == nil {
		t.Fatal("expected CALL to filtered address to be rejected")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}

	report := endpoint.NextReport(t)
	CheckCommonReportFields(t, ctx, builder, report, tx)
	if report.IsDelayed {
		t.Fatal("report should not be marked as delayed")
	}
	foundTarget := false
	for _, fa := range report.FilteredAddresses {
		if fa.Address == targetAddr && fa.FilterReason.Reason == filter.ReasonCallTarget {
			if fa.FilterReason.EventRuleMatch != nil {
				t.Fatal("expected nil EventRuleMatch for direct address filter via CALL")
			}
			foundTarget = true
			break
		}
	}
	if !foundTarget {
		t.Fatalf("report should contain filtered target address %s with reason %s", targetAddr.Hex(), filter.ReasonCallTarget)
	}

	// Deploy another target (not filtered) - should succeed
	cleanTargetAddr, _ := deployAddressFilterTestContract(t, ctx, builder)
	auth = builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	tx, err = caller.CallTarget(&auth, cleanTargetAddr)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	endpoint.AssertNoReport(t, 500*time.Millisecond)
}

func TestAddressFilterStaticCall(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.isSequencer = true
	cleanup := builder.Build(t)
	defer cleanup()

	// Deploy caller contract (not filtered)
	_, caller := deployAddressFilterTestContract(t, ctx, builder)

	// Deploy target contract (will be filtered)
	targetAddr, _ := deployAddressFilterTestContract(t, ctx, builder)

	// Set up filter to block the target contract
	filter := newHashedChecker([]common.Address{targetAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(t, filter)

	// Test: STATICCALL to filtered address within a transaction should fail
	// We use staticcallTargetInTx which does a state change + staticcall
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	_, err := caller.StaticcallTargetInTx(&auth, targetAddr)
	if err == nil {
		t.Fatal("expected STATICCALL to filtered address to be rejected")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}

	// Deploy another target (not filtered) - should succeed
	cleanTargetAddr, _ := deployAddressFilterTestContract(t, ctx, builder)
	auth = builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	tx, err := caller.StaticcallTargetInTx(&auth, cleanTargetAddr)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
}

// runInnerCallFilterTest exercises an inner CALL-family opcode (CALL / DELEGATECALL / CALLCODE)
// from a wrapper contract to either a filtered EOA or a filtered contract, and asserts that the
// target appears in the report with ReasonCallTarget.
func runInnerCallFilterTest(t *testing.T, useEOATarget bool, invoke func(*localgen.AddressFilterTest, *bind.TransactOpts, common.Address) (*types.Transaction, error)) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.isSequencer = true
	filteringReportStack, endpoint := SetupFilteringReport(t)
	builder.execConfig.TransactionFiltering.FilteringReportRPCClient.URL = filteringReportStack.HTTPEndpoint()
	cleanup := builder.Build(t)
	defer cleanup()

	_, caller := deployAddressFilterTestContract(t, ctx, builder)

	var filteredAddr common.Address
	if useEOATarget {
		builder.L2Info.GenerateAccount("FilteredEOA")
		filteredAddr = builder.L2Info.GetAddress("FilteredEOA")
	} else {
		filteredAddr, _ = deployAddressFilterTestContract(t, ctx, builder)
	}

	checker := newHashedChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(t, checker)

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	tx, err := invoke(caller, &auth, filteredAddr)
	if err == nil {
		t.Fatal("expected inner call to filtered address to be rejected")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}

	report := endpoint.NextReport(t)
	CheckCommonReportFields(t, ctx, builder, report, tx)
	if report.IsDelayed {
		t.Fatal("report should not be marked as delayed")
	}
	found := false
	for _, fa := range report.FilteredAddresses {
		if fa.Address == filteredAddr && fa.FilterReason.Reason == filter.ReasonCallTarget {
			if fa.FilterReason.EventRuleMatch != nil {
				t.Fatal("expected nil EventRuleMatch for call-target filter")
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("report should contain filtered address %s with reason %s, got %+v", filteredAddr.Hex(), filter.ReasonCallTarget, report.FilteredAddresses)
	}

	// Sanity: an unfiltered target should pass.
	var cleanAddr common.Address
	if useEOATarget {
		builder.L2Info.GenerateAccount("CleanEOA")
		cleanAddr = builder.L2Info.GetAddress("CleanEOA")
	} else {
		cleanAddr, _ = deployAddressFilterTestContract(t, ctx, builder)
	}
	auth = builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	tx, err = invoke(caller, &auth, cleanAddr)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	endpoint.AssertNoReport(t, 500*time.Millisecond)
}

func TestAddressFilterCallToContract(t *testing.T) {
	runInnerCallFilterTest(t, false, (*localgen.AddressFilterTest).CallTarget)
}

func TestAddressFilterCallToEOA(t *testing.T) {
	runInnerCallFilterTest(t, true, (*localgen.AddressFilterTest).CallTarget)
}

func TestAddressFilterDelegateCallToContract(t *testing.T) {
	runInnerCallFilterTest(t, false, (*localgen.AddressFilterTest).DelegatecallTarget)
}

func TestAddressFilterDelegateCallToEOA(t *testing.T) {
	runInnerCallFilterTest(t, true, (*localgen.AddressFilterTest).DelegatecallTarget)
}

func TestAddressFilterCallCodeToContract(t *testing.T) {
	runInnerCallFilterTest(t, false, (*localgen.AddressFilterTest).CallcodeTarget)
}

func TestAddressFilterCallCodeToEOA(t *testing.T) {
	runInnerCallFilterTest(t, true, (*localgen.AddressFilterTest).CallcodeTarget)
}

func TestAddressFilterStaticCallToContract(t *testing.T) {
	runInnerCallFilterTest(t, false, (*localgen.AddressFilterTest).StaticcallTargetTx)
}

func TestAddressFilterStaticCallToEOA(t *testing.T) {
	runInnerCallFilterTest(t, true, (*localgen.AddressFilterTest).StaticcallTargetTx)
}

// runStylusCallFilterTest deploys multicall.wasm as the Stylus caller and exercises
// the call hostio against a filtered target (contract or EOA).
func runStylusCallFilterTest(t *testing.T, useEOATarget bool) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.isSequencer = true
	filteringReportStack, endpoint := SetupFilteringReport(t)
	builder.execConfig.TransactionFiltering.FilteringReportRPCClient.URL = filteringReportStack.HTTPEndpoint()
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	multicallAddr := deployWasm(t, ctx, auth, builder.L2.Client, rustFile("multicall"))

	var filteredAddr common.Address
	if useEOATarget {
		builder.L2Info.GenerateAccount("FilteredEOA")
		filteredAddr = builder.L2Info.GetAddress("FilteredEOA")
	} else {
		auth2 := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
		filteredAddr = deployWasm(t, ctx, auth2, builder.L2.Client, rustFile("storage"))
	}

	checker := newHashedChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(t, checker)

	args := argsForMulticall(vm.CALL, filteredAddr, nil, nil)
	tx := builder.L2Info.PrepareTxTo("Owner", &multicallAddr, 10_000_000, common.Big0, args)
	err := builder.L2.Client.SendTransaction(ctx, tx)
	if err == nil {
		t.Fatal("expected Stylus CALL to filtered target to be rejected")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}

	report := endpoint.NextReport(t)
	CheckCommonReportFields(t, ctx, builder, report, tx)
	if report.IsDelayed {
		t.Fatal("report should not be marked as delayed")
	}
	found := false
	for _, fa := range report.FilteredAddresses {
		if fa.Address == filteredAddr && fa.FilterReason.Reason == filter.ReasonCallTarget {
			if fa.FilterReason.EventRuleMatch != nil {
				t.Fatal("expected nil EventRuleMatch for Stylus call-target filter")
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("report should contain filtered address %s with reason %s, got %+v", filteredAddr.Hex(), filter.ReasonCallTarget, report.FilteredAddresses)
	}
}

func TestAddressFilterStylusCallToContract(t *testing.T) {
	runStylusCallFilterTest(t, false)
}

func TestAddressFilterStylusCallToEOA(t *testing.T) {
	runStylusCallFilterTest(t, true)
}

func TestAddressFilterDisabled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.isSequencer = true
	cleanup := builder.Build(t)
	defer cleanup()

	// Create account
	builder.L2Info.GenerateAccount("TestUser")
	builder.L2.TransferBalance(t, "Owner", "TestUser", big.NewInt(1e18), builder.L2Info)

	// Set up an empty filter (disabled)
	filter := newHashedChecker([]common.Address{})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(t, filter)

	// All transactions should succeed when filter is disabled
	tx := builder.L2Info.PrepareTx("Owner", "TestUser", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	err := builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	tx = builder.L2Info.PrepareTx("TestUser", "Owner", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	err = builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
}

func TestAddressFilterCreate2(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.isSequencer = true
	filteringReportStack, endpoint := SetupFilteringReport(t)
	builder.execConfig.TransactionFiltering.FilteringReportRPCClient.URL = filteringReportStack.HTTPEndpoint()
	cleanup := builder.Build(t)
	defer cleanup()

	// Deploy caller contract (not filtered)
	_, caller := deployAddressFilterTestContract(t, ctx, builder)

	// Compute the CREATE2 address for a known salt
	salt := [32]byte{1, 2, 3}
	create2Addr, err := caller.ComputeCreate2Address(nil, salt)
	Require(t, err)

	// Set up filter to block the computed address
	checker := newHashedChecker([]common.Address{create2Addr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(t, checker)

	// Test: CREATE2 to filtered address should fail
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	tx, err := caller.Create2Contract(&auth, salt)
	if err == nil {
		t.Fatal("expected CREATE2 to filtered address to be rejected")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}

	report := endpoint.NextReport(t)
	CheckCommonReportFields(t, ctx, builder, report, tx)
	if report.IsDelayed {
		t.Fatal("report should not be marked as delayed")
	}
	foundTarget := false
	for _, fa := range report.FilteredAddresses {
		if fa.Address == create2Addr && fa.FilterReason.Reason == filter.ReasonCreate {
			if fa.FilterReason.EventRuleMatch != nil {
				t.Fatal("expected nil EventRuleMatch for direct address filter via CREATE2")
			}
			foundTarget = true
			break
		}
	}
	if !foundTarget {
		t.Fatalf("report should contain filtered address %s with reason %s, got %+v", create2Addr.Hex(), filter.ReasonCreate, report.FilteredAddresses)
	}

	// Test: CREATE2 with different salt (different address) should succeed
	differentSalt := [32]byte{4, 5, 6}
	auth = builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	tx, err = caller.Create2Contract(&auth, differentSalt)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	endpoint.AssertNoReport(t, 500*time.Millisecond)
}

func TestAddressFilterCreate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.isSequencer = true
	filteringReportStack, endpoint := SetupFilteringReport(t)
	builder.execConfig.TransactionFiltering.FilteringReportRPCClient.URL = filteringReportStack.HTTPEndpoint()
	cleanup := builder.Build(t)
	defer cleanup()

	// Deploy caller contract (not filtered)
	callerAddr, caller := deployAddressFilterTestContract(t, ctx, builder)

	// Get the current nonce of the caller contract
	nonce, err := builder.L2.Client.NonceAt(ctx, callerAddr, nil)
	Require(t, err)

	// Compute the CREATE address based on the caller's address and nonce
	createAddr := crypto.CreateAddress(callerAddr, nonce)

	// Set up filter to block the computed address
	checker := newHashedChecker([]common.Address{createAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(t, checker)

	// Test: CREATE to filtered address should fail
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	tx, err := caller.CreateContract(&auth)
	if err == nil {
		t.Fatal("expected CREATE to filtered address to be rejected")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}

	report := endpoint.NextReport(t)
	CheckCommonReportFields(t, ctx, builder, report, tx)
	if report.IsDelayed {
		t.Fatal("report should not be marked as delayed")
	}
	foundTarget := false
	for _, fa := range report.FilteredAddresses {
		if fa.Address == createAddr && fa.FilterReason.Reason == filter.ReasonCreate {
			if fa.FilterReason.EventRuleMatch != nil {
				t.Fatal("expected nil EventRuleMatch for direct address filter via CREATE")
			}
			foundTarget = true
			break
		}
	}
	if !foundTarget {
		t.Fatalf("report should contain filtered address %s with reason %s, got %+v", createAddr.Hex(), filter.ReasonCreate, report.FilteredAddresses)
	}

	// Test: CREATE to non-filtered address (after nonce incremented) should succeed
	// Clear the filter to allow the next CREATE
	emptyChecker := newHashedChecker([]common.Address{})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(t, emptyChecker)

	auth = builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	tx, err = caller.CreateContract(&auth)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	endpoint.AssertNoReport(t, 500*time.Millisecond)
}

func TestAddressFilterTopLevelDeployment(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.isSequencer = true
	filteringReportStack, endpoint := SetupFilteringReport(t)
	builder.execConfig.TransactionFiltering.FilteringReportRPCClient.URL = filteringReportStack.HTTPEndpoint()
	cleanup := builder.Build(t)
	defer cleanup()

	senderAddr := builder.L2Info.GetAddress("Owner")
	senderNonce, err := builder.L2.Client.NonceAt(ctx, senderAddr, nil)
	Require(t, err)
	// createAddr is the address the EVM will derive for the new contract when
	// state_transition processes the deployment tx (evm.Create -> evm.create).
	// We pre-compute it with the same formula so the filter can block it.
	createAddr := crypto.CreateAddress(senderAddr, senderNonce)

	checker := newHashedChecker([]common.Address{createAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(t, checker)

	// Minimal init code: deploys a 0x35-byte runtime that always reverts. Same bytecode used by
	// AddressFilterTest.createContract — we only need a valid constructor that runs to completion.
	deployCode := common.FromHex("6080604052348015600f57600080fd5b50603580601d6000396000f3fe6080604052600080fdfea164736f6c6343000811000a")

	tx := builder.L2Info.PrepareTxTo("Owner", nil, 10_000_000, common.Big0, deployCode)
	err = builder.L2.Client.SendTransaction(ctx, tx)
	if err == nil {
		t.Fatal("expected top-level deployment to filtered address to be rejected")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}

	report := endpoint.NextReport(t)
	CheckCommonReportFields(t, ctx, builder, report, tx)
	if report.IsDelayed {
		t.Fatal("report should not be marked as delayed")
	}
	foundTarget := false
	for _, fa := range report.FilteredAddresses {
		if fa.Address == createAddr && fa.FilterReason.Reason == filter.ReasonCreate {
			if fa.FilterReason.EventRuleMatch != nil {
				t.Fatal("expected nil EventRuleMatch for direct address filter via top-level deployment")
			}
			foundTarget = true
			break
		}
	}
	if !foundTarget {
		t.Fatalf("report should contain filtered address %s with reason %s, got %+v", createAddr.Hex(), filter.ReasonCreate, report.FilteredAddresses)
	}
}

func TestAddressFilterSelfdestruct(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.isSequencer = true
	cleanup := builder.Build(t)
	defer cleanup()

	// Deploy contract that will selfdestruct
	_, contract := deployAddressFilterTestContract(t, ctx, builder)

	// Create a target address to be filtered (the selfdestruct beneficiary)
	builder.L2Info.GenerateAccount("FilteredBeneficiary")
	filteredAddr := builder.L2Info.GetAddress("FilteredBeneficiary")

	// Set up filter to block the beneficiary
	filter := newHashedChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(t, filter)

	// Test: SELFDESTRUCT to filtered beneficiary should fail
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	_, err := contract.SelfDestructTo(&auth, filteredAddr)
	if err == nil {
		t.Fatal("expected SELFDESTRUCT to filtered beneficiary to be rejected")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}

	// Deploy another contract and test with non-filtered beneficiary
	_, contract2 := deployAddressFilterTestContract(t, ctx, builder)
	builder.L2Info.GenerateAccount("CleanBeneficiary")
	cleanAddr := builder.L2Info.GetAddress("CleanBeneficiary")

	auth = builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	tx, err := contract2.SelfDestructTo(&auth, cleanAddr)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
}

// Test the special scenario introduced by EIP-6780
// Since EIP-6780 behave differently for selfdestruct in constructor vs later calls,
// we need to test both cases. This test covers selfdestruct in constructor.
func TestAddressFilterSelfdestructOnConstruct(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.isSequencer = true
	cleanup := builder.Build(t)
	defer cleanup()

	// Fund sender account
	builder.L2Info.GenerateAccount("Deployer")
	builder.L2.TransferBalance(t, "Owner", "Deployer", big.NewInt(1e18), builder.L2Info)

	// Create filtered beneficiary address
	builder.L2Info.GenerateAccount("FilteredBeneficiary")
	filteredAddr := builder.L2Info.GetAddress("FilteredBeneficiary")

	// Create non-filtered beneficiary address
	builder.L2Info.GenerateAccount("CleanBeneficiary")
	cleanAddr := builder.L2Info.GetAddress("CleanBeneficiary")

	// Set up address filter to block FilteredBeneficiary
	filter := newHashedChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(t, filter)

	// Test 1: Deploy contract that selfdestructs to filtered address in constructor should fail
	auth := builder.L2Info.GetDefaultTransactOpts("Deployer", ctx)
	auth.Value = big.NewInt(1e15) // Send some ETH to be transferred on selfdestruct
	_, _, _, err := localgen.DeploySelfDestructInConstructorWithDestination(&auth, builder.L2.Client, filteredAddr)
	if err == nil {
		t.Fatal("expected deployment with selfdestruct to filtered beneficiary to be rejected")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}

	// Test 2: Deploy contract that selfdestructs to non-filtered address should succeed
	auth = builder.L2Info.GetDefaultTransactOpts("Deployer", ctx)
	auth.Value = big.NewInt(1e15)
	_, tx, _, err := localgen.DeploySelfDestructInConstructorWithDestination(&auth, builder.L2.Client, cleanAddr)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
}

func TestAddressFilterWithFilteredEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	specs := []struct {
		event          string
		topicAddresses []int
	}{
		{
			event:          "Transfer(address,address,uint256)",
			topicAddresses: []int{1, 2},
		},
		{
			event:          "TransferSingle(address,address,address,uint256,uint256)",
			topicAddresses: []int{2, 3},
		},
		{
			event:          "TransferBatch(address,address,address,uint256[],uint256[])",
			topicAddresses: []int{2, 3},
		},
	}

	rules := make([]eventfilter.EventRule, 0, len(specs))
	for _, s := range specs {
		selector, _, err := eventfilter.CanonicalSelectorFromEvent(s.event)
		Require(t, err)

		rules = append(rules, eventfilter.EventRule{
			Event:          s.event,
			Selector:       selector,
			TopicAddresses: s.topicAddresses,
		})
	}

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false).WithEventFilterRules(rules)
	builder.isSequencer = true

	cleanup := builder.Build(t)
	defer cleanup()

	// Deploy test contract
	_, contract := deployAddressFilterTestContract(t, ctx, builder)

	// Create filtered address
	builder.L2Info.GenerateAccount("FilteredBeneficiary")
	filteredAddr := builder.L2Info.GetAddress("FilteredBeneficiary")

	// Create non-filtered address
	builder.L2Info.GenerateAccount("CleanBeneficiary")
	cleanAddr := builder.L2Info.GetAddress("CleanBeneficiary")

	filter := newHashedChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(t, filter)

	// Test 1: Transfer to filtered beneficiary should fail
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	_, err := contract.EmitTransfer(&auth, auth.From, filteredAddr)
	if err == nil {
		t.Fatal("expected EmitTransfer to filtered beneficiary to be rejected")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}

	// Test 2: Transfer from filtered beneficiary should fail
	_, err = contract.EmitTransfer(&auth, filteredAddr, auth.From)
	if err == nil {
		t.Fatal("expected EmitTransfer from filtered beneficiary to be rejected")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}

	// Test 3: Transfer to and from clean beneficiary should succeed
	auth = builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	tx, err := contract.EmitTransfer(&auth, auth.From, cleanAddr)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	tx, err = contract.EmitTransfer(&auth, cleanAddr, auth.From)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// Test 4: TransferSingle involving filtered beneficiary should fail
	_, err = contract.EmitTransferSingle(&auth, auth.From, cleanAddr, filteredAddr)
	if err == nil {
		t.Fatal("expected EmitTransferSingle to filtered beneficiary to be rejected")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}

	_, err = contract.EmitTransferSingle(&auth, auth.From, filteredAddr, cleanAddr)
	if err == nil {
		t.Fatal("expected EmitTransferSingle from filtered beneficiary to be rejected")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}

	// Test 5: TransferBatch involving filtered beneficiary should fail
	_, err = contract.EmitTransferBatch(&auth, auth.From, cleanAddr, filteredAddr)
	if err == nil {
		t.Fatal("expected EmitTransferBatch to filtered beneficiary to be rejected")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}

	_, err = contract.EmitTransferBatch(&auth, auth.From, filteredAddr, cleanAddr)
	if err == nil {
		t.Fatal("expected EmitTransferBatch from filtered beneficiary to be rejected")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}

	// Test 6: UnfilteredEvent should always succeed
	tx, err = contract.EmitUnfiltered(&auth, filteredAddr)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
}

func TestFilteringReadyWithoutAddressFilter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.isSequencer = true
	cleanup := builder.Build(t)
	defer cleanup()

	if !builder.L2.ExecNode.Sequencer.FilteringReady() {
		t.Fatal("FilteringReady should be true when no address filter service is configured")
	}
}

func TestSyncBlockedUntilFilteringReady(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.isSequencer = true
	// Use large MsgLag and SyncInterval to prevent ConsensusExecutionSyncer from overwriting it.
	builder.execConfig.SyncMonitor.MsgLag = time.Hour
	builder.nodeConfig.ConsensusExecutionSyncer.SyncInterval = time.Hour
	cleanup := builder.Build(t)
	defer cleanup()

	execNode := builder.L2.ExecNode

	// Small delay to ensure ConsensusExecutionSyncer's initial sync call
	// (which happens immediately on start) completes before we set our own data.
	time.Sleep(500 * time.Millisecond)

	// Push sync data to make SyncMonitor.Synced return true
	execNode.SyncMonitor.SetConsensusSyncData(&execution.ConsensusSyncData{
		Synced:          true,
		MaxMessageCount: 1,
		UpdatedAt:       time.Now(),
	})

	if !execNode.SyncMonitor.Synced(ctx) {
		t.Fatal("SyncMonitor.Synced should return true after pushing sync data")
	}

	// Create a filter service with valid config but without loaded rules
	filterCfg := &addressfilter.Config{
		S3: s3syncer.Config{
			Config:    s3client.Config{Region: "us-east-1"},
			Bucket:    "test-bucket",
			ObjectKey: "test-key",
		},
		PollInterval:              5 * time.Minute,
		CacheSize:                 100,
		AddressCheckerWorkerCount: 1,
		AddressCheckerQueueSize:   10,
	}
	filterService, err := addressfilter.NewFilterService(filterCfg)
	Require(t, err)
	execNode.Sequencer.SetAddressFilterServiceForTest(t, filterService)

	// Filter service exists but rules haven't been loaded
	if execNode.Sequencer.FilteringReady() {
		t.Fatal("FilteringReady should be false before filter rules are loaded")
	}

	if execNode.Synced(ctx) {
		t.Fatal("Synced should return false when filtering is not ready")
	}

	// Store hashes to the hashstore so FilteringReady returns true
	salt, err := uuid.Parse("3ccf0cbf-b23f-47ba-9c2f-4e7bd672b4c7")
	Require(t, err)
	filterService.GetHashStore().Store(uuid.New(), salt, addressfilter.HashingSchemeStringInput, nil, "test-digest")

	if !execNode.Sequencer.FilteringReady() {
		t.Fatal("FilteringReady should be true after filter rules are loaded")
	}

	if !execNode.Synced(ctx) {
		t.Fatal("Synced should return true when both SyncMonitor is synced and filtering is ready")
	}
}

// Exercises an end-to-end filtering tx flow under the raw-bytes hashing scheme:
// the checker's HashStore is loaded with sha256-rawbytesinput hashes and the
// sequencer must still reject txs to/from a listed address.
func TestAddressFilterDirectTransferRawBytesScheme(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.isSequencer = true
	filteringReportStack, endpoint := SetupFilteringReport(t)
	builder.execConfig.TransactionFiltering.FilteringReportRPCClient.URL = filteringReportStack.HTTPEndpoint()
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("FilteredUser")
	builder.L2Info.GenerateAccount("NormalUser")
	builder.L2.TransferBalance(t, "Owner", "NormalUser", big.NewInt(1e18), builder.L2Info)
	builder.L2.TransferBalance(t, "Owner", "FilteredUser", big.NewInt(1e18), builder.L2Info)

	filteredAddr := builder.L2Info.GetAddress("FilteredUser")
	addrFilter := newHashedCheckerWithScheme([]common.Address{filteredAddr}, addressfilter.HashingSchemeRawBytesInput)
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(t, addrFilter)

	tx := builder.L2Info.PrepareTx("NormalUser", "FilteredUser", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	err := builder.L2.Client.SendTransaction(ctx, tx)
	if err == nil {
		t.Fatal("expected transaction to filtered address to be rejected under raw-bytes scheme")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}
	report := endpoint.NextReport(t)
	CheckCommonReportFields(t, ctx, builder, report, tx)

	// Sanity check: tx between non-filtered addresses succeeds.
	builder.L2Info.GetInfoWithPrivKey("NormalUser").Nonce.Store(0)
	builder.L2Info.GenerateAccount("AnotherUser")
	builder.L2.TransferBalance(t, "Owner", "AnotherUser", big.NewInt(1e18), builder.L2Info)
	tx = builder.L2Info.PrepareTx("NormalUser", "AnotherUser", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	err = builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	endpoint.AssertNoReport(t, 500*time.Millisecond)
}
