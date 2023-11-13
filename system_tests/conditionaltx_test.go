package arbtest

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
)

func getStorageRootHash(t *testing.T, execNode *gethexec.ExecutionNode, address common.Address) common.Hash {
	t.Helper()
	statedb, err := execNode.Backend.ArbInterface().BlockChain().State()
	Require(t, err)
	trie, err := statedb.StorageTrie(address)
	Require(t, err)
	return trie.Hash()
}

func getStorageSlotValue(t *testing.T, execNode *gethexec.ExecutionNode, address common.Address) map[common.Hash]common.Hash {
	t.Helper()
	statedb, err := execNode.Backend.ArbInterface().BlockChain().State()
	Require(t, err)
	slotValue := make(map[common.Hash]common.Hash)
	Require(t, err)
	err = statedb.ForEachStorage(address, func(key, value common.Hash) bool {
		slotValue[key] = value
		return true
	})
	Require(t, err)
	return slotValue
}

func testConditionalTxThatShouldSucceed(t *testing.T, ctx context.Context, idx int, l2info info, rpcClient *rpc.Client, options *arbitrum_types.ConditionalOptions) {
	t.Helper()
	tx := l2info.PrepareTx("Owner", "User2", l2info.TransferGas, big.NewInt(1e12), nil)
	err := arbitrum.SendConditionalTransactionRPC(ctx, rpcClient, tx, options)
	if err != nil {
		Fatal(t, "SendConditionalTransactionRPC failed, idx:", idx, "err:", err)
	}
}

func testConditionalTxThatShouldFail(t *testing.T, ctx context.Context, idx int, l2info info, rpcClient *rpc.Client, options *arbitrum_types.ConditionalOptions, expectedErrorCode int) {
	t.Helper()
	accountInfo := l2info.GetInfoWithPrivKey("Owner")
	nonce := accountInfo.Nonce
	tx := l2info.PrepareTx("Owner", "User2", l2info.TransferGas, big.NewInt(1e12), nil)
	err := arbitrum.SendConditionalTransactionRPC(ctx, rpcClient, tx, options)
	if err == nil {
		if options == nil {
			Fatal(t, "SendConditionalTransactionRPC didn't fail as expected, idx:", idx, "options:", options)
		} else {
			Fatal(t, "SendConditionalTransactionRPC didn't fail as expected, idx:", idx, "options:", *options)
		}
	} else {
		var rErr rpc.Error
		if errors.As(err, &rErr) {
			if rErr.ErrorCode() != expectedErrorCode {
				Fatal(t, "unexpected error code, have:", rErr.ErrorCode(), "want:", expectedErrorCode)
			}
		} else {
			Fatal(t, "unexpected error type, err:", err)
		}
	}
	accountInfo.Nonce = nonce // revert nonce as the tx failed
}

func getEmptyOptions(address common.Address) []*arbitrum_types.ConditionalOptions {
	return []*arbitrum_types.ConditionalOptions{
		{},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{}},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{address: {}}},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{address: {SlotValue: map[common.Hash]common.Hash{}}}},
	}
}

func getOptions(address common.Address, rootHash common.Hash, slotValueMap map[common.Hash]common.Hash) []*arbitrum_types.ConditionalOptions {
	return []*arbitrum_types.ConditionalOptions{
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{address: {RootHash: &rootHash}}},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{address: {RootHash: &rootHash}}},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{address: {SlotValue: slotValueMap}}},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{address: {SlotValue: slotValueMap}}},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{address: {RootHash: &rootHash}}},
		{KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{address: {SlotValue: slotValueMap}}},
	}
}

func getFulfillableBlockTimeLimits(t *testing.T, blockNumber uint64, timestamp uint64) []*arbitrum_types.ConditionalOptions {
	future := math.HexOrDecimal64(timestamp + 30)
	past := math.HexOrDecimal64(timestamp - 1)
	futureBlockNumber := math.HexOrDecimal64(blockNumber + 1000)
	currentBlockNumber := math.HexOrDecimal64(blockNumber)
	return getBlockTimeLimits(t, currentBlockNumber, futureBlockNumber, past, future)
}

func getUnfulfillableBlockTimeLimits(t *testing.T, blockNumber uint64, timestamp uint64) []*arbitrum_types.ConditionalOptions {
	future := math.HexOrDecimal64(timestamp + 30)
	past := math.HexOrDecimal64(timestamp - 1)
	futureBlockNumber := math.HexOrDecimal64(blockNumber + 1000)
	previousBlockNumber := math.HexOrDecimal64(blockNumber - 1)
	// skip first empty options
	return getBlockTimeLimits(t, futureBlockNumber, previousBlockNumber, future, past)[1:]
}

func getBlockTimeLimits(t *testing.T, blockMin, blockMax math.HexOrDecimal64, timeMin, timeMax math.HexOrDecimal64) []*arbitrum_types.ConditionalOptions {
	basic := []*arbitrum_types.ConditionalOptions{
		{},
		{TimestampMin: &timeMin},
		{TimestampMax: &timeMax},
		{BlockNumberMin: &blockMin},
		{BlockNumberMax: &blockMax},
	}
	power := []*arbitrum_types.ConditionalOptions{
		{},
	}
	for range basic {
		power = optionsProduct(power, basic)
	}
	return dedupOptions(t, power)
}

func optionsDedupProduct(t *testing.T, optionsA, optionsB []*arbitrum_types.ConditionalOptions) []*arbitrum_types.ConditionalOptions {
	return dedupOptions(t, optionsProduct(optionsA, optionsB))
}

// Product of options slices, where each element from optionsA is merged with element of optionsB
// The merge involves:
// * merging KnownAccounts maps, where in case of key collision the value is taken from optionsB element
// * assigning new block and timestamp limits preferring values from optionsB element
func optionsProduct(optionsA, optionsB []*arbitrum_types.ConditionalOptions) []*arbitrum_types.ConditionalOptions {
	var optionsC []*arbitrum_types.ConditionalOptions
	for _, a := range optionsA {
		for _, b := range optionsB {
			var c arbitrum_types.ConditionalOptions
			c.KnownAccounts = make(map[common.Address]arbitrum_types.RootHashOrSlots)
			for k, v := range a.KnownAccounts {
				c.KnownAccounts[k] = v
			}
			for k, v := range b.KnownAccounts {
				c.KnownAccounts[k] = v
			}
			limitTriples := []struct {
				a *math.HexOrDecimal64
				b *math.HexOrDecimal64
				c **math.HexOrDecimal64
			}{
				{a.BlockNumberMin, b.BlockNumberMin, &c.BlockNumberMin},
				{a.BlockNumberMax, b.BlockNumberMax, &c.BlockNumberMax},
				{a.TimestampMin, b.TimestampMin, &c.TimestampMin},
				{a.TimestampMax, b.TimestampMax, &c.TimestampMax},
			}
			for _, tripple := range limitTriples {
				if tripple.b != nil {
					value := math.HexOrDecimal64(*tripple.b)
					*tripple.c = &value
				} else if tripple.a != nil {
					value := math.HexOrDecimal64(*tripple.a)
					*tripple.c = &value
				} else {
					*tripple.c = nil
				}
			}
			optionsC = append(optionsC, &c)
		}
	}
	return optionsC
}

func dedupOptions(t *testing.T, options []*arbitrum_types.ConditionalOptions) []*arbitrum_types.ConditionalOptions {
	var result []*arbitrum_types.ConditionalOptions
	seenBefore := make(map[common.Hash]struct{})
	for _, opt := range options {
		data, err := json.Marshal(opt)
		Require(t, err)
		dataHash := crypto.Keccak256Hash(data)
		_, seen := seenBefore[dataHash]
		if !seen {
			result = append(result, opt)
			seenBefore[dataHash] = struct{}{}
		}
	}
	return result
}

func TestSendRawTransactionConditionalBasic(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	contractAddress1, simple1 := builder.L2.DeploySimple(t, auth)
	tx, err := simple1.Increment(&auth)
	Require(t, err, "failed to call Increment()")
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	contractAddress2, simple2 := builder.L2.DeploySimple(t, auth)
	tx, err = simple2.Increment(&auth)
	Require(t, err, "failed to call Increment()")
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	tx, err = simple2.Increment(&auth)
	Require(t, err, "failed to call Increment()")
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	currentRootHash1 := getStorageRootHash(t, builder.L2.ExecNode, contractAddress1)
	currentSlotValueMap1 := getStorageSlotValue(t, builder.L2.ExecNode, contractAddress1)
	currentRootHash2 := getStorageRootHash(t, builder.L2.ExecNode, contractAddress2)
	currentSlotValueMap2 := getStorageSlotValue(t, builder.L2.ExecNode, contractAddress2)

	rpcClient, err := builder.L2.ConsensusNode.Stack.Attach()
	Require(t, err)

	builder.L2Info.GenerateAccount("User2")

	testConditionalTxThatShouldSucceed(t, ctx, -1, builder.L2Info, rpcClient, nil)
	for i, options := range getEmptyOptions(contractAddress1) {
		testConditionalTxThatShouldSucceed(t, ctx, i, builder.L2Info, rpcClient, options)
	}

	block, err := builder.L1.Client.BlockByNumber(ctx, nil)
	Require(t, err)
	blockNumber := block.NumberU64()
	blockTime := block.Time()

	optionsA := getOptions(contractAddress1, currentRootHash1, currentSlotValueMap1)
	optionsB := getOptions(contractAddress2, currentRootHash2, currentSlotValueMap2)
	optionsAB := optionsProduct(optionsA, optionsB)
	options1 := dedupOptions(t, append(append(optionsAB, optionsA...), optionsB...))
	options1 = optionsDedupProduct(t, options1, getFulfillableBlockTimeLimits(t, blockNumber, blockTime))
	for i, options := range options1 {
		testConditionalTxThatShouldSucceed(t, ctx, i, builder.L2Info, rpcClient, options)
	}

	tx, err = simple1.Increment(&auth)
	Require(t, err, "failed to call Increment()")
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	tx, err = simple2.Increment(&auth)
	Require(t, err, "failed to call Increment()")
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	previousStorageRootHash1 := currentRootHash1
	currentRootHash1 = getStorageRootHash(t, builder.L2.ExecNode, contractAddress1)
	if bytes.Equal(previousStorageRootHash1.Bytes(), currentRootHash1.Bytes()) {
		Fatal(t, "storage root hash didn't change as expected")
	}
	currentSlotValueMap1 = getStorageSlotValue(t, builder.L2.ExecNode, contractAddress1)

	previousStorageRootHash2 := currentRootHash2
	currentRootHash2 = getStorageRootHash(t, builder.L2.ExecNode, contractAddress2)
	if bytes.Equal(previousStorageRootHash2.Bytes(), currentRootHash2.Bytes()) {
		Fatal(t, "storage root hash didn't change as expected")
	}
	currentSlotValueMap2 = getStorageSlotValue(t, builder.L2.ExecNode, contractAddress2)

	block, err = builder.L1.Client.BlockByNumber(ctx, nil)
	Require(t, err)
	blockNumber = block.NumberU64()
	blockTime = block.Time()

	optionsC := getOptions(contractAddress1, currentRootHash1, currentSlotValueMap1)
	optionsD := getOptions(contractAddress2, currentRootHash2, currentSlotValueMap2)
	optionsCD := optionsProduct(optionsC, optionsD)
	options2 := dedupOptions(t, append(append(optionsCD, optionsC...), optionsD...))
	options2 = optionsDedupProduct(t, options2, getFulfillableBlockTimeLimits(t, blockNumber, blockTime))
	for i, options := range options2 {
		testConditionalTxThatShouldSucceed(t, ctx, i, builder.L2Info, rpcClient, options)
	}
	for i, options := range options1 {
		testConditionalTxThatShouldFail(t, ctx, i, builder.L2Info, rpcClient, options, -32003)
	}
	block, err = builder.L1.Client.BlockByNumber(ctx, nil)
	Require(t, err)
	blockNumber = block.NumberU64()
	blockTime = block.Time()
	options3 := optionsDedupProduct(t, options2, getUnfulfillableBlockTimeLimits(t, blockNumber, blockTime))
	for i, options := range options3 {
		testConditionalTxThatShouldFail(t, ctx, i, builder.L2Info, rpcClient, options, -32003)
	}
	options4 := optionsDedupProduct(t, options2, options1)
	for i, options := range options4 {
		testConditionalTxThatShouldFail(t, ctx, i, builder.L2Info, rpcClient, options, -32003)
	}
}

func TestSendRawTransactionConditionalMultiRoutine(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	rpcClient, err := builder.L2.ConsensusNode.Stack.Attach()
	Require(t, err)

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	contractAddress, simple := builder.L2.DeploySimple(t, auth)

	simpleContract, err := abi.JSON(strings.NewReader(mocksgen.SimpleABI))
	Require(t, err)

	numTxes := 200
	expectedSuccesses := numTxes / 20
	var txes types.Transactions
	var options []*arbitrum_types.ConditionalOptions
	for i := 0; i < numTxes; i++ {
		account := fmt.Sprintf("User%v", i)
		builder.L2Info.GenerateAccount(account)
		tx := builder.L2Info.PrepareTx("Owner", account, builder.L2Info.TransferGas, big.NewInt(1e16), nil)
		err := builder.L2.Client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
	}
	for i := numTxes - 1; i >= 0; i-- {
		expected := i % expectedSuccesses
		data, err := simpleContract.Pack("logAndIncrement", big.NewInt(int64(expected)))
		Require(t, err)
		account := fmt.Sprintf("User%v", i)
		txes = append(txes, builder.L2Info.PrepareTxTo(account, &contractAddress, builder.L2Info.TransferGas, big.NewInt(0), data))
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
			Fatal(t, "test timeouted")
		}
	}
	cancelCtxWithTimeout()
	wg.Wait()
	bc := builder.L2.ExecNode.Backend.ArbInterface().BlockChain()
	genesis := bc.Config().ArbitrumChainParams.GenesisBlockNum

	var receipts types.Receipts
	header := bc.GetHeaderByNumber(genesis)
	for i := genesis + 1; header != nil; i++ {
		blockReceipts := bc.GetReceiptsByHash(header.Hash())
		if blockReceipts == nil {
			Fatal(t, "Failed to get block receipts, block number:", header.Number)
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
				Fatal(t, "Got invalid log, log.Expected:", parsed.Expected, "log.Have:", parsed.Have)
			} else {
				succeeded++
			}
		}
	}
	if succeeded != expectedSuccesses {
		Fatal(t, "Unexpected number of successful txes, want:", numTxes, "have:", succeeded)
	}
}

func TestSendRawTransactionConditionalPreCheck(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.execConfig.Sequencer.MaxBlockSpeed = 0
	builder.execConfig.TxPreChecker.Strictness = gethexec.TxPreCheckerStrictnessLikelyCompatible
	builder.execConfig.TxPreChecker.RequiredStateAge = 1
	builder.execConfig.TxPreChecker.RequiredStateMaxBlocks = 2
	cleanup := builder.Build(t)
	defer cleanup()

	rpcClient, err := builder.L2.ConsensusNode.Stack.Attach()
	Require(t, err)

	builder.L2Info.GenerateAccount("User2")

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	start := time.Now().Unix()
	contractAddress, simple := builder.L2.DeploySimple(t, auth)
	if time.Since(time.Unix(start, 0)) > 200*time.Millisecond {
		start++
		time.Sleep(time.Until(time.Unix(start, 0)))
	}
	tx, err := simple.Increment(&auth)
	Require(t, err, "failed to call Increment()")
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	currentRootHash := getStorageRootHash(t, builder.L2.ExecNode, contractAddress)
	options := &arbitrum_types.ConditionalOptions{
		KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{
			contractAddress: {RootHash: &currentRootHash},
		},
	}
	testConditionalTxThatShouldFail(t, ctx, 0, builder.L2Info, rpcClient, options, -32003)
	time.Sleep(time.Until(time.Unix(start+1, 0)))
	testConditionalTxThatShouldSucceed(t, ctx, 1, builder.L2Info, rpcClient, options)

	start = time.Now().Unix()
	if time.Since(time.Unix(start, 0)) > 200*time.Millisecond {
		start++
		time.Sleep(time.Until(time.Unix(start, 0)))
	}
	tx, err = simple.Increment(&auth)
	Require(t, err, "failed to call Increment()")
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	currentRootHash = getStorageRootHash(t, builder.L2.ExecNode, contractAddress)
	options = &arbitrum_types.ConditionalOptions{
		KnownAccounts: map[common.Address]arbitrum_types.RootHashOrSlots{
			contractAddress: {RootHash: &currentRootHash},
		},
	}
	testConditionalTxThatShouldFail(t, ctx, 2, builder.L2Info, rpcClient, options, -32003)
	tx = builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	Require(t, builder.L2.Client.SendTransaction(ctx, tx))
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	testConditionalTxThatShouldFail(t, ctx, 3, builder.L2Info, rpcClient, options, -32003)
	tx = builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	Require(t, builder.L2.Client.SendTransaction(ctx, tx))
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	testConditionalTxThatShouldSucceed(t, ctx, 4, builder.L2Info, rpcClient, options)
}
