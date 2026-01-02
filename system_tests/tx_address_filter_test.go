// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
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

// sendDelayedTx sends a transaction via L1 delayed inbox and waits for processing.
// It does NOT verify success - caller must check state to determine outcome.
func sendDelayedTx(t *testing.T, ctx context.Context, builder *NodeBuilder, tx *types.Transaction) {
	t.Helper()
	delayedInbox, err := bridgegen.NewInbox(builder.L1Info.GetAddress("Inbox"), builder.L1.Client)
	Require(t, err)

	txbytes, err := tx.MarshalBinary()
	Require(t, err)
	txwrapped := append([]byte{arbos.L2MessageKind_SignedTx}, txbytes...)

	l1opts := builder.L1Info.GetDefaultTransactOpts("User", ctx)
	l1tx, err := delayedInbox.SendL2Message(&l1opts, txwrapped)
	Require(t, err)
	_, err = builder.L1.EnsureTxSucceeded(l1tx)
	Require(t, err)

	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 30)
	<-time.After(time.Second * 2)
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

func TestAddressFilterDelayedDirectTransfer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Need L1 for delayed messages
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.isSequencer = true
	cleanup := builder.Build(t)
	defer cleanup()

	// Create and fund accounts
	builder.L2Info.GenerateAccount("FilteredUser")
	builder.L2Info.GenerateAccount("NormalUser")
	builder.L2.TransferBalance(t, "Owner", "NormalUser", big.NewInt(1e18), builder.L2Info)

	// Get initial balance of filtered user
	filteredAddr := builder.L2Info.GetAddress("FilteredUser")
	initialBalance, err := builder.L2.Client.BalanceAt(ctx, filteredAddr, nil)
	Require(t, err)

	// Set up address filter to block FilteredUser
	filter := txfilter.NewStaticAsyncChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Send a delayed message TO the filtered address via L1
	delayedTx := builder.L2Info.PrepareTx("NormalUser", "FilteredUser", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	sendDelayedTx(t, ctx, builder, delayedTx)

	// Check that the balance didn't change (message was filtered)
	finalBalance, err := builder.L2.Client.BalanceAt(ctx, filteredAddr, nil)
	Require(t, err)
	if finalBalance.Cmp(initialBalance) != 0 {
		t.Fatalf("expected filtered address balance to remain %v, got %v", initialBalance, finalBalance)
	}

	// Now disable the filter and verify a normal delayed message works
	emptyFilter := txfilter.NewStaticAsyncChecker([]common.Address{})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(emptyFilter)

	// Reset nonce since the filtered tx wasn't processed
	builder.L2Info.GetInfoWithPrivKey("NormalUser").Nonce.Store(0)

	// Send another delayed message to a different (non-filtered) address
	builder.L2Info.GenerateAccount("CleanUser")
	cleanAddr := builder.L2Info.GetAddress("CleanUser")
	cleanInitialBalance, err := builder.L2.Client.BalanceAt(ctx, cleanAddr, nil)
	Require(t, err)

	delayedTx2 := builder.L2Info.PrepareTx("NormalUser", "CleanUser", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	builder.L1.SendSignedTx(t, builder.L2.Client, delayedTx2, builder.L1Info)

	// Verify the clean transaction succeeded
	cleanFinalBalance, err := builder.L2.Client.BalanceAt(ctx, cleanAddr, nil)
	Require(t, err)
	expectedBalance := new(big.Int).Add(cleanInitialBalance, big.NewInt(1e12))
	if cleanFinalBalance.Cmp(expectedBalance) != 0 {
		t.Fatalf("expected clean address balance to be %v, got %v", expectedBalance, cleanFinalBalance)
	}
}

func TestAddressFilterDelayedCall(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.isSequencer = true
	cleanup := builder.Build(t)
	defer cleanup()

	// Deploy caller contract (not filtered)
	callerAddr, caller := deployAddressFilterTestContract(t, ctx, builder)

	// Deploy target contract (will be filtered)
	targetAddr, _ := deployAddressFilterTestContract(t, ctx, builder)

	// Set up filter to block the target contract
	filter := txfilter.NewStaticAsyncChecker([]common.Address{targetAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Prepare CALL tx with NoSend
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	auth.NoSend = true
	auth.GasLimit = 1000000
	tx, err := caller.CallTarget(&auth, targetAddr)
	Require(t, err)

	// Send via delayed inbox
	sendDelayedTx(t, ctx, builder, tx)

	// Verify target's dummy value unchanged (tx was filtered)
	// Note: CallTarget just calls the target with empty data; the target's dummy wouldn't
	// change anyway. The key verification is that the caller's state didn't change either.
	callerContract, err := localgen.NewAddressFilterTest(callerAddr, builder.L2.Client)
	Require(t, err)
	callerDummy, err := callerContract.Dummy(nil)
	Require(t, err)
	if callerDummy.Cmp(big.NewInt(0)) != 0 {
		t.Fatalf("expected caller dummy to remain 0, got %v", callerDummy)
	}

	// Clear filter and verify unfiltered call via delayed tx works
	emptyFilter := txfilter.NewStaticAsyncChecker([]common.Address{})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(emptyFilter)

	// Sync nonce - filtered tx wasn't included so on-chain nonce didn't increment
	ownerAddr := builder.L2Info.GetAddress("Owner")
	currentNonce, err := builder.L2.Client.NonceAt(ctx, ownerAddr, nil)
	Require(t, err)
	builder.L2Info.GetInfoWithPrivKey("Owner").Nonce.Store(currentNonce)

	// Use the same caller to call a non-filtered target - verify nonce advances
	cleanTargetAddr, _ := deployAddressFilterTestContract(t, ctx, builder)

	// Get nonce before delayed tx
	nonceBefore, err := builder.L2.Client.NonceAt(ctx, ownerAddr, nil)
	Require(t, err)

	auth = builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	auth.NoSend = true
	auth.GasLimit = 1000000
	tx, err = caller.CallTarget(&auth, cleanTargetAddr)
	Require(t, err)

	sendDelayedTx(t, ctx, builder, tx)

	// Verify nonce incremented (tx was processed, not filtered)
	nonceAfter, err := builder.L2.Client.NonceAt(ctx, ownerAddr, nil)
	Require(t, err)
	if nonceAfter <= nonceBefore {
		t.Fatalf("expected nonce to increment after unfiltered delayed tx, was %v, now %v", nonceBefore, nonceAfter)
	}
}

func TestAddressFilterDelayedStaticCall(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.isSequencer = true
	cleanup := builder.Build(t)
	defer cleanup()

	// Deploy caller contract (not filtered)
	callerAddr, caller := deployAddressFilterTestContract(t, ctx, builder)

	// Deploy target contract (will be filtered)
	targetAddr, _ := deployAddressFilterTestContract(t, ctx, builder)

	// Get initial caller dummy value (StaticcallTargetInTx increments this)
	callerContract, err := localgen.NewAddressFilterTest(callerAddr, builder.L2.Client)
	Require(t, err)
	initialDummy, err := callerContract.Dummy(nil)
	Require(t, err)

	// Set up filter to block the target contract
	filter := txfilter.NewStaticAsyncChecker([]common.Address{targetAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Prepare STATICCALL tx with NoSend
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	auth.NoSend = true
	auth.GasLimit = 1000000
	tx, err := caller.StaticcallTargetInTx(&auth, targetAddr)
	Require(t, err)

	// Send via delayed inbox
	sendDelayedTx(t, ctx, builder, tx)

	// Verify caller's dummy value unchanged (tx was filtered/reverted)
	finalDummy, err := callerContract.Dummy(nil)
	Require(t, err)
	if finalDummy.Cmp(initialDummy) != 0 {
		t.Fatalf("expected caller dummy to remain %v, got %v", initialDummy, finalDummy)
	}

	// Clear filter and verify staticcall works
	emptyFilter := txfilter.NewStaticAsyncChecker([]common.Address{})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(emptyFilter)

	// Sync nonce - filtered tx wasn't included so on-chain nonce didn't increment
	ownerAddr := builder.L2Info.GetAddress("Owner")
	currentNonce, err := builder.L2.Client.NonceAt(ctx, ownerAddr, nil)
	Require(t, err)
	builder.L2Info.GetInfoWithPrivKey("Owner").Nonce.Store(currentNonce)

	// Deploy clean target and staticcall it
	cleanTargetAddr, _ := deployAddressFilterTestContract(t, ctx, builder)

	auth = builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	auth.NoSend = true
	auth.GasLimit = 1000000
	tx, err = caller.StaticcallTargetInTx(&auth, cleanTargetAddr)
	Require(t, err)

	sendDelayedTx(t, ctx, builder, tx)

	cleanFinalDummy, err := callerContract.Dummy(nil)
	Require(t, err)
	expectedDummy := new(big.Int).Add(initialDummy, big.NewInt(1))
	if cleanFinalDummy.Cmp(expectedDummy) != 0 {
		t.Fatalf("expected caller dummy to be %v, got %v", expectedDummy, cleanFinalDummy)
	}
}

func TestAddressFilterDelayedCreate2(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.isSequencer = true
	cleanup := builder.Build(t)
	defer cleanup()

	// Deploy caller contract
	_, caller := deployAddressFilterTestContract(t, ctx, builder)

	// Compute CREATE2 address for a known salt
	salt := [32]byte{1, 2, 3}
	create2Addr, err := caller.ComputeCreate2Address(nil, salt)
	Require(t, err)

	// Verify no code at that address yet
	code, err := builder.L2.Client.CodeAt(ctx, create2Addr, nil)
	Require(t, err)
	if len(code) > 0 {
		t.Fatal("expected no code at CREATE2 address before deployment")
	}

	// Set up filter to block the computed address
	filter := txfilter.NewStaticAsyncChecker([]common.Address{create2Addr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Prepare CREATE2 tx with NoSend
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	auth.NoSend = true
	auth.GasLimit = 1000000
	tx, err := caller.Create2Contract(&auth, salt)
	Require(t, err)

	// Send via delayed inbox
	sendDelayedTx(t, ctx, builder, tx)

	// Verify no code deployed (tx was filtered)
	code, err = builder.L2.Client.CodeAt(ctx, create2Addr, nil)
	Require(t, err)
	if len(code) > 0 {
		t.Fatal("expected no code at filtered CREATE2 address after filtered tx")
	}

	// Clear filter and deploy with different salt
	emptyFilter := txfilter.NewStaticAsyncChecker([]common.Address{})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(emptyFilter)

	// Sync nonce - filtered tx wasn't included so on-chain nonce didn't increment
	ownerAddr := builder.L2Info.GetAddress("Owner")
	currentNonce, err := builder.L2.Client.NonceAt(ctx, ownerAddr, nil)
	Require(t, err)
	builder.L2Info.GetInfoWithPrivKey("Owner").Nonce.Store(currentNonce)

	differentSalt := [32]byte{4, 5, 6}
	differentAddr, err := caller.ComputeCreate2Address(nil, differentSalt)
	Require(t, err)

	auth = builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	auth.NoSend = true
	auth.GasLimit = 1000000
	tx, err = caller.Create2Contract(&auth, differentSalt)
	Require(t, err)

	sendDelayedTx(t, ctx, builder, tx)

	// Verify code deployed at non-filtered address
	code, err = builder.L2.Client.CodeAt(ctx, differentAddr, nil)
	Require(t, err)
	if len(code) == 0 {
		t.Fatal("expected code at non-filtered CREATE2 address")
	}
}

func TestAddressFilterDelayedCreate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.isSequencer = true
	cleanup := builder.Build(t)
	defer cleanup()

	// Deploy caller contract
	callerAddr, caller := deployAddressFilterTestContract(t, ctx, builder)

	// Get the current nonce of the caller contract
	nonce, err := builder.L2.Client.NonceAt(ctx, callerAddr, nil)
	Require(t, err)

	// Compute CREATE address based on caller's address and nonce
	createAddr := crypto.CreateAddress(callerAddr, nonce)

	// Verify no code at that address yet
	code, err := builder.L2.Client.CodeAt(ctx, createAddr, nil)
	Require(t, err)
	if len(code) > 0 {
		t.Fatal("expected no code at CREATE address before deployment")
	}

	// Set up filter to block the computed address
	filter := txfilter.NewStaticAsyncChecker([]common.Address{createAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Prepare CREATE tx with NoSend
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	auth.NoSend = true
	auth.GasLimit = 1000000
	tx, err := caller.CreateContract(&auth)
	Require(t, err)

	// Send via delayed inbox
	sendDelayedTx(t, ctx, builder, tx)

	// Verify no code deployed (tx was filtered)
	code, err = builder.L2.Client.CodeAt(ctx, createAddr, nil)
	Require(t, err)
	if len(code) > 0 {
		t.Fatal("expected no code at filtered CREATE address after filtered tx")
	}

	// Clear filter and retry
	emptyFilter := txfilter.NewStaticAsyncChecker([]common.Address{})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(emptyFilter)

	// Sync nonce - filtered tx wasn't included so on-chain nonce didn't increment
	ownerAddr := builder.L2Info.GetAddress("Owner")
	currentNonce, err := builder.L2.Client.NonceAt(ctx, ownerAddr, nil)
	Require(t, err)
	builder.L2Info.GetInfoWithPrivKey("Owner").Nonce.Store(currentNonce)

	// Deploy a fresh caller to get a new CREATE address (original address was filtered)
	callerAddr2, caller2 := deployAddressFilterTestContract(t, ctx, builder)
	nonce2, err := builder.L2.Client.NonceAt(ctx, callerAddr2, nil)
	Require(t, err)
	createAddr2 := crypto.CreateAddress(callerAddr2, nonce2)

	auth = builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	auth.NoSend = true
	auth.GasLimit = 1000000
	tx, err = caller2.CreateContract(&auth)
	Require(t, err)

	sendDelayedTx(t, ctx, builder, tx)

	// Verify code deployed at non-filtered address
	code, err = builder.L2.Client.CodeAt(ctx, createAddr2, nil)
	Require(t, err)
	if len(code) == 0 {
		t.Fatal("expected code at non-filtered CREATE address")
	}
}

func TestAddressFilterDelayedSelfdestruct(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.isSequencer = true
	cleanup := builder.Build(t)
	defer cleanup()

	// Deploy contract and fund it with ETH via normal tx
	contractAddr, contract := deployAddressFilterTestContract(t, ctx, builder)
	builder.L2Info.SetFullAccountInfo("Contract", &AccountInfo{Address: contractAddr})
	builder.L2.TransferBalance(t, "Owner", "Contract", big.NewInt(1e15), builder.L2Info)

	// Create filtered beneficiary
	builder.L2Info.GenerateAccount("FilteredBeneficiary")
	filteredAddr := builder.L2Info.GetAddress("FilteredBeneficiary")
	initialBalance, err := builder.L2.Client.BalanceAt(ctx, filteredAddr, nil)
	Require(t, err)

	// Set up filter to block the beneficiary
	filter := txfilter.NewStaticAsyncChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Prepare SELFDESTRUCT tx with NoSend - only this part uses delayed inbox
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	auth.NoSend = true
	auth.GasLimit = 1000000
	tx, err := contract.SelfDestructTo(&auth, filteredAddr)
	Require(t, err)

	// Send via delayed inbox
	sendDelayedTx(t, ctx, builder, tx)

	// Verify beneficiary balance unchanged (tx was filtered)
	finalBalance, err := builder.L2.Client.BalanceAt(ctx, filteredAddr, nil)
	Require(t, err)
	if finalBalance.Cmp(initialBalance) != 0 {
		t.Fatalf("expected filtered beneficiary balance to remain %v, got %v", initialBalance, finalBalance)
	}

	// Clear filter and test with non-filtered beneficiary
	emptyFilter := txfilter.NewStaticAsyncChecker([]common.Address{})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(emptyFilter)

	// Sync nonce - filtered tx wasn't included so on-chain nonce didn't increment
	ownerAddr := builder.L2Info.GetAddress("Owner")
	currentNonce, err := builder.L2.Client.NonceAt(ctx, ownerAddr, nil)
	Require(t, err)
	builder.L2Info.GetInfoWithPrivKey("Owner").Nonce.Store(currentNonce)

	// Deploy new contract and fund it via normal tx
	contractAddr2, contract2 := deployAddressFilterTestContract(t, ctx, builder)
	builder.L2Info.SetFullAccountInfo("Contract2", &AccountInfo{Address: contractAddr2})
	builder.L2.TransferBalance(t, "Owner", "Contract2", big.NewInt(1e15), builder.L2Info)

	builder.L2Info.GenerateAccount("CleanBeneficiary")
	cleanAddr := builder.L2Info.GetAddress("CleanBeneficiary")
	cleanInitialBalance, err := builder.L2.Client.BalanceAt(ctx, cleanAddr, nil)
	Require(t, err)

	auth = builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	auth.NoSend = true
	auth.GasLimit = 1000000
	tx, err = contract2.SelfDestructTo(&auth, cleanAddr)
	Require(t, err)

	sendDelayedTx(t, ctx, builder, tx)

	// Verify clean beneficiary received ETH
	cleanFinalBalance, err := builder.L2.Client.BalanceAt(ctx, cleanAddr, nil)
	Require(t, err)
	if cleanFinalBalance.Cmp(cleanInitialBalance) <= 0 {
		t.Fatalf("expected clean beneficiary balance to increase, was %v, got %v", cleanInitialBalance, cleanFinalBalance)
	}
}
