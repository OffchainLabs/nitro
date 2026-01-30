// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/execution/gethexec/eventfilter"
	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/txfilter"
)

func isFilteredError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "internal error")
}

func TestAddressFilterDirectTransfer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.isSequencer = true
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
	filter := txfilter.NewStaticAsyncChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Test 1: Transaction TO a filtered address should fail
	tx := builder.L2Info.PrepareTx("NormalUser", "FilteredUser", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	err := builder.L2.Client.SendTransaction(ctx, tx)
	if err == nil {
		t.Fatal("expected transaction to filtered address to be rejected")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}
	// Reset nonce since tx was rejected
	builder.L2Info.GetInfoWithPrivKey("NormalUser").Nonce.Store(0)

	// Test 2: Transaction FROM a filtered address should fail
	tx = builder.L2Info.PrepareTx("FilteredUser", "NormalUser", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	err = builder.L2.Client.SendTransaction(ctx, tx)
	if err == nil {
		t.Fatal("expected transaction from filtered address to be rejected")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}
	// Reset nonce since tx was rejected
	builder.L2Info.GetInfoWithPrivKey("FilteredUser").Nonce.Store(0)

	// Test 3: Transaction between non-filtered addresses should succeed
	builder.L2Info.GenerateAccount("AnotherUser")
	builder.L2.TransferBalance(t, "Owner", "AnotherUser", big.NewInt(1e18), builder.L2Info)
	tx = builder.L2Info.PrepareTx("NormalUser", "AnotherUser", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	err = builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
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
	cleanup := builder.Build(t)
	defer cleanup()

	// Deploy caller contract (not filtered)
	_, caller := deployAddressFilterTestContract(t, ctx, builder)

	// Deploy target contract (will be filtered)
	targetAddr, _ := deployAddressFilterTestContract(t, ctx, builder)

	// Set up filter to block the target contract
	filter := txfilter.NewStaticAsyncChecker([]common.Address{targetAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Test: CALL to filtered address should fail
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	_, err := caller.CallTarget(&auth, targetAddr)
	if err == nil {
		t.Fatal("expected CALL to filtered address to be rejected")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}

	// Deploy another target (not filtered) - should succeed
	cleanTargetAddr, _ := deployAddressFilterTestContract(t, ctx, builder)
	auth = builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	tx, err := caller.CallTarget(&auth, cleanTargetAddr)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
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
	filter := txfilter.NewStaticAsyncChecker([]common.Address{targetAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

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
	filter := txfilter.NewStaticAsyncChecker([]common.Address{})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

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
	cleanup := builder.Build(t)
	defer cleanup()

	// Deploy caller contract (not filtered)
	_, caller := deployAddressFilterTestContract(t, ctx, builder)

	// Compute the CREATE2 address for a known salt
	salt := [32]byte{1, 2, 3}
	create2Addr, err := caller.ComputeCreate2Address(nil, salt)
	Require(t, err)

	// Set up filter to block the computed address
	filter := txfilter.NewStaticAsyncChecker([]common.Address{create2Addr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Test: CREATE2 to filtered address should fail
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	_, err = caller.Create2Contract(&auth, salt)
	if err == nil {
		t.Fatal("expected CREATE2 to filtered address to be rejected")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}

	// Test: CREATE2 with different salt (different address) should succeed
	differentSalt := [32]byte{4, 5, 6}
	auth = builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	tx, err := caller.Create2Contract(&auth, differentSalt)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
}

func TestAddressFilterCreate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.isSequencer = true
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
	filter := txfilter.NewStaticAsyncChecker([]common.Address{createAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Test: CREATE to filtered address should fail
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	_, err = caller.CreateContract(&auth)
	if err == nil {
		t.Fatal("expected CREATE to filtered address to be rejected")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}

	// Test: CREATE to non-filtered address (after nonce incremented) should succeed
	// Clear the filter to allow the next CREATE
	emptyChecker := txfilter.NewStaticAsyncChecker([]common.Address{})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(emptyChecker)

	auth = builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	tx, err := caller.CreateContract(&auth)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
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
	filter := txfilter.NewStaticAsyncChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

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

	filter := txfilter.NewStaticAsyncChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

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
