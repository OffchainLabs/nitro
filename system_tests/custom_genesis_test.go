// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"bytes"
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

// TestCustomGenesisFile boots a node from a custom genesis JSON file and verifies
// that wallet balances, contract code, and contract storage declared in the genesis
// file actually materialize on-chain.
func TestCustomGenesisFile(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false).
		WithGenesisFile(t, "../cmd/nitro/init/testdata/custom_genesis.json")
	cleanup := builder.Build(t)
	defer cleanup()

	client := builder.L2.Client

	// 1. Verify wallet balance (0xAAAA...0001 should have 1 ETH).
	walletAddr := common.HexToAddress("0xAAAA000000000000000000000000000000000001")
	balance, err := client.BalanceAt(ctx, walletAddr, nil)
	Require(t, err)
	expectedBalance, _ := new(big.Int).SetString("DE0B6B3A7640000", 16) // 1 ETH in wei
	if balance.Cmp(expectedBalance) != 0 {
		t.Fatalf("wallet balance mismatch: got %s, want %s", balance, expectedBalance)
	}

	// 2. Verify contract code (0xBBBB...0001 should have bytecode that returns 0x42).
	contractAddr := common.HexToAddress("0xBBBB000000000000000000000000000000000001")
	code, err := client.CodeAt(ctx, contractAddr, nil)
	Require(t, err)
	expectedCode := common.FromHex("604260005260206000f3")
	if !bytes.Equal(code, expectedCode) {
		t.Fatalf("contract code mismatch: got %x, want %x", code, expectedCode)
	}

	// 3. Verify contract execution returns 0x42 via eth_call.
	result, err := client.CallContract(ctx, ethereum.CallMsg{To: &contractAddr}, nil)
	Require(t, err)
	resultVal := new(big.Int).SetBytes(result)
	if resultVal.Cmp(big.NewInt(0x42)) != 0 {
		t.Fatalf("contract call returned %s, want 0x42", resultVal)
	}

	// 4. Verify contract with storage (0xCCCC...0001).
	storageAddr := common.HexToAddress("0xCCCC000000000000000000000000000000000001")

	// 4a. Storage slot 0x01 should contain 0xdeadbeef.
	slot1, err := client.StorageAt(ctx, storageAddr, common.HexToHash("0x01"), nil)
	Require(t, err)
	expectedSlot1 := common.HexToHash("0x00000000000000000000000000000000000000000000000000000000deadbeef")
	if common.BytesToHash(slot1) != expectedSlot1 {
		t.Fatalf("storage slot 1 mismatch: got %x, want %x", slot1, expectedSlot1.Bytes())
	}

	// 4b. Storage slot 0x02 should contain 0xcafebabe.
	slot2, err := client.StorageAt(ctx, storageAddr, common.HexToHash("0x02"), nil)
	Require(t, err)
	expectedSlot2 := common.HexToHash("0x00000000000000000000000000000000000000000000000000000000cafebabe")
	if common.BytesToHash(slot2) != expectedSlot2 {
		t.Fatalf("storage slot 2 mismatch: got %x, want %x", slot2, expectedSlot2.Bytes())
	}

	// 4c. The storage contract's bytecode loads slot 1 and returns it.
	storageResult, err := client.CallContract(ctx, ethereum.CallMsg{To: &storageAddr}, nil)
	Require(t, err)
	if common.BytesToHash(storageResult) != expectedSlot1 {
		t.Fatalf("storage contract call returned %x, want %x", storageResult, expectedSlot1.Bytes())
	}

	// 4d. Verify code exists on the storage contract.
	storageCode, err := client.CodeAt(ctx, storageAddr, nil)
	Require(t, err)
	if len(storageCode) == 0 {
		t.Fatal("storage contract has no code")
	}

	// 5. Verify large balance wallet (0xDDDD...0001).
	largeAddr := common.HexToAddress("0xDDDD000000000000000000000000000000000001")
	largeBal, err := client.BalanceAt(ctx, largeAddr, nil)
	Require(t, err)
	expectedLarge, _ := new(big.Int).SetString("C097CE7BC90715B34B9F1000000000", 16)
	if largeBal.Cmp(expectedLarge) != 0 {
		t.Fatalf("large wallet balance mismatch: got %s, want %s", largeBal, expectedLarge)
	}

	// 6. Verify the chain is functional by sending a transaction.
	builder.L2Info.GenerateAccount("User")
	builder.L2.TransferBalance(t, "Owner", "User", big.NewInt(1e12), builder.L2Info)
	userBal, err := client.BalanceAt(ctx, builder.L2Info.GetAddress("User"), nil)
	Require(t, err)
	if userBal.Cmp(big.NewInt(1e12)) < 0 {
		t.Fatalf("transfer failed: User balance %s < expected", userBal)
	}
}

// TestCustomGenesisWithChainOwner boots a node from a genesis with a custom chain ID
// and InitialChainOwner, then verifies the chain ID is correct, the designated owner
// appears in the chain owners list, and genesis alloc accounts are present.
func TestCustomGenesisWithChainOwner(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false).
		WithGenesisFile(t, "../cmd/nitro/init/testdata/custom_genesis_chain_owner.json")
	cleanup := builder.Build(t)
	defer cleanup()

	client := builder.L2.Client

	// 1. Verify custom chain ID (999999).
	chainID, err := client.ChainID(ctx)
	Require(t, err)
	if chainID.Int64() != 999999 {
		t.Fatalf("chain ID mismatch: got %d, want 999999", chainID.Int64())
	}

	// 2. Verify InitialChainOwner is in the owners list.
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, client)
	Require(t, err)
	callOpts := builder.L2Info.GetDefaultCallOpts("Owner", ctx)
	owners, err := arbOwner.GetAllChainOwners(callOpts)
	Require(t, err)

	genesisOwner := common.HexToAddress("0x1111111111111111111111111111111111111111")
	found := false
	for _, owner := range owners {
		if owner == genesisOwner {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("InitialChainOwner %s not found in chain owners: %v", genesisOwner, owners)
	}

	// 3. Verify genesis alloc wallet is present.
	walletAddr := common.HexToAddress("0xAAAA000000000000000000000000000000000001")
	balance, err := client.BalanceAt(ctx, walletAddr, nil)
	Require(t, err)
	expectedBalance, _ := new(big.Int).SetString("DE0B6B3A7640000", 16)
	if balance.Cmp(expectedBalance) != 0 {
		t.Fatalf("wallet balance mismatch: got %s, want %s", balance, expectedBalance)
	}

	// 4. Verify genesis alloc contract is present and executable.
	contractAddr := common.HexToAddress("0xBBBB000000000000000000000000000000000001")
	code, err := client.CodeAt(ctx, contractAddr, nil)
	Require(t, err)
	if len(code) == 0 {
		t.Fatal("genesis contract has no code")
	}
	result, err := client.CallContract(ctx, ethereum.CallMsg{To: &contractAddr}, nil)
	Require(t, err)
	if new(big.Int).SetBytes(result).Cmp(big.NewInt(0x42)) != 0 {
		t.Fatalf("contract call returned %x, want 0x42", result)
	}

	// 5. Verify chain is functional with a transfer.
	builder.L2Info.GenerateAccount("User")
	builder.L2.TransferBalance(t, "Owner", "User", big.NewInt(1e12), builder.L2Info)
	userBal, err := client.BalanceAt(ctx, builder.L2Info.GetAddress("User"), nil)
	Require(t, err)
	if userBal.Cmp(big.NewInt(1e12)) < 0 {
		t.Fatalf("transfer failed: User balance %s < expected", userBal)
	}
}

// TestCustomGenesisWithNativeToken boots a node from a genesis that enables native
// token supply management via arbOSInit, then verifies the feature is active and
// genesis alloc accounts are present including a contract with storage.
func TestCustomGenesisWithNativeToken(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false).
		WithGenesisFile(t, "../cmd/nitro/init/testdata/custom_genesis_native_token.json")
	cleanup := builder.Build(t)
	defer cleanup()

	client := builder.L2.Client

	// 1. Verify native token management is active by adding a native token owner.
	//    This call would revert if the feature wasn't enabled at genesis.
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, client)
	Require(t, err)
	ownerAuth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	nativeTokenOwner := common.HexToAddress("0xEEEE000000000000000000000000000000000001")
	tx, err := arbOwner.AddNativeTokenOwner(&ownerAuth, nativeTokenOwner)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// Verify the owner was added.
	callOpts := builder.L2Info.GetDefaultCallOpts("Owner", ctx)
	isOwner, err := arbOwner.IsNativeTokenOwner(callOpts, nativeTokenOwner)
	Require(t, err)
	if !isOwner {
		t.Fatal("AddNativeTokenOwner succeeded but IsNativeTokenOwner returned false")
	}

	// 2. Verify genesis alloc wallet is present.
	walletAddr := common.HexToAddress("0xAAAA000000000000000000000000000000000001")
	balance, err := client.BalanceAt(ctx, walletAddr, nil)
	Require(t, err)
	expectedBalance, _ := new(big.Int).SetString("DE0B6B3A7640000", 16)
	if balance.Cmp(expectedBalance) != 0 {
		t.Fatalf("wallet balance mismatch: got %s, want %s", balance, expectedBalance)
	}

	// 3. Verify genesis alloc contract with storage is present.
	storageAddr := common.HexToAddress("0xCCCC000000000000000000000000000000000001")
	slot1, err := client.StorageAt(ctx, storageAddr, common.HexToHash("0x01"), nil)
	Require(t, err)
	expectedSlot1 := common.HexToHash("0x00000000000000000000000000000000000000000000000000000000deadbeef")
	if common.BytesToHash(slot1) != expectedSlot1 {
		t.Fatalf("storage slot 1 mismatch: got %x, want %x", slot1, expectedSlot1.Bytes())
	}

	// 4. Verify the storage contract executes correctly.
	storageResult, err := client.CallContract(ctx, ethereum.CallMsg{To: &storageAddr}, nil)
	Require(t, err)
	if common.BytesToHash(storageResult) != expectedSlot1 {
		t.Fatalf("storage contract call returned %x, want %x", storageResult, expectedSlot1.Bytes())
	}
}
