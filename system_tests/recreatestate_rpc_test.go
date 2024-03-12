package arbtest

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util"
)

func makeSomeTransfers(t *testing.T, ctx context.Context, builder *NodeBuilder, txCount uint64) {
	var txs []*types.Transaction
	for i := uint64(0); i < txCount; i++ {
		tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, common.Big1, nil)
		txs = append(txs, tx)
		err := builder.L2.Client.SendTransaction(ctx, tx)
		Require(t, err)
	}
	for _, tx := range txs {
		_, err := builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
	}
}

func prepareNodeWithHistory(t *testing.T, ctx context.Context, execConfig *gethexec.Config, txCount uint64) (*NodeBuilder, func()) {
	t.Helper()
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.execConfig = execConfig
	cleanup := builder.Build(t)
	builder.L2Info.GenerateAccount("User2")
	makeSomeTransfers(t, ctx, builder, txCount)
	return builder, cleanup
}

func fillHeaderCache(t *testing.T, bc *core.BlockChain, from, to uint64) {
	t.Helper()
	for i := from; i <= to; i++ {
		header := bc.GetHeaderByNumber(i)
		if header == nil {
			Fatal(t, "internal test error - failed to get header while trying to fill headerCache, header:", i)
		}
	}
}

func fillBlockCache(t *testing.T, bc *core.BlockChain, from, to uint64) {
	t.Helper()
	for i := from; i <= to; i++ {
		block := bc.GetBlockByNumber(i)
		if block == nil {
			Fatal(t, "internal test error - failed to get block while trying to fill blockCache, block:", i)
		}
	}
}

func removeStatesFromDb(t *testing.T, bc *core.BlockChain, db ethdb.Database, from, to uint64) {
	t.Helper()
	for i := from; i <= to; i++ {
		header := bc.GetHeaderByNumber(i)
		if header == nil {
			Fatal(t, "failed to get last block header")
		}
		hash := header.Root
		err := db.Delete(hash.Bytes())
		Require(t, err)
	}
	for i := from; i <= to; i++ {
		header := bc.GetHeaderByNumber(i)
		_, err := bc.StateAt(header.Root)
		if err == nil {
			Fatal(t, "internal test error - failed to remove state from db")
		}
		expectedErr := &trie.MissingNodeError{}
		if !errors.As(err, &expectedErr) {
			Fatal(t, "internal test error - failed to remove state from db, err: ", err)
		}
	}
}

func TestRecreateStateForRPCNoDepthLimit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	execConfig := gethexec.ConfigDefaultTest()
	execConfig.RPC.MaxRecreateStateDepth = arbitrum.InfiniteMaxRecreateStateDepth
	execConfig.Sequencer.MaxBlockSpeed = 0
	execConfig.Sequencer.MaxTxDataSize = 150 // 1 test tx ~= 110
	execConfig.Caching.Archive = true
	execConfig.Caching.SnapshotCache = 0 // disable snapshots
	// disable trie/Database.cleans cache, so as states removed from ChainDb won't be cached there
	execConfig.Caching.TrieCleanCache = 0
	execConfig.Caching.MaxNumberOfBlocksToSkipStateSaving = 0
	execConfig.Caching.MaxAmountOfGasToSkipStateSaving = 0
	builder, cancelNode := prepareNodeWithHistory(t, ctx, execConfig, 32)
	defer cancelNode()
	execNode, l2client := builder.L2.ExecNode, builder.L2.Client
	bc := execNode.Backend.ArbInterface().BlockChain()
	db := execNode.Backend.ChainDb()

	lastBlock, err := l2client.BlockNumber(ctx)
	Require(t, err)
	middleBlock := lastBlock / 2

	expectedBalance, err := l2client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(lastBlock))
	Require(t, err)

	removeStatesFromDb(t, bc, db, middleBlock, lastBlock)

	balance, err := l2client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(lastBlock))
	Require(t, err)
	if balance.Cmp(expectedBalance) != 0 {
		Fatal(t, "unexpected balance result for last block, want: ", expectedBalance, " have: ", balance)
	}
}

func TestRecreateStateForRPCBigEnoughDepthLimit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	depthGasLimit := int64(256 * util.NormalizeL2GasForL1GasInitial(800_000, params.GWei))
	execConfig := gethexec.ConfigDefaultTest()
	execConfig.RPC.MaxRecreateStateDepth = depthGasLimit
	execConfig.Sequencer.MaxBlockSpeed = 0
	execConfig.Sequencer.MaxTxDataSize = 150 // 1 test tx ~= 110
	execConfig.Caching.Archive = true
	// disable trie/Database.cleans cache, so as states removed from ChainDb won't be cached there
	execConfig.Caching.TrieCleanCache = 0
	execConfig.Caching.MaxNumberOfBlocksToSkipStateSaving = 0
	execConfig.Caching.MaxAmountOfGasToSkipStateSaving = 0
	builder, cancelNode := prepareNodeWithHistory(t, ctx, execConfig, 32)
	defer cancelNode()
	execNode, l2client := builder.L2.ExecNode, builder.L2.Client
	bc := execNode.Backend.ArbInterface().BlockChain()
	db := execNode.Backend.ChainDb()

	lastBlock, err := l2client.BlockNumber(ctx)
	Require(t, err)
	middleBlock := lastBlock / 2

	expectedBalance, err := l2client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(lastBlock))
	Require(t, err)

	removeStatesFromDb(t, bc, db, middleBlock, lastBlock)

	balance, err := l2client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(lastBlock))
	Require(t, err)
	if balance.Cmp(expectedBalance) != 0 {
		Fatal(t, "unexpected balance result for last block, want: ", expectedBalance, " have: ", balance)
	}

}

func TestRecreateStateForRPCDepthLimitExceeded(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	execConfig := gethexec.ConfigDefaultTest()
	execConfig.RPC.MaxRecreateStateDepth = int64(200)
	execConfig.Sequencer.MaxBlockSpeed = 0
	execConfig.Sequencer.MaxTxDataSize = 150 // 1 test tx ~= 110
	execConfig.Caching.Archive = true
	// disable trie/Database.cleans cache, so as states removed from ChainDb won't be cached there
	execConfig.Caching.TrieCleanCache = 0
	execConfig.Caching.MaxNumberOfBlocksToSkipStateSaving = 0
	execConfig.Caching.MaxAmountOfGasToSkipStateSaving = 0
	builder, cancelNode := prepareNodeWithHistory(t, ctx, execConfig, 32)
	defer cancelNode()
	execNode, l2client := builder.L2.ExecNode, builder.L2.Client
	bc := execNode.Backend.ArbInterface().BlockChain()
	db := execNode.Backend.ChainDb()

	lastBlock, err := l2client.BlockNumber(ctx)
	Require(t, err)
	middleBlock := lastBlock / 2

	removeStatesFromDb(t, bc, db, middleBlock, lastBlock)

	_, err = l2client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(lastBlock))
	if err == nil {
		Fatal(t, "Didn't fail as expected")
	}
	if err.Error() != arbitrum.ErrDepthLimitExceeded.Error() {
		Fatal(t, "Failed with unexpected error:", err)
	}
}

func TestRecreateStateForRPCMissingBlockParent(t *testing.T) {
	// HeaderChain.headerCache size limit is currently core.headerCacheLimit = 512
	var headerCacheLimit uint64 = 512
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	execConfig := gethexec.ConfigDefaultTest()
	execConfig.RPC.MaxRecreateStateDepth = arbitrum.InfiniteMaxRecreateStateDepth
	execConfig.Sequencer.MaxBlockSpeed = 0
	execConfig.Sequencer.MaxTxDataSize = 150 // 1 test tx ~= 110
	execConfig.Caching.Archive = true
	// disable trie/Database.cleans cache, so as states removed from ChainDb won't be cached there
	execConfig.Caching.TrieCleanCache = 0
	execConfig.Caching.MaxNumberOfBlocksToSkipStateSaving = 0
	execConfig.Caching.MaxAmountOfGasToSkipStateSaving = 0
	builder, cancelNode := prepareNodeWithHistory(t, ctx, execConfig, headerCacheLimit+5)
	defer cancelNode()
	execNode, l2client := builder.L2.ExecNode, builder.L2.Client
	bc := execNode.Backend.ArbInterface().BlockChain()
	db := execNode.Backend.ChainDb()

	lastBlock, err := l2client.BlockNumber(ctx)
	Require(t, err)
	if lastBlock < headerCacheLimit+4 {
		Fatal(t, "Internal test error - not enough blocks produced during preparation, want:", headerCacheLimit, "have:", lastBlock)
	}

	removeStatesFromDb(t, bc, db, lastBlock-4, lastBlock)

	headerToRemove := lastBlock - 4
	hash := rawdb.ReadCanonicalHash(db, headerToRemove)
	rawdb.DeleteHeader(db, hash, headerToRemove)

	firstBlock := lastBlock - headerCacheLimit - 5
	fillHeaderCache(t, bc, firstBlock, firstBlock+headerCacheLimit)

	for i := lastBlock; i > lastBlock-3; i-- {
		_, err = l2client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(i))
		if err == nil {
			hash := rawdb.ReadCanonicalHash(db, i)
			Fatal(t, "Didn't fail to get balance at block:", i, " with hash:", hash, ", lastBlock:", lastBlock)
		}
		if !strings.Contains(err.Error(), "chain doesn't contain parent of block") {
			Fatal(t, "Failed with unexpected error: \"", err, "\", at block:", i, "lastBlock:", lastBlock)
		}
	}
}

func TestRecreateStateForRPCBeyondGenesis(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	execConfig := gethexec.ConfigDefaultTest()
	execConfig.RPC.MaxRecreateStateDepth = arbitrum.InfiniteMaxRecreateStateDepth
	execConfig.Sequencer.MaxBlockSpeed = 0
	execConfig.Sequencer.MaxTxDataSize = 150 // 1 test tx ~= 110
	execConfig.Caching.Archive = true
	// disable trie/Database.cleans cache, so as states removed from ChainDb won't be cached there
	execConfig.Caching.TrieCleanCache = 0
	execConfig.Caching.MaxNumberOfBlocksToSkipStateSaving = 0
	execConfig.Caching.MaxAmountOfGasToSkipStateSaving = 0
	builder, cancelNode := prepareNodeWithHistory(t, ctx, execConfig, 32)
	execNode, l2client := builder.L2.ExecNode, builder.L2.Client
	defer cancelNode()
	bc := execNode.Backend.ArbInterface().BlockChain()
	db := execNode.Backend.ChainDb()

	lastBlock, err := l2client.BlockNumber(ctx)
	Require(t, err)

	genesis := bc.Config().ArbitrumChainParams.GenesisBlockNum
	removeStatesFromDb(t, bc, db, genesis, lastBlock)

	_, err = l2client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(lastBlock))
	if err == nil {
		hash := rawdb.ReadCanonicalHash(db, lastBlock)
		Fatal(t, "Didn't fail to get balance at block:", lastBlock, " with hash:", hash, ", lastBlock:", lastBlock)
	}
	if !strings.Contains(err.Error(), "moved beyond genesis") {
		Fatal(t, "Failed with unexpected error: \"", err, "\", at block:", lastBlock, "lastBlock:", lastBlock)
	}
}

func TestRecreateStateForRPCBlockNotFoundWhileRecreating(t *testing.T) {
	// BlockChain.blockCache size limit is currently core.blockCacheLimit = 256
	var blockCacheLimit uint64 = 256
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	execConfig := gethexec.ConfigDefaultTest()
	execConfig.RPC.MaxRecreateStateDepth = arbitrum.InfiniteMaxRecreateStateDepth
	execConfig.Sequencer.MaxBlockSpeed = 0
	execConfig.Sequencer.MaxTxDataSize = 150 // 1 test tx ~= 110
	execConfig.Caching.Archive = true
	// disable trie/Database.cleans cache, so as states removed from ChainDb won't be cached there
	execConfig.Caching.TrieCleanCache = 0

	execConfig.Caching.MaxNumberOfBlocksToSkipStateSaving = 0
	execConfig.Caching.MaxAmountOfGasToSkipStateSaving = 0
	builder, cancelNode := prepareNodeWithHistory(t, ctx, execConfig, blockCacheLimit+4)
	execNode, l2client := builder.L2.ExecNode, builder.L2.Client
	defer cancelNode()
	bc := execNode.Backend.ArbInterface().BlockChain()
	db := execNode.Backend.ChainDb()

	lastBlock, err := l2client.BlockNumber(ctx)
	Require(t, err)
	if lastBlock < blockCacheLimit+4 {
		Fatal(t, "Internal test error - not enough blocks produced during preparation, want:", blockCacheLimit, "have:", lastBlock)
	}

	removeStatesFromDb(t, bc, db, lastBlock-4, lastBlock)

	blockBodyToRemove := lastBlock - 1
	hash := rawdb.ReadCanonicalHash(db, blockBodyToRemove)
	rawdb.DeleteBody(db, hash, blockBodyToRemove)

	firstBlock := lastBlock - blockCacheLimit - 4
	fillBlockCache(t, bc, firstBlock, firstBlock+blockCacheLimit)

	_, err = l2client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(lastBlock))
	if err == nil {
		hash := rawdb.ReadCanonicalHash(db, lastBlock)
		Fatal(t, "Didn't fail to get balance at block:", lastBlock, " with hash:", hash, ", lastBlock:", lastBlock)
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("block #%d not found", blockBodyToRemove)) {
		Fatal(t, "Failed with unexpected error: \"", err, "\", at block:", lastBlock, "lastBlock:", lastBlock)
	}
}

func testSkippingSavingStateAndRecreatingAfterRestart(t *testing.T, cacheConfig *gethexec.CachingConfig, txCount int) {
	maxRecreateStateDepth := int64(30 * 1000 * 1000)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx1, cancel1 := context.WithCancel(ctx)
	execConfig := gethexec.ConfigDefaultTest()
	execConfig.RPC.MaxRecreateStateDepth = maxRecreateStateDepth
	execConfig.Sequencer.MaxBlockSpeed = 0
	execConfig.Sequencer.MaxTxDataSize = 150 // 1 test tx ~= 110
	execConfig.Caching = *cacheConfig

	skipBlocks := execConfig.Caching.MaxNumberOfBlocksToSkipStateSaving
	skipGas := execConfig.Caching.MaxAmountOfGasToSkipStateSaving

	feedErrChan := make(chan error, 10)
	l2info, stack, chainDb, arbDb, blockchain := createL2BlockChain(t, nil, t.TempDir(), params.ArbitrumDevTestChainConfig(), &execConfig.Caching)

	Require(t, execConfig.Validate())
	execConfigFetcher := func() *gethexec.Config { return execConfig }
	execNode, err := gethexec.CreateExecutionNode(ctx1, stack, chainDb, blockchain, nil, execConfigFetcher)
	Require(t, err)

	parentChainID := big.NewInt(1337)
	node, err := arbnode.CreateNode(ctx1, stack, execNode, arbDb, NewFetcherFromConfig(arbnode.ConfigDefaultL2Test()), blockchain.Config(), nil, nil, nil, nil, nil, feedErrChan, parentChainID, nil)
	Require(t, err)
	err = node.TxStreamer.AddFakeInitMessage()
	Require(t, err)
	Require(t, node.Start(ctx1))
	client := ClientForStack(t, stack)

	StartWatchChanErr(t, ctx, feedErrChan, node)
	dataDir := node.Stack.DataDir()

	l2info.GenerateAccount("User2")
	var txs []*types.Transaction
	for i := 0; i < txCount; i++ {
		tx := l2info.PrepareTx("Owner", "User2", l2info.TransferGas, common.Big1, nil)
		txs = append(txs, tx)
		err := client.SendTransaction(ctx, tx)
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, client, tx)
		Require(t, err)
		if have, want := receipt.BlockNumber.Uint64(), uint64(i)+1; have != want {
			Fatal(t, "internal test error - tx got included in unexpected block number, have:", have, "want:", want)
		}
	}
	bc := execNode.Backend.ArbInterface().BlockChain()
	genesis := uint64(0)
	currentHeader := bc.CurrentBlock()
	if currentHeader == nil {
		Fatal(t, "missing current block")
	}
	lastBlock := currentHeader.Number.Uint64()
	if want := genesis + uint64(txCount); lastBlock < want {
		Fatal(t, "internal test error - not enough blocks produced during preparation, want:", want, "have:", lastBlock)
	}
	expectedBalance, err := client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(lastBlock))
	Require(t, err)

	node.StopAndWait()
	cancel1()
	t.Log("stopped first node")

	l2info, stack, chainDb, arbDb, blockchain = createL2BlockChain(t, l2info, dataDir, params.ArbitrumDevTestChainConfig(), &execConfig.Caching)

	execNode, err = gethexec.CreateExecutionNode(ctx1, stack, chainDb, blockchain, nil, execConfigFetcher)
	Require(t, err)

	node, err = arbnode.CreateNode(ctx, stack, execNode, arbDb, NewFetcherFromConfig(arbnode.ConfigDefaultL2Test()), blockchain.Config(), nil, node.DeployInfo, nil, nil, nil, feedErrChan, parentChainID, nil)
	Require(t, err)
	Require(t, node.Start(ctx))
	client = ClientForStack(t, stack)
	defer node.StopAndWait()
	bc = execNode.Backend.ArbInterface().BlockChain()
	gas := skipGas
	blocks := skipBlocks
	for i := genesis + 1; i <= genesis+uint64(txCount); i++ {
		block := bc.GetBlockByNumber(i)
		if block == nil {
			Fatal(t, "header not found for block number:", i)
			continue
		}
		gas += block.GasUsed()
		_, err := bc.StateAt(block.Root())
		blocks++
		if (skipBlocks == 0 && skipGas == 0) || (skipBlocks != 0 && blocks > skipBlocks) || (skipGas != 0 && gas > skipGas) {
			if err != nil {
				t.Log("blocks:", blocks, "skipBlocks:", skipBlocks, "gas:", gas, "skipGas:", skipGas)
			}
			Require(t, err, "state not found, root:", block.Root(), "blockNumber:", i, "blockHash", block.Hash(), "err:", err)
			gas = 0
			blocks = 0
		} else {
			if int(i) >= int(lastBlock)-int(cacheConfig.BlockCount) {
				// skipping nonexistence check - the state might have been saved on node shutdown
				continue
			}
			if err == nil {
				t.Log("blocks:", blocks, "skipBlocks:", skipBlocks, "gas:", gas, "skipGas:", skipGas)
				Fatal(t, "state shouldn't be available, root:", block.Root(), "blockNumber:", i, "blockHash", block.Hash())
			}
			expectedErr := &trie.MissingNodeError{}
			if !errors.As(err, &expectedErr) {
				Fatal(t, "getting state failed with unexpected error, root:", block.Root(), "blockNumber:", i, "blockHash:", block.Hash(), "err:", err)
			}
		}
	}
	for i := genesis + 1; i <= genesis+uint64(txCount); i += i % 10 {
		_, err = client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(i))
		if err != nil {
			t.Log("skipBlocks:", skipBlocks, "skipGas:", skipGas)
		}
		Require(t, err)
	}

	balance, err := client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(lastBlock))
	Require(t, err)
	if balance.Cmp(expectedBalance) != 0 {
		Fatal(t, "unexpected balance result for last block, want: ", expectedBalance, " have: ", balance)
	}
}

func TestSkippingSavingStateAndRecreatingAfterRestart(t *testing.T) {
	cacheConfig := gethexec.DefaultCachingConfig
	cacheConfig.Archive = true
	cacheConfig.SnapshotCache = 0 // disable snapshots
	cacheConfig.BlockAge = 0      // use only Caching.BlockCount to keep only last N blocks in dirties cache, no matter how new they are

	// test defaults
	testSkippingSavingStateAndRecreatingAfterRestart(t, &cacheConfig, 512)

	cacheConfig.MaxNumberOfBlocksToSkipStateSaving = 127
	cacheConfig.MaxAmountOfGasToSkipStateSaving = 0
	testSkippingSavingStateAndRecreatingAfterRestart(t, &cacheConfig, 512)

	cacheConfig.MaxNumberOfBlocksToSkipStateSaving = 0
	cacheConfig.MaxAmountOfGasToSkipStateSaving = 15 * 1000 * 1000
	testSkippingSavingStateAndRecreatingAfterRestart(t, &cacheConfig, 512)

	cacheConfig.MaxNumberOfBlocksToSkipStateSaving = 127
	cacheConfig.MaxAmountOfGasToSkipStateSaving = 15 * 1000 * 1000
	testSkippingSavingStateAndRecreatingAfterRestart(t, &cacheConfig, 512)

	// lower number of blocks in triegc below 100 blocks, to be able to check for nonexistence in testSkippingSavingStateAndRecreatingAfterRestart (it doesn't check last BlockCount blocks as some of them may be persisted on node shutdown)
	cacheConfig.BlockCount = 16

	testBlockGas := uint64(925000) // one test block ~ 925000 gas
	skipBlockValues := []uint64{0, 1, 2, 3, 5, 21, 51, 100, 101}
	var skipGasValues []uint64
	for _, i := range skipBlockValues {
		skipGasValues = append(skipGasValues, i*testBlockGas)
	}
	for _, skipGas := range skipGasValues {
		for _, skipBlocks := range skipBlockValues[:len(skipBlockValues)-2] {
			cacheConfig.MaxAmountOfGasToSkipStateSaving = skipGas
			cacheConfig.MaxNumberOfBlocksToSkipStateSaving = uint32(skipBlocks)
			testSkippingSavingStateAndRecreatingAfterRestart(t, &cacheConfig, 100)
		}
	}
}

func TestGettingStateForRPCFullNode(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	execConfig := gethexec.ConfigDefaultTest()
	execConfig.Caching.SnapshotCache = 0 // disable snapshots
	execConfig.Caching.BlockAge = 0      // use only Caching.BlockCount to keep only last N blocks in dirties cache, no matter how new they are
	execConfig.Sequencer.MaxBlockSpeed = 0
	execConfig.Sequencer.MaxTxDataSize = 150 // 1 test tx ~= 110
	builder, cancelNode := prepareNodeWithHistory(t, ctx, execConfig, 16)
	execNode, _ := builder.L2.ExecNode, builder.L2.Client
	defer cancelNode()
	bc := execNode.Backend.ArbInterface().BlockChain()
	api := execNode.Backend.APIBackend()

	header := bc.CurrentBlock()
	if header == nil {
		Fatal(t, "failed to get current block header")
	}
	state, _, err := api.StateAndHeaderByNumber(ctx, rpc.BlockNumber(header.Number.Uint64()))
	Require(t, err)
	addr := builder.L2Info.GetAddress("User2")
	exists := state.Exist(addr)
	err = state.Error()
	Require(t, err)
	if !exists {
		Fatal(t, "User2 address does not exist in the state")
	}
	// Get the state again to avoid caching
	state, _, err = api.StateAndHeaderByNumber(ctx, rpc.BlockNumber(header.Number.Uint64()))
	Require(t, err)

	blockCountRequiredToFlushDirties := builder.execConfig.Caching.BlockCount
	makeSomeTransfers(t, ctx, builder, blockCountRequiredToFlushDirties)

	exists = state.Exist(addr)
	err = state.Error()
	Require(t, err)
	if !exists {
		Fatal(t, "User2 address does not exist in the state")
	}
}

func TestGettingStateForRPCHybridArchiveNode(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	execConfig := gethexec.ConfigDefaultTest()
	execConfig.Caching.Archive = true
	execConfig.Caching.MaxNumberOfBlocksToSkipStateSaving = 128
	execConfig.Caching.BlockCount = 128
	execConfig.Caching.SnapshotCache = 0 // disable snapshots
	execConfig.Caching.BlockAge = 0      // use only Caching.BlockCount to keep only last N blocks in dirties cache, no matter how new they are
	execConfig.Sequencer.MaxBlockSpeed = 0
	execConfig.Sequencer.MaxTxDataSize = 150 // 1 test tx ~= 110
	builder, cancelNode := prepareNodeWithHistory(t, ctx, execConfig, 16)
	execNode, _ := builder.L2.ExecNode, builder.L2.Client
	defer cancelNode()
	bc := execNode.Backend.ArbInterface().BlockChain()
	api := execNode.Backend.APIBackend()

	header := bc.CurrentBlock()
	if header == nil {
		Fatal(t, "failed to get current block header")
	}
	state, _, err := api.StateAndHeaderByNumber(ctx, rpc.BlockNumber(header.Number.Uint64()))
	Require(t, err)
	addr := builder.L2Info.GetAddress("User2")
	exists := state.Exist(addr)
	err = state.Error()
	Require(t, err)
	if !exists {
		Fatal(t, "User2 address does not exist in the state")
	}
	// Get the state again to avoid caching
	state, _, err = api.StateAndHeaderByNumber(ctx, rpc.BlockNumber(header.Number.Uint64()))
	Require(t, err)

	blockCountRequiredToFlushDirties := builder.execConfig.Caching.BlockCount
	makeSomeTransfers(t, ctx, builder, blockCountRequiredToFlushDirties)

	exists = state.Exist(addr)
	err = state.Error()
	Require(t, err)
	if !exists {
		Fatal(t, "User2 address does not exist in the state")
	}
}

func TestStateAndHeaderForRecentBlock(t *testing.T) {
	threads := 32
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.execConfig.Caching.Archive = true
	builder.execConfig.RPC.MaxRecreateStateDepth = 0
	cleanup := builder.Build(t)
	defer cleanup()
	builder.L2Info.GenerateAccount("User2")

	errors := make(chan error, threads+1)
	senderDone := make(chan struct{})
	go func() {
		defer close(senderDone)
		for ctx.Err() == nil {
			tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, new(big.Int).Lsh(big.NewInt(1), 128), nil)
			err := builder.L2.Client.SendTransaction(ctx, tx)
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				errors <- err
				return
			}
			_, err = builder.L2.EnsureTxSucceeded(tx)
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				errors <- err
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	api := builder.L2.ExecNode.Backend.APIBackend()
	db := builder.L2.ExecNode.Backend.ChainDb()
	i := 1
	var mtx sync.RWMutex
	var wgCallers sync.WaitGroup
	for j := 0; j < threads && ctx.Err() == nil; j++ {
		wgCallers.Add(1)
		go func() {
			defer wgCallers.Done()
			mtx.RLock()
			blockNumber := i
			mtx.RUnlock()
			for blockNumber < 300 && ctx.Err() == nil {
				prefix := make([]byte, 8)
				binary.BigEndian.PutUint64(prefix, uint64(blockNumber))
				prefix = append([]byte("b"), prefix...)
				it := db.NewIterator(prefix, nil)
				defer it.Release()
				if it.Next() {
					key := it.Key()
					if len(key) != len(prefix)+common.HashLength {
						Fatal(t, "Wrong key length, have:", len(key), "want:", len(prefix)+common.HashLength)
					}
					blockHash := common.BytesToHash(key[len(prefix):])
					start := time.Now()
					for ctx.Err() == nil {
						_, _, err := api.StateAndHeaderByNumberOrHash(ctx, rpc.BlockNumberOrHash{BlockHash: &blockHash})
						if err == nil {
							mtx.Lock()
							if blockNumber == i {
								i++
							}
							mtx.Unlock()
							break
						}
						if ctx.Err() != nil {
							return
						}
						if !strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "missing trie node") {
							errors <- err
							return
						}
						if time.Since(start) > 5*time.Second {
							errors <- fmt.Errorf("timeout - failed to get state for more then 5 seconds, block: %d, err: %w", blockNumber, err)
							return
						}
					}
				}
				it.Release()
				mtx.RLock()
				blockNumber = i
				mtx.RUnlock()
			}
		}()
	}
	callersDone := make(chan struct{})
	go func() {
		wgCallers.Wait()
		close(callersDone)
	}()

	select {
	case <-callersDone:
		cancel()
	case <-senderDone:
		cancel()
	case err := <-errors:
		t.Error(err)
		cancel()
	}
	<-callersDone
	<-senderDone
	close(errors)
	for err := range errors {
		if err != nil {
			t.Error(err)
		}
	}
}
