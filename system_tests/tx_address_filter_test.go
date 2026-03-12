// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"crypto/sha256"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/execution/gethexec/addressfilter"
	"github.com/offchainlabs/nitro/execution/gethexec/eventfilter"
	"github.com/offchainlabs/nitro/solgen/go/localgen"
)

func isFilteredError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "internal error")
}

func newHashedChecker(addrs []common.Address) *addressfilter.HashedAddressChecker {
	const cacheSize = 100
	store := addressfilter.NewHashStore(cacheSize)
	if len(addrs) > 0 {
		salt := []byte("test-salt")
		hashes := make([]common.Hash, len(addrs))
		for i, addr := range addrs {
			salted := make([]byte, len(salt)+common.AddressLength)
			copy(salted, salt)
			copy(salted[len(salt):], addr.Bytes())
			hashes[i] = sha256.Sum256(salted)
		}
		store.Store(salt, hashes, "test")
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
	filter := newHashedChecker([]common.Address{filteredAddr})
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
	filter := newHashedChecker([]common.Address{targetAddr})
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
	filter := newHashedChecker([]common.Address{targetAddr})
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
	filter := newHashedChecker([]common.Address{})
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
	filter := newHashedChecker([]common.Address{create2Addr})
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
	filter := newHashedChecker([]common.Address{createAddr})
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
	emptyChecker := newHashedChecker([]common.Address{})
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
	filter := newHashedChecker([]common.Address{filteredAddr})
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
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

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

func TestAddressFilterIndirectPayment(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.isSequencer = true
	cleanup := builder.Build(t)
	defer cleanup()

	// Deploy payer contract
	_, payer := deployAddressFilterTestContract(t, ctx, builder)

	// Deploy intermediary contract and fund it so it can forward payments
	intermediaryAddr, _ := deployAddressFilterTestContract(t, ctx, builder)

	// Create filtered destination
	builder.L2Info.GenerateAccount("FilteredDest")
	filteredAddr := builder.L2Info.GetAddress("FilteredDest")

	// Create clean destination for the positive test
	builder.L2Info.GenerateAccount("CleanDest")
	cleanAddr := builder.L2Info.GetAddress("CleanDest")

	// Set up filter to block FilteredDest
	filter := newHashedChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Test 1: Indirect payment payer -> intermediary -> filtered address should fail
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	auth.Value = big.NewInt(1e15)
	_, err := payer.PayVia(&auth, intermediaryAddr, filteredAddr)
	if err == nil {
		t.Fatal("expected indirect payment to filtered address to be rejected")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}

	// Test 2: Indirect payment payer -> intermediary -> clean address should succeed
	auth = builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	auth.Value = big.NewInt(1e15)
	tx, err := payer.PayVia(&auth, intermediaryAddr, cleanAddr)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// Verify the clean destination received the funds
	balance := builder.L2.GetBalance(t, cleanAddr)
	if balance.Cmp(big.NewInt(1e15)) != 0 {
		t.Fatalf("expected clean destination balance of 1e15, got %s", balance.String())
	}
}

// TestAddressFilterDelegateCall verifies that DELEGATECALL to a filtered address
// does NOT trigger filtering. DELEGATECALL loads code from the target but executes
// in the caller's context, so the target address is never "entered" from the
// filter's perspective (PushContract sees caller, not the code source).
func TestAddressFilterDelegateCall(t *testing.T) {
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
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// DELEGATECALL to filtered address should succeed because the target is
	// only a code source - execution stays in the caller's context.
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	tx, err := caller.DelegatecallTarget(&auth, targetAddr)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// Sanity check: a regular CALL to the same filtered address should still fail
	auth = builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	_, err = caller.CallTarget(&auth, targetAddr)
	if err == nil {
		t.Fatal("expected CALL to filtered address to be rejected")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}
}

// TestAddressFilterCallCode verifies that CALLCODE to a filtered address
// does NOT trigger filtering, for the same reason as DELEGATECALL: the target
// address is only used as a code source.
func TestAddressFilterCallCode(t *testing.T) {
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
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// CALLCODE to filtered address should succeed
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	tx, err := caller.CallcodeTarget(&auth, targetAddr)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
}

// TestAddressFilterCallViaFilteredIntermediary verifies that when a non-filtered
// contract CALLs a filtered intermediary, the transaction is rejected even though
// the final target is not filtered.
func TestAddressFilterCallViaFilteredIntermediary(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.isSequencer = true
	cleanup := builder.Build(t)
	defer cleanup()

	// Deploy payer contract (not filtered)
	_, payer := deployAddressFilterTestContract(t, ctx, builder)

	// Deploy intermediary contract (will be filtered)
	intermediaryAddr, _ := deployAddressFilterTestContract(t, ctx, builder)

	// Deploy final target contract (not filtered)
	targetAddr, _ := deployAddressFilterTestContract(t, ctx, builder)

	// Filter only the intermediary
	filter := newHashedChecker([]common.Address{intermediaryAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Test: payer -> filtered intermediary -> target should fail at the intermediary
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	_, err := payer.CallVia(&auth, intermediaryAddr, targetAddr)
	if err == nil {
		t.Fatal("expected call via filtered intermediary to be rejected")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}
}

// TestAddressFilterContractDeploy verifies that deploying a contract from an EOA
// is rejected when the resulting contract address is filtered.
func TestAddressFilterContractDeploy(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.isSequencer = true
	cleanup := builder.Build(t)
	defer cleanup()

	// Fund a deployer account
	builder.L2Info.GenerateAccount("Deployer")
	builder.L2.TransferBalance(t, "Owner", "Deployer", big.NewInt(1e18), builder.L2Info)

	// Compute the address that the next deployment from Deployer will create
	deployerAddr := builder.L2Info.GetAddress("Deployer")
	nonce, err := builder.L2.Client.NonceAt(ctx, deployerAddr, nil)
	Require(t, err)
	futureAddr := crypto.CreateAddress(deployerAddr, nonce)

	// Filter that future contract address
	filter := newHashedChecker([]common.Address{futureAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Deploy a contract (tx with no To address) - should be rejected
	auth := builder.L2Info.GetDefaultTransactOpts("Deployer", ctx)
	_, _, _, err = localgen.DeployAddressFilterTest(&auth, builder.L2.Client)
	if err == nil {
		t.Fatal("expected deployment to filtered address to be rejected")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}

	// Clear filter and verify deployment succeeds (nonce didn't increment, same address)
	emptyChecker := newHashedChecker([]common.Address{})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(emptyChecker)

	auth = builder.L2Info.GetDefaultTransactOpts("Deployer", ctx)
	_, tx, _, err := localgen.DeployAddressFilterTest(&auth, builder.L2.Client)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
}

func TestAddressFilterEventBypassRule(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	transferEvent := "Transfer(address,address,uint256)"
	selector, _, err := eventfilter.CanonicalSelectorFromEvent(transferEvent)
	Require(t, err)

	// Create rules with bypass: skip filtering when topic[1] (from) matches bypassAddr
	bypassAddr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	rules := []eventfilter.EventRule{
		{
			Event:          transferEvent,
			Selector:       selector,
			TopicAddresses: []int{1, 2},
			Bypass: &eventfilter.BypassRule{
				TopicIndex: 1,
				Equals:     bypassAddr,
			},
		},
	}

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false).WithEventFilterRules(rules)
	builder.isSequencer = true
	cleanup := builder.Build(t)
	defer cleanup()

	// Deploy test contract
	_, contract := deployAddressFilterTestContract(t, ctx, builder)

	// Create a filtered address
	builder.L2Info.GenerateAccount("FilteredUser")
	filteredAddr := builder.L2Info.GetAddress("FilteredUser")

	filter := newHashedChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Test 1: Transfer from random address to filtered address should be rejected
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	_, err = contract.EmitTransfer(&auth, auth.From, filteredAddr)
	if err == nil {
		t.Fatal("expected Transfer to filtered address to be rejected")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}

	// Test 2: Transfer FROM the bypass address TO the filtered address should succeed
	// because the bypass rule skips filtering when from == bypassAddr
	auth = builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	tx, err := contract.EmitTransfer(&auth, bypassAddr, filteredAddr)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// Test 3: Transfer FROM filtered address (not bypass) should still be rejected
	auth = builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	_, err = contract.EmitTransfer(&auth, filteredAddr, auth.From)
	if err == nil {
		t.Fatal("expected Transfer from filtered address to be rejected")
	}
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error, got: %v", err)
	}
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
