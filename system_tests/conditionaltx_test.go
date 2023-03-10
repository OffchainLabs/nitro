package arbtest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
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

func testConditionalTxThatShouldSucceed(t *testing.T, ctx context.Context, idx int, l2info info, rpcClient *rpc.Client, options *arbitrum_types.ConditionalOptions) {
	t.Helper()
	tx := l2info.PrepareTx("Owner", "User2", l2info.TransferGas, big.NewInt(1e12), nil)
	err := arbitrum.SendConditionalTransactionRPC(ctx, rpcClient, tx, options)
	if err != nil {
		testhelpers.FailImpl(t, "SendConditionalTransactionRPC failed, idx:", idx, "err:", err)
	}
}

func testConditionalTxThatShouldFail(t *testing.T, ctx context.Context, idx int, l2info info, rpcClient *rpc.Client, options *arbitrum_types.ConditionalOptions, expectedErrorCode int) {
	t.Helper()
	accountInfo := l2info.GetInfoWithPrivKey("Owner")
	nonce := accountInfo.Nonce
	tx := l2info.PrepareTx("Owner", "User2", l2info.TransferGas, big.NewInt(1e12), nil)
	err := arbitrum.SendConditionalTransactionRPC(ctx, rpcClient, tx, options)
	if err == nil {
		testhelpers.FailImpl(t, "SendConditionalTransactionRPC didn't fail as expected, idx:", idx)
	} else {
		var rErr rpc.Error
		if errors.As(err, &rErr) {
			if rErr.ErrorCode() != expectedErrorCode {
				testhelpers.FailImpl(t, "unexpected error code, have:", rErr.ErrorCode(), "want:", expectedErrorCode)
			}
		} else {
			testhelpers.FailImpl(t, "unexpected error type, err:", err)
		}
	}
	accountInfo.Nonce = nonce // revert nonce as the tx failed
}

func getSuccessOptions(address1, address2 common.Address, currentRootHash1, currentRootHash2 common.Hash, currentSlotValueMap1, currentSlotValueMap2 map[common.Hash]common.Hash, blockNumber uint64, timestamp uint64) []*arbitrum_types.ConditionalOptions {
	future := hexutil.Uint64(timestamp + 5)
	past := hexutil.Uint64(timestamp - 1)
	futureBlockNumber := hexutil.Uint64(blockNumber + 1000)
	currentBlockNumber := hexutil.Uint64(blockNumber)
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
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{address1: {RootHash: &currentRootHash1}, address2: {SlotValue: currentSlotValueMap2}}, TimestampMax: &future},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{address1: {RootHash: &currentRootHash1}, address2: {SlotValue: currentSlotValueMap2}}, TimestampMin: &past},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{address1: {RootHash: &currentRootHash1}, address2: {SlotValue: currentSlotValueMap2}}, TimestampMax: &future, TimestampMin: &past},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{address1: {RootHash: &currentRootHash1}, address2: {SlotValue: currentSlotValueMap2}}, BlockNumberMax: &futureBlockNumber},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{address1: {RootHash: &currentRootHash1}, address2: {SlotValue: currentSlotValueMap2}}, BlockNumberMin: &currentBlockNumber},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{address1: {RootHash: &currentRootHash1}, address2: {SlotValue: currentSlotValueMap2}}, BlockNumberMax: &futureBlockNumber, BlockNumberMin: &currentBlockNumber},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{address1: {RootHash: &currentRootHash1}, address2: {SlotValue: currentSlotValueMap2}}, BlockNumberMax: &futureBlockNumber, BlockNumberMin: &currentBlockNumber, TimestampMax: &future, TimestampMin: &past},
	}
}

func TestSendRawTransactionConditionalBasic(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l2info, node, l2client, _, _, l1client, l1stack := createTestNodeOnL1(t, ctx, true)
	defer requireClose(t, l1stack)
	defer node.StopAndWait()

	auth := l2info.GetDefaultTransactOpts("Owner", ctx)
	contractAddress1, simple1 := deploySimple(t, ctx, auth, l2client)
	tx, err := simple1.Increment(&auth)
	Require(t, err, "failed to call Increment()")
	_, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)
	contractAddress2, simple2 := deploySimple(t, ctx, auth, l2client)
	tx, err = simple2.Increment(&auth)
	Require(t, err, "failed to call Increment()")
	_, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)
	tx, err = simple2.Increment(&auth)
	Require(t, err, "failed to call Increment()")
	_, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	currentRootHash1 := getStorageRootHash(t, node, contractAddress1)
	currentSlotValueMap1 := getStorageSlotValue(t, node, contractAddress1)
	currentRootHash2 := getStorageRootHash(t, node, contractAddress2)
	currentSlotValueMap2 := getStorageSlotValue(t, node, contractAddress2)

	rpcClient, err := node.Stack.Attach()
	testhelpers.RequireImpl(t, err)

	l2info.GenerateAccount("User2")

	testConditionalTxThatShouldSucceed(t, ctx, -1, l2info, rpcClient, nil)

	block, err := l1client.BlockByNumber(ctx, nil)
	testhelpers.RequireImpl(t, err)
	blockNumber := block.NumberU64()
	blockTime := block.Time()
	successOptions := getSuccessOptions(contractAddress1, contractAddress2, currentRootHash1, currentRootHash2, currentSlotValueMap1, currentSlotValueMap2, blockNumber, blockTime)
	for i, options := range successOptions {
		testConditionalTxThatShouldSucceed(t, ctx, i, l2info, rpcClient, options)
	}

	tx, err = simple1.Increment(&auth)
	Require(t, err, "failed to call Increment()")
	_, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)
	tx, err = simple2.Increment(&auth)
	Require(t, err, "failed to call Increment()")
	_, err = EnsureTxSucceeded(ctx, l2client, tx)
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

	block, err = l1client.BlockByNumber(ctx, nil)
	testhelpers.RequireImpl(t, err)
	blockNumber = block.NumberU64()
	blockTime = block.Time()
	successOptions = getSuccessOptions(contractAddress1, contractAddress2, currentRootHash1, currentRootHash2, currentSlotValueMap1, currentSlotValueMap2, blockNumber, blockTime)
	for i, options := range successOptions {
		testConditionalTxThatShouldSucceed(t, ctx, i, l2info, rpcClient, options)
	}
	block, err = l1client.BlockByNumber(ctx, nil)
	testhelpers.RequireImpl(t, err)
	blockNumber = block.NumberU64()
	blockTime = block.Time()
	future := hexutil.Uint64(blockTime + 30)
	past := hexutil.Uint64(blockTime - 1)
	futureBlockNumber := hexutil.Uint64(blockNumber + 1000)
	currentBlockNumber := hexutil.Uint64(blockNumber)
	if blockNumber == 0 {
		testhelpers.FailImpl(t, "internal test error: unexpected blockNumber == 0")
	}
	previousBlockNumber := hexutil.Uint64(blockNumber - 1)
	failOptions := []*arbitrum_types.ConditionalOptions{
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{contractAddress1: {RootHash: &previousStorageRootHash1}}},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{contractAddress1: {SlotValue: previousSlotValueMap1}}},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{contractAddress1: {RootHash: &previousStorageRootHash1}, contractAddress2: {RootHash: &previousStorageRootHash2}}},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{contractAddress1: {RootHash: &currentRootHash1}, contractAddress2: {RootHash: &previousStorageRootHash2}}},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{contractAddress1: {SlotValue: previousSlotValueMap1}, contractAddress2: {SlotValue: previousSlotValueMap2}}},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{contractAddress1: {SlotValue: currentSlotValueMap1}, contractAddress2: {SlotValue: previousSlotValueMap2}}},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{contractAddress1: {SlotValue: map[common.Hash]common.Hash{}}, contractAddress2: {SlotValue: previousSlotValueMap2}}},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{contractAddress1: {RootHash: &previousStorageRootHash1}}, BlockNumberMax: &futureBlockNumber, BlockNumberMin: &currentBlockNumber, TimestampMax: &future, TimestampMin: &past},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{contractAddress1: {RootHash: &currentRootHash1}, contractAddress2: {SlotValue: currentSlotValueMap2}}, TimestampMax: &past},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{contractAddress1: {RootHash: &currentRootHash1}, contractAddress2: {SlotValue: currentSlotValueMap2}}, TimestampMin: &future},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{contractAddress1: {RootHash: &currentRootHash1}, contractAddress2: {SlotValue: currentSlotValueMap2}}, TimestampMax: &future, TimestampMin: &future},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{contractAddress1: {RootHash: &currentRootHash1}, contractAddress2: {SlotValue: currentSlotValueMap2}}, TimestampMax: &past, TimestampMin: &past},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{contractAddress1: {RootHash: &currentRootHash1}, contractAddress2: {SlotValue: currentSlotValueMap2}}, BlockNumberMax: &previousBlockNumber},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{contractAddress1: {RootHash: &currentRootHash1}, contractAddress2: {SlotValue: currentSlotValueMap2}}, BlockNumberMin: &futureBlockNumber},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{contractAddress1: {RootHash: &currentRootHash1}, contractAddress2: {SlotValue: currentSlotValueMap2}}, BlockNumberMax: &futureBlockNumber, BlockNumberMin: &futureBlockNumber},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{contractAddress1: {RootHash: &currentRootHash1}, contractAddress2: {SlotValue: currentSlotValueMap2}}, BlockNumberMax: &previousBlockNumber, BlockNumberMin: &previousBlockNumber},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{contractAddress1: {RootHash: &currentRootHash1}, contractAddress2: {SlotValue: currentSlotValueMap2}}, BlockNumberMax: &previousBlockNumber, BlockNumberMin: &previousBlockNumber, TimestampMax: &future, TimestampMin: &past},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{contractAddress1: {RootHash: &currentRootHash1}, contractAddress2: {SlotValue: currentSlotValueMap2}}, BlockNumberMax: &futureBlockNumber, BlockNumberMin: &futureBlockNumber, TimestampMax: &future, TimestampMin: &past},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{contractAddress1: {RootHash: &currentRootHash1}, contractAddress2: {SlotValue: currentSlotValueMap2}}, BlockNumberMax: &futureBlockNumber, BlockNumberMin: &previousBlockNumber, TimestampMax: &past, TimestampMin: &past},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{contractAddress1: {RootHash: &currentRootHash1}, contractAddress2: {SlotValue: currentSlotValueMap2}}, BlockNumberMax: &futureBlockNumber, BlockNumberMin: &previousBlockNumber, TimestampMax: &future, TimestampMin: &future},
	}
	for i, options := range failOptions {
		testConditionalTxThatShouldFail(t, ctx, i, l2info, rpcClient, options, -32003)
	}
}

func TestSendRawTransactionConditionalMultiRoutine(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2info, node, client := CreateTestL2(t, ctx)
	defer node.StopAndWait()
	rpcClient, err := node.Stack.Attach()
	testhelpers.RequireImpl(t, err)

	auth := l2info.GetDefaultTransactOpts("Owner", ctx)
	contractAddress, simple := deploySimple(t, ctx, auth, client)

	simpleContract, err := abi.JSON(strings.NewReader(mocksgen.SimpleABI))
	testhelpers.RequireImpl(t, err)

	numTxes := 200
	expectedSuccesses := numTxes / 20
	var txes types.Transactions
	var options []*arbitrum_types.ConditionalOptions
	for i := 0; i < numTxes; i++ {
		account := fmt.Sprintf("User%v", i)
		l2info.GenerateAccount(account)
		tx := l2info.PrepareTx("Owner", account, l2info.TransferGas, big.NewInt(1e16), nil)
		err := client.SendTransaction(ctx, tx)
		testhelpers.RequireImpl(t, err)
		_, err = EnsureTxSucceeded(ctx, client, tx)
		Require(t, err)
	}
	for i := numTxes - 1; i >= 0; i-- {
		expected := i % expectedSuccesses
		data, err := simpleContract.Pack("logAndIncrement", big.NewInt(int64(expected)))
		testhelpers.RequireImpl(t, err)
		account := fmt.Sprintf("User%v", i)
		txes = append(txes, l2info.PrepareTxTo(account, &contractAddress, l2info.TransferGas, big.NewInt(0), data))
		options = append(options, &arbitrum_types.ConditionalOptions{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{contractAddress: {SlotValue: map[common.Hash]common.Hash{{0}: common.BigToHash(big.NewInt(int64(expected)))}}}})
	}
	ctxWithTimeout, cancelCtxWithTimeout := context.WithTimeout(ctx, 5*time.Second)
	success := make(chan struct{}, len(txes))
	wg := sync.WaitGroup{}
	for i := 0; i < len(txes); i++ {
		wg.Add(1)
		tx := txes[i]
		opts := options[i]
		go func() {
			defer wg.Done()
			for ctxWithTimeout.Err() == nil {
				err := arbitrum.SendConditionalTransactionRPC(ctxWithTimeout, rpcClient, tx, opts)
				if err == nil {
					success <- struct{}{}
					break
				}
			}
		}()
	}
	for i := 0; i < expectedSuccesses; i++ {
		select {
		case <-success:
		case <-ctxWithTimeout.Done():
			testhelpers.FailImpl(t, "test timeouted")
		}
	}
	cancelCtxWithTimeout()
	wg.Wait()
	bc := node.Backend.ArbInterface().BlockChain()
	genesis := bc.Config().ArbitrumChainParams.GenesisBlockNum

	var receipts types.Receipts
	header := bc.GetHeaderByNumber(genesis)
	for i := genesis + 1; header != nil; i++ {
		blockReceipts := bc.GetReceiptsByHash(header.Hash())
		if blockReceipts == nil {
			testhelpers.FailImpl(t, "Failed to get block receipts, block number:", header.Number)
		}
		receipts = append(receipts, blockReceipts...)
		header = bc.GetHeaderByNumber(i)
	}

	succeeded := 0
	for _, receipt := range receipts {
		if receipt.Status == types.ReceiptStatusSuccessful && len(receipt.Logs) == 1 {
			parsed, err := simple.ParseLogAndIncrementCalled(*receipt.Logs[0])
			Require(t, err)
			if parsed.Expected.Int64() != parsed.Have.Int64() {
				testhelpers.FailImpl(t, "Got invalid log, log.Expected:", parsed.Expected, "log.Have:", parsed.Have)
			} else {
				succeeded++
			}
		}
	}
	if succeeded != expectedSuccesses {
		testhelpers.FailImpl(t, "Unexpected number of successful txes, want:", numTxes, "have:", succeeded)
	}
}

func TestSendRawTransactionConditionalPreCheck(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodeConfig := arbnode.ConfigDefaultL1Test()
	nodeConfig.TxPreCheckerStrictness = arbnode.TxPreCheckerStrictnessLikelyCompatible
	l2info, node, l2client, _, _, _, l1stack := createTestNodeOnL1WithConfig(t, ctx, true, nodeConfig, nil, nil)
	defer requireClose(t, l1stack)
	defer node.StopAndWait()
	rpcClient, err := node.Stack.Attach()
	testhelpers.RequireImpl(t, err)

	l2info.GenerateAccount("User2")

	auth := l2info.GetDefaultTransactOpts("Owner", ctx)
	start := time.Now().Unix()
	contractAddress, simple := deploySimple(t, ctx, auth, l2client)
	if time.Since(time.Unix(start, 0)) > 200*time.Millisecond {
		start++
		time.Sleep(time.Until(time.Unix(start, 0)))
	}
	tx, err := simple.Increment(&auth)
	Require(t, err, "failed to call Increment()")
	_, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)
	currentRootHash := getStorageRootHash(t, node, contractAddress)
	options := &arbitrum_types.ConditionalOptions{
		KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{
			contractAddress: {RootHash: &currentRootHash},
		},
	}
	testConditionalTxThatShouldFail(t, ctx, 0, l2info, rpcClient, options, -32003)
	time.Sleep(time.Until(time.Unix(start+1, 0)))
	testConditionalTxThatShouldSucceed(t, ctx, 1, l2info, rpcClient, options)
}
