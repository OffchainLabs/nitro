package arbtest

import (
	"bytes"
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func getStorageRootHash(t *testing.T, node *arbnode.Node, address common.Address) common.Hash {
	t.Helper()
	statedb, err := node.Backend.ArbInterface().BlockChain().State()
	testhelpers.RequireImpl(t, err)
	trie := statedb.StorageTrie(address)
	return trie.Hash()
}

func getStorageSlotValue(t *testing.T, node *arbnode.Node, address common.Address) map[common.Hash]common.Hash {
	t.Helper()
	statedb, err := node.Backend.ArbInterface().BlockChain().State()
	testhelpers.RequireImpl(t, err)
	slotValue := make(map[common.Hash]common.Hash)
	testhelpers.RequireImpl(t, err)
	err = statedb.ForEachStorage(address, func(key, value common.Hash) bool {
		slotValue[key] = value
		return true
	})
	testhelpers.RequireImpl(t, err)
	return slotValue
}

func testConditionalTxThatShouldSucceed(t *testing.T, ctx context.Context, l2info info, rpcClient *rpc.Client, options *arbitrum_types.ConditionalOptions) {
	t.Helper()
	tx := l2info.PrepareTx("Owner", "User2", l2info.TransferGas, big.NewInt(1e12), nil)
	err := arbitrum.SendConditionalTransactionRPC(ctx, rpcClient, tx, options)
	if err != nil {
		testhelpers.FailImpl(t, "SendConditionalTransactionRPC failed: ", err)
	}
}

func testConditionalTxThatShouldFail(t *testing.T, ctx context.Context, l2info info, rpcClient *rpc.Client, options *arbitrum_types.ConditionalOptions) {
	t.Helper()
	accountInfo := l2info.GetInfoWithPrivKey("Owner")
	nonce := accountInfo.Nonce
	tx := l2info.PrepareTx("Owner", "User2", l2info.TransferGas, big.NewInt(1e12), nil)
	err := arbitrum.SendConditionalTransactionRPC(ctx, rpcClient, tx, options)
	if err == nil {
		testhelpers.FailImpl(t, "SendConditionalTransactionRPC didn't fail as expected")
	}
	accountInfo.Nonce = nonce // revert nonce as the tx failed
}

func getSuccessOptions(address1, address2 common.Address, currentRootHash1, currentRootHash2 common.Hash, currentSlotValueMap1, currentSlotValueMap2 map[common.Hash]common.Hash) []*arbitrum_types.ConditionalOptions {
	return []*arbitrum_types.ConditionalOptions{
		// empty options
		{},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{}},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{address1: {}}},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{address1: {SlotValue: map[common.Hash]common.Hash{}}}},

		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{address1: {RootHash: &currentRootHash1}}},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{address1: {RootHash: &currentRootHash1}, address2: {RootHash: &currentRootHash2}}},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{address1: {SlotValue: currentSlotValueMap1}}},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{address1: {SlotValue: currentSlotValueMap1}, address2: {SlotValue: currentSlotValueMap2}}},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{address1: {RootHash: &currentRootHash1}, address2: {SlotValue: currentSlotValueMap2}}},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{address1: {SlotValue: currentSlotValueMap1}, address2: {RootHash: &currentRootHash2}}},
	}
}

func TestSendRawTransactionConditional(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2info, node, client := CreateTestL2(t, ctx)
	defer node.StopAndWait()

	auth := l2info.GetDefaultTransactOpts("Owner", ctx)
	contractAddress1, simple1 := deploySimple(t, ctx, auth, client)
	tx, err := simple1.Increment(&auth)
	Require(t, err, "failed to call Increment()")
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)
	contractAddress2, simple2 := deploySimple(t, ctx, auth, client)
	tx, err = simple2.Increment(&auth)
	Require(t, err, "failed to call Increment()")
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)
	tx, err = simple2.Increment(&auth)
	Require(t, err, "failed to call Increment()")
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	currentRootHash1 := getStorageRootHash(t, node, contractAddress1)
	currentSlotValueMap1 := getStorageSlotValue(t, node, contractAddress1)
	currentRootHash2 := getStorageRootHash(t, node, contractAddress2)
	currentSlotValueMap2 := getStorageSlotValue(t, node, contractAddress2)

	rpcClient, err := node.Stack.Attach()
	testhelpers.RequireImpl(t, err)

	l2info.GenerateAccount("User2")

	testConditionalTxThatShouldSucceed(t, ctx, l2info, rpcClient, nil)

	successOptions := getSuccessOptions(contractAddress1, contractAddress2, currentRootHash1, currentRootHash2, currentSlotValueMap1, currentSlotValueMap2)
	for _, options := range successOptions {
		testConditionalTxThatShouldSucceed(t, ctx, l2info, rpcClient, options)
	}

	tx, err = simple1.Increment(&auth)
	Require(t, err, "failed to call Increment()")
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)
	tx, err = simple2.Increment(&auth)
	Require(t, err, "failed to call Increment()")
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	previousStorageRootHash1 := currentRootHash1
	currentRootHash1 = getStorageRootHash(t, node, contractAddress1)
	if bytes.Equal(previousStorageRootHash1.Bytes(), currentRootHash1.Bytes()) {
		testhelpers.FailImpl(t, "storage root hash didn't change as expected")
	}
	previousSlotValueMap1 := currentSlotValueMap1
	currentSlotValueMap1 = getStorageSlotValue(t, node, contractAddress1)

	previousStorageRootHash2 := currentRootHash2
	currentRootHash2 = getStorageRootHash(t, node, contractAddress2)
	if bytes.Equal(previousStorageRootHash2.Bytes(), currentRootHash2.Bytes()) {
		testhelpers.FailImpl(t, "storage root hash didn't change as expected")
	}
	previousSlotValueMap2 := currentSlotValueMap2
	currentSlotValueMap2 = getStorageSlotValue(t, node, contractAddress2)

	successOptions = getSuccessOptions(contractAddress1, contractAddress2, currentRootHash1, currentRootHash2, currentSlotValueMap1, currentSlotValueMap2)
	for _, options := range successOptions {
		testConditionalTxThatShouldSucceed(t, ctx, l2info, rpcClient, options)
	}
	failOptions := []*arbitrum_types.ConditionalOptions{
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{contractAddress1: {RootHash: &previousStorageRootHash1}}},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{contractAddress1: {SlotValue: previousSlotValueMap1}}},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{contractAddress1: {RootHash: &previousStorageRootHash1}, contractAddress2: {RootHash: &previousStorageRootHash2}}},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{contractAddress1: {RootHash: &currentRootHash1}, contractAddress2: {RootHash: &previousStorageRootHash2}}},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{contractAddress1: {SlotValue: previousSlotValueMap1}, contractAddress2: {SlotValue: previousSlotValueMap2}}},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{contractAddress1: {SlotValue: currentSlotValueMap1}, contractAddress2: {SlotValue: previousSlotValueMap2}}},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{contractAddress1: {SlotValue: map[common.Hash]common.Hash{}}, contractAddress2: {SlotValue: previousSlotValueMap2}}},
	}
	for _, options := range failOptions {
		testConditionalTxThatShouldFail(t, ctx, l2info, rpcClient, options)
	}
}
